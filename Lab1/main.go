package main

import (
	"log/slog"
	"os"
	"runtime"
)

func main() {
	// Set up simple debug logging.
	if os.Getenv("LOG_LEVEL") == "debug" {
		slog.SetLogLoggerLevel(slog.LevelDebug)
	}
	if runtime.GOOS != "linux" && runtime.GOOS != "darwin" {
		slog.Debug("Warning: This server is designed to run on Unix.")
		os.Exit(19)
	}
	// Accept port number from command line arguments
	port := os.Args[1]
	Serve(port)
}
