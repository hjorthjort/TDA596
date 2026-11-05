package main

import (
	"log/slog"
	"os"
)

func main() {
	// Set up simple debug logging.
	if os.Getenv("LOG_LEVEL") == "debug" {
		slog.SetLogLoggerLevel(slog.LevelDebug)
	}
	// Accept port number from command line arguments
	port := os.Args[1]
	Serve(port)
}
