package main

import (
	"bufio"
	"fmt"
	"log/slog"
	"net"
	"net/http"
)

var semaphore, _ = CreateSemaphore(10) // Limit to 10 concurrent connections

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
