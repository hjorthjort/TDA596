package main

import (
	"crypto/rand"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

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
	if _, err := conn.Write(CreateHttpResponseHeader(200, int(fileInfo.Size()),
		contentType)); err != nil {
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
	// TODO(hj): Make a general function for writing response and then
	// debugging. Can correctly set conentLength too.
	if _, err := conn.Write(CreateHttpResponseHeader(200, 0, "")); err != nil {
		slog.Debug(fmt.Sprintf("Error writing response header: %v", err))
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
