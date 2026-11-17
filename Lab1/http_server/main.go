package main

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	maxConcurrentClients = 10
	rootDir              = "."
)

var allowedTypes = map[string]string{
	".html": "text/html",
	".txt":  "text/plain",
	".gif":  "image/gif",
	".jpeg": "image/jpeg",
	".jpg":  "image/jpeg",
	".css":  "text/css",
}

func main(){

	// Check if entered port-num in cmd-line 
	if len(os.Args) != 2 {
		fmt.Println("Please enter a port number")
		os.Exit(1)
	}
	
	// Get port-num from cmd-line
	port, err := strconv.Atoi(os.Args[1])
	if err != nil || port < 1 || port > 65535 {
		fmt.Println("Please provide a valide port number")	
		os.Exit(1)
	}

	// Establish TCP listener
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))

	if err != nil {
		fmt.Println("Failed to start server:", err)
		os.Exit(1)
	}

	defer listener.Close()
	
	fmt.Printf("HTTP server listening on port %d\n", port)

	// Semaphore for how many clients can be connected to the server at the same time 
	sem := make(chan struct{}, maxConcurrentClients)

	for {
		// Wait for a connection.
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Error accepting client connections:", err)
			continue
		}

		sem <- struct{}{}

		// Handle the connection in a new goroutine.
		// The loop then returns to accepting, so that
		// multiple connections may be served concurrently.
		go func(c net.Conn) {
			// Clean up 
			defer func() {
				// release the sem and free up slot 
				<- sem 
				conn.Close()
			}()

			handleClient(conn)
		}(conn)
	}
}

func handleClient(conn net.Conn){

	reader := bufio.NewReader(conn)
	
	// Parse the http req
	req, err := http.ReadRequest(reader)
	if err != nil {
		fmt.Print("Error parsing http request:", err)
		os.Exit(1)
	}
	defer req.Body.Close()
	
	// Handle requests that expect a 100 Continue response.
	if req.Header.Get("Expect") == "100-continue" {
		fmt.Fprint(conn, "HTTP/1.1 100 Continue\r\n\r\n")
	}

	// Handle only GET and POST methodes 
	switch req.Method {
	case http.MethodGet:
		handleGET(conn, req)			
	case http.MethodPost:
		handlePOST(conn, req)
	default:
		sendError(conn, http.StatusNotImplemented)	
	}
}

// getFilePath validates the requested path and returns its local path.
func getFilePath(requestPath string) (string, string, error) {
	// Reject invalid paths.
	if requestPath == "" ||
		!strings.HasPrefix(requestPath, "/") ||
		strings.Contains(requestPath, "\\") ||
		strings.ContainsRune(requestPath, '\x00') {
		return "", "", fmt.Errorf("invalid path")
	}

	// Convert URL path into a cleaned local path.
	relativePath := strings.TrimPrefix(requestPath, "/")
	cleanPath := filepath.Clean(filepath.FromSlash(relativePath))

	// Prevent directory traversal.
	if cleanPath == "." ||
		cleanPath == ".." ||
		filepath.IsAbs(cleanPath) ||
		strings.HasPrefix(cleanPath, ".."+string(os.PathSeparator)) {
		return "", "", fmt.Errorf("invalid path")
	}

	// Check file extension.
	ext := strings.ToLower(filepath.Ext(cleanPath))

	contentType, exists := allowedTypes[ext]
	if !exists {
		return "", "", fmt.Errorf("unsupported file extension")
	}

	filePath := filepath.Join(rootDir, cleanPath)

	return filePath, contentType, nil
}

// handleGET reads a locally stored file and sends it to the client.
func handleGET(conn net.Conn, req *http.Request) {
	filePath, contentType, err := getFilePath(req.URL.Path)
	if err != nil {
		sendError(conn, http.StatusBadRequest)
		return
	}

	// Open file.
	file, err := os.Open(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			sendError(conn, http.StatusNotFound)
		} else {
			sendError(conn, http.StatusInternalServerError)
		}
		return
	}
	defer file.Close()

	// Read complete file.
	data, err := io.ReadAll(file)
	if err != nil {
		sendError(conn, http.StatusInternalServerError)
		return
	}

	// Send HTTP response.
	sendResponse(conn, http.StatusOK, contentType, data)
}

// handlePOST stores the request body in a local file.
func handlePOST(conn net.Conn, req *http.Request) {
	filePath, contentType, err := getFilePath(req.URL.Path)
	if err != nil {
		sendError(conn, http.StatusBadRequest)
		return
	}

	// Read complete request body.
	data, err := io.ReadAll(req.Body)
	if err != nil {
		sendError(conn, http.StatusBadRequest)
		return
	}

	// Make sure the parent directory exists.
	err = os.MkdirAll(filepath.Dir(filePath), 0755)
	if err != nil {
		sendError(conn, http.StatusInternalServerError)
		return
	}

	// Create a temporary file in the same directory.
	// This ensures that renaming is atomic.
	tempFile, err := os.CreateTemp(filepath.Dir(filePath), ".tmp-*")
	if err != nil {
		sendError(conn, http.StatusInternalServerError)
		return
	}

	tempPath := tempFile.Name()

	// Remove temporary file if something goes wrong.
	defer os.Remove(tempPath)

	// Write complete POST body to temporary file.
	_, err = tempFile.Write(data)
	if err != nil {
		tempFile.Close()
		sendError(conn, http.StatusInternalServerError)
		return
	}

	// Ensure data is written before replacing the original file.
	err = tempFile.Sync()
	if err != nil {
		tempFile.Close()
		sendError(conn, http.StatusInternalServerError)
		return
	}

	err = tempFile.Close()
	if err != nil {
		sendError(conn, http.StatusInternalServerError)
		return
	}

	// Atomically replace the destination file.
	err = os.Rename(tempPath, filePath)
	if err != nil {
		sendError(conn, http.StatusInternalServerError)
		return
	}

	// Send successful response.
	sendResponse(conn, http.StatusOK, contentType, []byte("File stored successfully\n"))
}

// sendResponse constructs and sends an HTTP response.
func sendResponse(conn net.Conn, status int, contentType string, body []byte) {
	statusText := http.StatusText(status)

	fmt.Fprintf(conn,
		"HTTP/1.1 %d %s\r\n"+
			"Content-Type: %s\r\n"+
			"Content-Length: %d\r\n"+
			"Connection: close\r\n"+
			"\r\n",
		status,
		statusText,
		contentType,
		len(body),
	)

	conn.Write(body)
}

// sendError sends a well-formed HTTP error response.
func sendError(conn net.Conn, status int) {
	statusText := http.StatusText(status)
	body := []byte(fmt.Sprintf("%d %s\n", status, statusText))

	sendResponse(conn, status, "text/plain", body)
}

