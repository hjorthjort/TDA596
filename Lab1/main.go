package main

import (
	"fmt"
	"log/slog"
	"os"
	"runtime"
)

var brokerState = CreateBrokerState() // Global broker state for pub/sub
var serverState *ServerState          // Global server state for HTTP server

func main() {
	var err error
	serverState, err = CreateServerState()
	if err != nil {
		slog.Debug(fmt.Sprintf("Error creating server state: %v", err))
		os.Exit(1)
	}
	defer serverState.Cleanup()
	// Set up simple debug logging.
	if os.Getenv("LOG_LEVEL") == "debug" {
		slog.SetLogLoggerLevel(slog.LevelDebug)
	}
	if runtime.GOOS != "linux" && runtime.GOOS != "darwin" {
		slog.Debug("Warning: This server is designed to run on Unix.")
		os.Exit(19)
	}
	// Accept port number from command line arguments
	if len(os.Args) != 2 {
		slog.Debug("Usage: ./server <port>")
		os.Exit(1)
	}
	port := os.Args[1]
	if err := Serve(port); err != nil {
		slog.Debug(fmt.Sprintf("Error starting server: %v", err))
		os.Exit(1)
	}
}
