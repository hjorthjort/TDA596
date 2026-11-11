package main

import (
	"bufio"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var semaphore, _ = CreateSemaphore(10) // Limit to 10 concurrent connections

var wwwDir string    // Default document root
var rootDir *os.Root // Traversal resistant path

func Serve(port string) error {
	var err error
	wwwDir, err = os.Getwd()
	if err != nil {
		slog.Debug(fmt.Sprintf("Error getting current working directory: %v", err))
		return err
	}
	rootDir, err = os.OpenRoot(wwwDir + "/public")
	if err != nil {
		slog.Debug(fmt.Sprintf("Error opening root directory: %v", err))
		return err
	}
	defer rootDir.Close()

	listener, err := getListener(port)
	if err != nil {
		return err
	}
	// Close connection gracefully on application exit.
	defer func(listener net.Listener) {
		err := listener.Close()
		if err != nil {
			slog.Debug(fmt.Sprintf("Error closing listener: %v", err))
		}
	}(listener)

	// Lightweight *synchronous* loop to accept connections and handle them *asynchronously*.
	for {
		// Only spawn a child process if there are available slots. Always
		// release the semaphore after handling the request.
		semaphore.Acquire()
		conn, err := listener.Accept()
		if err != nil {
			semaphore.Release() // Release the semaphore if we failed to accept a connection.
			slog.Debug(fmt.Sprintf("Error accepting connection: %v", err))
			if errors.Is(net.ErrClosed, err) {
				// Listener closed, exit gracefully.
				return nil
			}
			continue
		} // Set a 30 second's timeout for the connection.
		if err = conn.SetDeadline(time.Now().Add(30 * time.Second)); err != nil {
			return err
		}
		go func() {
			defer semaphore.Release() // panic-resistant cleanup
			asyncHandleRequest(conn)
		}()
	}
}

func getListener(port string) (net.Listener, error) {
	// Open connection on specified port.
	listener, err := net.Listen("tcp", ":"+port)
	if err != nil {
		slog.Debug(fmt.Sprintf("Error getting listener: %v", err))
		return nil, err
	}
	slog.Debug(fmt.Sprintf("Listening on %s\n", listener.Addr().String()))
	return listener, nil
}

// The main request handling loop.
// reading and closing the connection.
// Acquires and releases locks for each request.
// Validates the request.
// Performs the request and writes the response to the connection.
func asyncHandleRequest(conn net.Conn) {
	defer conn.Close()
	reader := bufio.NewReader(conn)
	req, err := http.ReadRequest(reader)
	if err != nil {
		/*
			An invalid request from the client should be answered with an
			appropriate error code, i.e. "Bad Request" (400) ...
		*/
		slog.Debug(fmt.Sprintf("Error reading request: %v", err))
		writeErrorResponse(conn, 400)
		return
	}
	slog.Debug(fmt.Sprintf("Received request: %s %s %s", req.Method,
		req.URL.Path, req.Proto))

	// We do this check before checking file type since 501 for unimplemented
	// methods should take priority over 400 for unsupported file types.
	if req.Method != "GET" && req.Method != "POST" {
		/*
			if request method is not GET or POST, return 501 Not Implemented
		*/
		slog.Debug(fmt.Sprintf("Unsupported method: %s", req.Method))
		writeErrorResponse(conn, 501)
		return
	}
	if !isSupportedFileType(req.URL.Path) {
		/*
			if request for file is not supported file type, return 400 Bad
			Request
		*/
		slog.Debug(fmt.Sprintf("Unsupported file type: %s", req.URL.Path))
		writeErrorResponse(conn, 400)
		return
	}
	switch req.Method {
	case "GET":
		handleGetRequest(conn, req)
	case "POST":
		handlePostRequest(conn, req)
	}
}

func handleGetRequest(conn net.Conn, req *http.Request) {
	file, err := rootDir.Open(strings.TrimPrefix(req.URL.Path, "/"))
	if err != nil {
		slog.Debug(fmt.Sprintf("Error opening file: %v", err))
		slog.Debug(fmt.Sprintf("Full path %s", wwwDir+req.URL.Path))
		writeErrorResponse(conn, 404)
		return
	}
	defer file.Close()
	fileInfo, err := file.Stat()
	if err != nil {
		slog.Debug(fmt.Sprintf("Error getting file info: %v", err))
		writeErrorResponse(conn, 404)
		return
	}
	if fileInfo.IsDir() {
		slog.Debug(fmt.Sprintf("Requested path is a directory: %s", req.URL.Path))
		writeErrorResponse(conn, 404)
		return
	}

	contentType := supportedFileTypes[filepath.Ext(fileInfo.Name())]
	if _, err := conn.Write(createHttpResponseHeader(200, fileInfo.Size(), contentType)); err != nil {
		slog.Debug(fmt.Sprintf("Error writing response header: %v", err))
		return
	}
	if _, err := io.Copy(conn, file); err != nil {
		slog.Debug(fmt.Sprintf("Error reading file or writing response body: %v",
			err))
	}
}

func handlePostRequest(conn net.Conn, req *http.Request) {
	// POST writes to a new temporary file, and on success, it renames the
	// temporary file to the requested file name. If the requested file already
	// exists, it should be overwritten. On Unix this is atomic.
	tmpFile, err := createTmpFile()
	if err != nil {
		slog.Debug(fmt.Sprintf("Error creating tmp file: %v", err))
		writeErrorResponse(conn, 500)
		return
	}
	defer tmpFile.Close()
	defer rootDir.Remove(filepath.Base(tmpFile.Name())) // Clean up the temporary file if necessary.
	_, err = io.Copy(tmpFile, req.Body)
	if err != nil {
		slog.Debug(fmt.Sprintf("Error writing to tmp file: %v", err))
		writeErrorResponse(conn, 500)
		return
	}
	path := strings.TrimPrefix(req.URL.Path, "/")
	if err := rootDir.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		slog.Debug(fmt.Sprintf("Error creating parent directories: %v", err))
		writeErrorResponse(conn, 500)
		return
	}
	// Atomically rename the temporary file to the requested file name.
	err = rootDir.Rename(filepath.Base(tmpFile.Name()), path)
	if err != nil {
		slog.Debug(fmt.Sprintf("Error renaming tmp file: %v", err))
		writeErrorResponse(conn, 500)
		return
	}
	// The upload succeeded. The response has no body.
	if _, err := conn.Write(createHttpResponseHeader(200, 0, "")); err != nil {
		slog.Debug(fmt.Sprintf("Error writing response header: %v", err))
	}
}

func createHttpResponseHeader(statusCode int, contentLength int64, contentType string) []byte {
	header := fmt.Sprintf(
		"HTTP/1.1 %d %s\r\n"+
			"Date: %s\r\n"+
			"Content-Length: %d\r\n"+
			"Connection: close\r\n",
		statusCode,
		http.StatusText(statusCode),
		time.Now().UTC().Format(http.TimeFormat),
		contentLength,
	)
	if contentType != "" {
		header += "Content-Type: " + contentType + "\r\n"
	}
	// A blank line marks the end of the headers.
	return []byte(header + "\r\n")
}

func writeErrorResponse(conn net.Conn, statusCode int) {
	slog.Debug(fmt.Sprintf("Error response: %d", statusCode))
	if _, err := conn.Write(createHttpResponseHeader(statusCode, 0, "")); err != nil {
		slog.Debug(fmt.Sprintf("Error writing %d response: %v", statusCode, err))
	}
}

func createTmpFile() (*os.File, error) {
	// rand is "crypto/rand"
	// rand uses 128 bits of randomness. Collisions are essentially impossible.
	// A collision indicates and active attack, and it would be acceptable to
	// fail the request and return a 500 Internal Server Error in that case.
	name := ".upload-" + rand.Text() + ".tmp"
	tmp, err := rootDir.OpenFile(
		name,
		// write only | create new file | fail if file exists
		os.O_WRONLY|os.O_CREATE|os.O_EXCL,
		// chmod 600, is owner read/write only: -rw-------
		0o600,
	)
	return tmp, err
}
