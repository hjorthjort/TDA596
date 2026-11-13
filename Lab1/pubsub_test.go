package main

import (
	"bufio"
	"bytes"
	"fmt"
	"net"
	"net/http"
	"sync"
	"testing"
	"time"
)

// A gated writer lets the test hold a poll in response writing deterministically.
type pollTestConn struct {
	net.Conn
	output bytes.Buffer
	started chan struct{}
	resume chan struct{}
	once sync.Once
}

func (c *pollTestConn) SetDeadline(time.Time) error { return nil }

func (c *pollTestConn) Write(p []byte) (int, error) {
	if c.started != nil {
		c.once.Do(func() { close(c.started) })
		<-c.resume
	}
	return c.output.Write(p)
}

func TestOverlappingPollRejectedThroughResponseWrite(t *testing.T) {
	previousBroker, previousSemaphore := brokerState, semaphore
	defer func() { brokerState, semaphore = previousBroker, previousSemaphore }()
	brokerState, _ = CreateBrokerState()
	semaphore, _ = CreateSemaphore(10)
	id := brokerState.AddSubscriberSafe("topic")
	otherID := brokerState.AddSubscriberSafe("topic")
	if err := brokerState.PublishSafe("topic", []byte("hello")); err != nil {
		t.Fatal(err)
	}
	request := func(id int) *http.Request {
		req, err := http.NewRequest("GET", fmt.Sprintf("http://localhost/events?topic=topic&id=%d", id), nil)
		if err != nil {
			t.Fatal(err)
		}
		return req
	}
	run := func(conn *pollTestConn, req *http.Request) {
		semaphore.Acquire()
		defer semaphore.Release()
		handleEvents(conn, req)
	}
	first := &pollTestConn{started: make(chan struct{}), resume: make(chan struct{})}
	done := make(chan struct{})
	req := request(id)
	go func() {
		defer close(done)
		run(first, req)
	}()
	defer func() {
		close(first.resume)
		<-done
	}()
	select {
	case <-first.started:
	case <-time.After(time.Second):
		t.Fatal("first poll did not reach response writing")
	}
	assertStatus := func(conn *pollTestConn, want int) {
		response, err := http.ReadResponse(bufio.NewReader(&conn.output), nil)
		if err != nil {
			t.Fatal(err)
		}
		defer response.Body.Close()
		if response.StatusCode != want {
			t.Fatalf("expected status %d, got %d", want, response.StatusCode)
		}
	}
	duplicate := &pollTestConn{}
	run(duplicate, request(id))
	assertStatus(duplicate, 409)
	other := &pollTestConn{}
	run(other, request(otherID))
	assertStatus(other, 200)
	// A completed poll must release its slot so this subscriber can poll again.
	if err := brokerState.PublishSafe("topic", []byte("next")); err != nil {
		t.Fatal(err)
	}
	next := &pollTestConn{}
	run(next, request(otherID))
	assertStatus(next, 200)
}
