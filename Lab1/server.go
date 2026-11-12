package main

import (
	"bufio"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
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
	if _, exists := supportedPubSubPaths[req.URL.Path]; exists {
		// Is it pub/sub?
		handleWithPubSubBroker(conn, req)
		return
	} else if !isSupportedFileType(req.URL.Path) {
		// Is it a request on a file
		/*
			if request for file is not supported file type, return 400 Bad
			Request
		*/
		/*
			Or it is not a valid pub/sub path.
		*/
		slog.Debug(fmt.Sprintf("Method: %s - Unsupported path: %s",
			req.Method, req.URL.Path))
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

func CreateHttpResponseHeader(statusCode int, contentLength int,
	contentType string) []byte {
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
	if _, err := conn.Write(CreateHttpResponseHeader(statusCode, 0, "")); err != nil {
		slog.Debug(fmt.Sprintf("Error writing %d response: %v", statusCode, err))
	}
}
