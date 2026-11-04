package main

import (
	"fmt"
	"log/slog"
	"net"
)

func Serve(port string) error {
	// Open connection on specified port.
	listener, err := net.Listen("tcp", ":"+port)
	if err != nil {
		slog.Debug(fmt.Sprintf("Error starting server: %v", err))
		return err
	}
	slog.Debug(fmt.Sprintf("Listening on %s\n", listener.Addr().String()))
	// Close connection gracefully on application exit.
	defer func(listener net.Listener) {
		err := listener.Close()
		if err != nil {
			slog.Debug(fmt.Sprintf("Error closing listener: %v", err))
		}
	}(listener)

	return nil
}
