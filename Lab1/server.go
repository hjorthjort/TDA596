package main

import (
	"fmt"
	"log/slog"
	"net"
)

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

	return nil
}
