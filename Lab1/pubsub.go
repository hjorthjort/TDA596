package main

import (
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"strconv"
	"time"
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
	var message []byte
	if req.Body == nil {
		message = []byte{}
	} else {
		var err error
		message, err = io.ReadAll(req.Body)
		if err != nil {
			slog.Debug(fmt.Sprintf("Error reading request body: %v", err))
			writeErrorResponse(conn, 400)
			return
		}
	}
	if err := brokerState.PublishSafe(topic, message); err != nil {
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

	if !channel.mutex.TryAdd() {
		writeErrorResponse(conn, 409)
		return
	}
	// Keep exclusive ownership through response writing, including error paths.
	defer channel.mutex.Release()

	// In case we need to block here, release our semaphore.
	// Clear the connection deadline, so that we can wait for a message without timing out.
	if err = conn.SetDeadline(time.Time{}); err != nil {
		// Hard failure, not writing anything back.
		slog.Debug(fmt.Sprintf("Error removing deadline before waiting for message: %v", err))
		return
	}
	semaphore.Release()
	msg, timeOutErr := waitForMessage(channel)
	semaphore.Acquire()
	// Set deadline for us to write the response back to the client
	if err = conn.SetDeadline(time.Now().Add(timeout)); err != nil {
		// Hard failure, not writing anything back.
		slog.Debug(fmt.Sprintf("Error removing deadline: %v", err))
		return
	}
	if timeOutErr != nil {
		slog.Debug("Timeout waiting for event")
		conn.Write(CreateHttpResponseHeader(204, 0, ""))
		return
	}
	header := CreateHttpResponseHeader(200, len(msg.Body), "text/plain")
	// Insert the sequence header before the final blank line.
	header = append(header[:len(header)-2], []byte(fmt.Sprintf("X-Sequence-Number: %d\r\n\r\n", msg.Sequence))...)
	_, err1 := conn.Write(header)
	_, err2 := conn.Write(msg.Body)
	if err1 != nil || err2 != nil {
		slog.Debug(fmt.Sprintf("Error writing response: %v, %v", err1, err2))
	}
}

func waitForMessage(channel *SubscriptionChannel) (Message, error) {
	select {
	case <-time.After(timeout):
		return Message{}, fmt.Errorf("timeout waiting for event")
	case msg := <-channel.Queue:
		return msg, nil
	}
}
