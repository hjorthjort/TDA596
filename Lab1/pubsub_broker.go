package main

import (
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"strconv"
)

var brokerState, _ = CreateBrokerState() // Global broker state for pub/sub

// Debug printing
// TODO(hj): Delete
func (b *BrokerState) brokerStateToString() string {
	b.mutex.Acquire()
	defer b.mutex.Release()

	result := "Broker State:\n"
	for topic, subscribers := range b.queues {
		result += fmt.Sprintf("Topic: %s, Subscribers: %d\n", topic, len(subscribers))
	}
	return result
}

func CreateBrokerState() (BrokerState, error) {
	sem, err := CreateSemaphore(1)
	if err != nil {
		return BrokerState{}, err
	}
	return BrokerState{mutex: sem,
		queues: make(map[string][]SubscriptionChannel)}, nil
}

// CreateSubscriptionChannel Each subscription buffers up to 100 messages.
// TODO(hj): Make this larger. Small for now to test dropping messages.
func CreateSubscriptionChannel() SubscriptionChannel {
	return make(SubscriptionChannel, 3)
}

// AddSubscriberSafe adds a new subscriber to the topic in a thread-safe manner.
// TODO(hj): Guard from becoming too large, instead of failing with OOM.
// TODO(hj): Improvement: Random subscripber IDs, maybe BigInt or strings to of
// at least 128 byte randomness. For now, just use the index in the slice.
func (b *BrokerState) AddSubscriberSafe(topic string) (subscriptionId int) {
	b.mutex.Acquire()
	defer b.mutex.Release()

	if _, exists := b.queues[topic]; !exists {
		b.queues[topic] = make([]SubscriptionChannel, 0)
	}
	subscriptionId = len(b.queues[topic])
	b.queues[topic] = append(b.queues[topic], CreateSubscriptionChannel())
	return subscriptionId
}

// Implementation notes: every subscriber only receives the events that
// happen while they are subscribed.
// If the publisher's work is to put it in the queue of every subscriber that
// becomes O(N). That is fine for the initial implementation.
// An optimization is that the publisher puts the messages for each topic in a
// single list, and the subscriber's queue is a pointer to the last message they have seen.
// When there are zero pointers below a point in the queue,
// we pop that message off the queue.
func (b *BrokerState) PublishSafe(topic string, message Message) error {
	b.mutex.Acquire()
	defer b.mutex.Release()

	if subscribers, exists := b.queues[topic]; !exists {
		return fmt.Errorf("no subscribers for topic: %s", topic)
	} else {
		for _, subscriber := range subscribers {
			select {
			case subscriber <- message:
				// Message sent successfully
			default:
				// Subscriber's channel is full, drop the message for this subscriber
				slog.Debug(fmt.Sprintf("Dropping message for subscriber on topic: %s", topic))
			}
		}
	}
	return nil
}

// EventSafe assumes subscriberId is an int >= 0.
// Returns error only if the topic and/or subscriber is valid but missing.
func (b *BrokerState) EventSafe(topic string, subscriptionId int) (SubscriptionChannel, error) {
	if subscribers, exists := b.queues[topic]; !exists {
		return nil, fmt.Errorf("no subscribers for topic: %s", topic)
	} else {
		if subscriptionId >= len(subscribers) {
			return nil, fmt.Errorf("invalid subscription ID: %d for topic: %s",
				subscriptionId, topic)
		} else {
			return subscribers[subscriptionId], nil
		}
	}
}

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
	slog.Debug(brokerState.brokerStateToString())
}

func handleUnsubscribe(conn net.Conn, req *http.Request) {
	slog.Debug(fmt.Sprintf("Handling unsubscribe request: %s %s %s", req.Method,
		req.URL.Path, req.Proto))
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
	id, err := strconv.ParseInt(req.URL.Query().Get("id"), 10, 64)
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
