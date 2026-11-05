package main

var supportedMethods = map[string]struct{}{
	"GET":  {},
	"HEAD": {},
}
var errorResponses = map[int][]byte{
	400: []byte("HTTP/1.1 400 Bad Request\r\nContent-Length: 0\r\nConnection: close\r\n\r\n"),
	404: []byte("HTTP/1.1 404 Not Found\r\nContent-Length: 0\r\nConnection: close\r\n\r\n"),
	501: []byte("HTTP/1.1 501 Not Implemented\r\nContent-Length: 0\r\nConnection: close\r\n\r\n"),
}

var supportedFileTypes = map[string]struct{}{
	".html": {},
	".txt":  {},
	".gif":  {},
	".jpeg": {},
	".jpg":  {},
	".css":  {},
}

func isSupportedFileType(path string) bool {
	for ext := range supportedFileTypes {
		if len(path) >= len(ext) && path[len(path)-len(ext):] == ext {
			return true
		}
	}
	return false
}
