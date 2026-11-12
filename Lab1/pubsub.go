package main

import (
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"strconv"
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
	topic := req.URL.Query().Get("topic")
	if topic == "" {
		slog.Debug("Missing topic parameter in subscribe request")
		writeErrorResponse(conn, 400)
		return
	}
	subscriptionId := brokerState.AddSubscriberSafe(topic)
	body := fmt.Sprintf("%d", subscriptionId)
	header := CreateHttpResponseHeader(201, len(body), "text/plain")
	_, err1 := conn.Write(header)
	_, err2 := conn.Write([]byte(body))
	if err1 != nil || err2 != nil {
		slog.Debug(fmt.Sprintf("Error writing response: %v, %v", err1, err2))
	}
	slog.Debug(brokerState.BrokerStateToString())
}

func handleUnsubscribe(conn net.Conn, req *http.Request) {
	slog.Debug(fmt.Sprintf("Handling unsubscribe request: %s %s %s", req.Method,
		req.URL.Path, req.Proto))
	topic := req.URL.Query().Get("topic")
	subscriptionId, err := strconv.Atoi(req.URL.Query().Get("id"))
	if err != nil || subscriptionId < 0 || topic == "" {
		slog.Debug("Missing or invalid topic or ID parameter in unsubscribe request")
		writeErrorResponse(conn, 400)
		return
	}
	err = brokerState.UnsubscribeSafe(topic, subscriptionId)
	if err != nil {
		slog.Debug(fmt.Sprintf("Error unsubscribing: %v", err))
		writeErrorResponse(conn, 404)
		return
	}
	conn.Write(CreateHttpResponseHeader(200, 0, ""))
}

func handlePublish(conn net.Conn, req *http.Request) {
	topic := req.URL.Query().Get("topic")
	if topic == "" {
		slog.Debug("Missing topic parameter in publish request")
		writeErrorResponse(conn, 400)
		return
	}
	// Read request body
	message, err := io.ReadAll(req.Body)
	if err != nil {
		message = []byte{} // Empty body = empty message. No reason not to allow this.
	}
	err = brokerState.PublishSafe(topic, message)
	if err != nil {
		// This is not an error on the publisher's part, so no error response.
		// It is just interesting to log.
		slog.Debug(fmt.Sprintf("No subscribers for topic %s: %v", topic, err))
	}
	if _, err := conn.Write(CreateHttpResponseHeader(200, 0, "")); err != nil {
		slog.Debug(fmt.Sprintf("Error writing response: %v", err))
	}
}

func handleEvents(conn net.Conn, req *http.Request) {
	slog.Debug(fmt.Sprintf("Handling events request: %s %s %s", req.Method,
		req.URL.Path, req.Proto))
	topic := req.URL.Query().Get("topic")
	id, err := strconv.Atoi(req.URL.Query().Get("id"))
	if err != nil || id < 0 || topic == "" {
		slog.Debug("Missing or invalid topic or ID parameter in events request")
		writeErrorResponse(conn, 400)
		return
	}
	channel, err := brokerState.EventSafe(topic, int(id))
	if err != nil {
		slog.Debug(fmt.Sprintf("Subscriber missing: %v", err))
		writeErrorResponse(conn, 404)
		return
	}

	// In case we need to block here, release our semaphore.
	semaphore.Release()
	msg := <-channel
	// Don't proceed until we have acquired the semaphore again.
	semaphore.Acquire()
	header := CreateHttpResponseHeader(200, len(msg), "text/plain")
	_, err1 := conn.Write(header)
	_, err2 := conn.Write(msg)
	if err1 != nil || err2 != nil {
		slog.Debug(fmt.Sprintf("Error writing response: %v, %v", err1, err2))
	}
}
