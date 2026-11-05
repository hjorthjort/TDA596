package main

import (
	"bufio"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
)

var semaphore, _ = CreateSemaphore(10) // Limit to 10 concurrent connections

var wwwDir, _ = os.Getwd()                       // Default document root
var rootDir, _ = os.OpenRoot(wwwDir + "/public") // Traversal resistant path

func Serve(port string) {
	listener, err := getListener(port)
	if err != nil {
		return
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
		con, err := listener.Accept()
		if err != nil {
			slog.Debug(fmt.Sprintf("Error accepting connection: %v", err))
			continue
		}
		go asyncHandlerequest(con)
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

func asyncHandlerequest(conn net.Conn) {
	semaphore.Acquire()
	defer semaphore.Release()
	reader := bufio.NewReader(conn)
	req, err := http.ReadRequest(reader)
	if err != nil {
		/*
			An invalid request from the client should be answered with an
			appropriate error code, i.e. "Bad Request" (400) ...
		*/
		writeErrorResponse(conn, 400)
		return
	}
	slog.Debug(fmt.Sprintf("Received request: %s %s %s", req.Method,
		req.URL.Path, req.Proto))

	if _, ok := supportedMethods[req.Method]; !ok {
		/*
			... or "Not Implemented" (501) for valid HTTP methods other than GET
			(or POST).
		*/
		writeErrorResponse(conn, 501)
		return
	}
	if !isSupportedFileType(req.URL.Path) {
		/*
			if request for file is not supported file type, return 400 Bad
			Request
		*/
		writeErrorResponse(conn, 400)
		return
	}
	switch req.Method {
	case "GET":
		handleGetRequest(conn, req)
	default:
		writeErrorResponse(conn, 501)
	}
}

func handleGetRequest(conn net.Conn, req *http.Request) {
	file, err := rootDir.Open(req.URL.Path)
	if err != nil {
		writeErrorResponse(conn, 404)
		return
	}
	defer file.Close()
	fileInfo, err := file.Stat()
	if err != nil {
		writeErrorResponse(conn, 404)
		return
	}

	if _, err := conn.Write(createHttpResponseHeader(200, fileInfo)); err != nil {
		slog.Debug(fmt.Sprintf("Error writing response header: %v", err))
	}
	// TODO(hj): Buffer the file reading/writing?
	fileContent := make([]byte, fileInfo.Size())
	_, err = file.Read(fileContent)
	if err != nil {
		slog.Debug(fmt.Sprintf("Error reading file content: %v", err))
	}
	if _, err := conn.Write(fileContent); err != nil {
		slog.Debug(fmt.Sprintf("Error writing response body: %v", err))
	}
}

func createHttpResponseHeader(statusCode int, fileInfo os.FileInfo) []byte {
	header := fmt.Sprintf("HTTP/1.1 %d %s\r\nContent-Length: %d\r\nContent-Type: %s\r\nConnection: close\r\n\r\n",
		statusCode,
		http.StatusText(statusCode),
		fileInfo.Size(),
		http.DetectContentType(make([]byte, 512)))
	return []byte(header)
}

func writeErrorResponse(conn net.Conn, statusCode int) {
	slog.Debug(fmt.Sprintf("Error response: %d", statusCode))
	if response, ok := errorResponses[statusCode]; ok {
		if _, err := conn.Write(response); err != nil {
			slog.Debug(fmt.Sprintf("Error writing %d response: %v", statusCode,
				err))
		}
	}
}
