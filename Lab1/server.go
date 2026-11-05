package main

import (
	"fmt"
	"log/slog"
	"net"
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

}
