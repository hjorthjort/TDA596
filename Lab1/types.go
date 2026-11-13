package main

import (
	"os"
	"time"
)

type ServerState struct {
	semaphore *Semaphore
	wwwDir    string
	rootDir   *os.Root
	timeout   time.Duration
}

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

type Message struct {
	Sequence uint64
	Body     []byte
}

// A subscription should only be polled by one connection at a time.
type SubscriptionChannel struct {
	Queue chan Message
	mutex *Semaphore
}
type BrokerState struct {
	mutex     *Semaphore
	nextId    int // Unique nonce, allows up to total of 2^31-1 subscribers total
	queues    map[string]map[int]*SubscriptionChannel
	sequences map[string]uint64 // Last publication number for each topic
}
