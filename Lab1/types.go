package main

type Method = string

var supportedFileTypes = map[string]string{
	".html": "text/html",
	".txt":  "text/plain",
	".gif":  "image/gif",
	".jpeg": "image/jpeg",
	".jpg":  "image/jpeg",
	".css":  "text/css",
}

var supportedPubSubPaths = map[string]Method{
	"/subscribe":   "POST",
	"/unsubscribe": "POST",
	"/publish":     "POST",
	"/events":      "GET",
}

func isSupportedFileType(path string) bool {
	for ext := range supportedFileTypes {
		if len(path) >= len(ext) && path[len(path)-len(ext):] == ext {
			return true
		}
	}
	return false
}

var isSupportedPubSubPath = func(path string, method Method) bool {
	if expectedMethod, ok := supportedPubSubPaths[path]; ok {
		return expectedMethod == method
	}
	return false
}

func isSupportedPath(path string, method Method) bool {
	return isSupportedFileType(path) || isSupportedPubSubPath(path, method)
}

type Message []byte
type SubscriptionChannel = chan Message
type BrokerState struct {
	mutex  Semaphore
	queues map[string][]SubscriptionChannel
}
