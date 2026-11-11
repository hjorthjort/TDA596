package main

import (
	"fmt"
	"log/slog"
	"net"
	"net/http"
)

func handleWithPubSubBroker(conn net.Conn, req *http.Request) {
	method, supported := supportedPubSubPaths[req.URL.Path]
	if supported && method != req.Method {
		slog.Debug(fmt.Sprintf("Incorrect method for path: %s, expected: %s, got: %s", req.URL.Path, method, req.Method))
		writeErrorResponse(conn, 501)
		return
	}
	switch req.URL.Path {
	case "/subscribe":
		handleSubscribe(conn, req)
	case "/unsubscribe":
		handleUnsubscribe(conn, req)
	case "/publish":
		handlePublish(conn, req)
	case "/events":
		handleEvents(conn, req)
	default:
		slog.Debug(fmt.Sprintf("Unsupported pub/sub path: %s", req.URL.Path))
		writeErrorResponse(conn, 400)
	}
}

func handleSubscribe(conn net.Conn, req *http.Request) {
	slog.Debug(fmt.Sprintf("Handling subscribe request: %s %s %s", req.Method,
		req.URL.Path, req.Proto))
}

func handleUnsubscribe(conn net.Conn, req *http.Request) {
	slog.Debug(fmt.Sprintf("Handling unsubscribe request: %s %s %s", req.Method,
		req.URL.Path, req.Proto))
}

func handlePublish(conn net.Conn, req *http.Request) {
	slog.Debug(fmt.Sprintf("Handling publish request: %s %s %s", req.Method,
		req.URL.Path, req.Proto))
}

func handleEvents(conn net.Conn, req *http.Request) {
	slog.Debug(fmt.Sprintf("Handling events request: %s %s %s", req.Method,
		req.URL.Path, req.Proto))
}
