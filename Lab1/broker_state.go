package main

import (
	"fmt"
	"log/slog"
)

var brokerState, _ = CreateBrokerState() // Global broker state for pub/sub

func (b *BrokerState) BrokerStateToString() string {
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
	return BrokerState{mutex: sem, nextId: 0,
		queues: make(map[string]map[int]SubscriptionChannel)}, nil
}

// CreateSubscriptionChannel Each subscription buffers up to 100 messages.
func CreateSubscriptionChannel() SubscriptionChannel {
	return make(SubscriptionChannel, 100)
}

// AddSubscriberSafe adds a new subscriber to the topic in a thread-safe manner.
// TODO(hj): Guard from becoming too large, instead of failing with OOM.
// TODO(hj): Handle nonce wraparound. For now we are okay with max 2^31-1.
// TODO(hj): Improvement: Random subscripber IDs, maybe BigInt or strings to of
// at least 128 byte randomness. For now, just use a nonce.
func (b *BrokerState) AddSubscriberSafe(topic string) (subscriptionId int) {
	b.mutex.Acquire()
	defer b.mutex.Release()

	if _, exists := b.queues[topic]; !exists {
		b.queues[topic] = make(map[int]SubscriptionChannel)
	}
	id := b.nextId
	b.nextId++
	subscriptionId = id
	b.queues[topic][subscriptionId] = CreateSubscriptionChannel()
	return subscriptionId
}

func (b *BrokerState) UnsubscribeSafe(topic string, subscriptionId int) error {
	b.mutex.Acquire()
	defer b.mutex.Release()

	if subscribers, exists := b.queues[topic]; !exists {
		return fmt.Errorf("no subscribers for topic: %s", topic)
	} else {
		if _, exists := subscribers[subscriptionId]; !exists {
			return fmt.Errorf("invalid subscription ID: %d for topic: %s",
				subscriptionId, topic)
		} else {
			// Remove the subscriber by setting it to nil
			delete(subscribers, subscriptionId)
			return nil
		}
	}
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
		for id, subscriber := range subscribers {
			select {
			case subscriber <- message:
				// Message sent successfully
			default:
				// Subscriber's channel is full, drop the message for this subscriber
				slog.Debug(fmt.Sprintf(
					"Dropping message for subscriber %d on topic: %s", id,
					topic))
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
		if _, exists := subscribers[subscriptionId]; !exists {
			return nil, fmt.Errorf("invalid subscription ID: %d for topic: %s",
				subscriptionId, topic)
		}
		return subscribers[subscriptionId], nil
	}
}
