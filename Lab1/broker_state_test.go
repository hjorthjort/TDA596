package main

import (
	"fmt"
	"sync"
	"testing"
)

func TestConcurrentPublicationSequences(t *testing.T) {
	b, err := CreateBrokerState()
	if err != nil {
		t.Fatal(err)
	}
	a := b.AddSubscriberSafe("topic")
	c := b.AddSubscriberSafe("topic")
	firstSubscription, _ := b.EventSafe("topic", a)
	first := firstSubscription.Queue
	secondSubscription, _ := b.EventSafe("topic", c)
	second := secondSubscription.Queue
	var publishers sync.WaitGroup
	for i := 0; i < 50; i++ {
		publishers.Add(1)
		go func(i int) {
			defer publishers.Done()
			if err := b.PublishSafe("topic", []byte(fmt.Sprint(i))); err != nil {
				t.Error(err)
			}
		}(i)
	}
	publishers.Wait()
	if len(first) != 50 || len(second) != 50 {
		t.Fatalf("expected 50 messages per subscriber, got %d and %d", len(first), len(second))
	}
	for sequence := uint64(0); sequence < 50; sequence++ {
		x, y := <-first, <-second
		if x.Sequence != sequence || y.Sequence != sequence || string(x.Body) != string(y.Body) {
			t.Fatalf("publication %d differs between subscribers: %+v, %+v", sequence, x, y)
		}
	}
}

func TestSequencesSurviveDropsAndResubscription(t *testing.T) {
	b, err := CreateBrokerState()
	if err != nil {
		t.Fatal(err)
	}
	id := b.AddSubscriberSafe("topic")
	subscription, _ := b.EventSafe("topic", id)
	queue := subscription.Queue
	for i := 0; i < cap(queue)+1; i++ {
		if err := b.PublishSafe("topic", []byte("message")); err != nil {
			t.Fatal(err)
		}
	}
	for i := 0; i < cap(queue); i++ {
		if msg := <-queue; msg.Sequence != uint64(i) {
			t.Fatalf("expected sequence %d, got %d", i, msg.Sequence)
		}
	}
	if err := b.PublishSafe("topic", []byte("after drop")); err != nil {
		t.Fatal(err)
	}
	if msg := <-queue; msg.Sequence != uint64(cap(queue)+1) {
		t.Fatalf("dropped publication did not leave a sequence gap: %+v", msg)
	}
	if err := b.UnsubscribeSafe("topic", id); err != nil {
		t.Fatal(err)
	}
	id = b.AddSubscriberSafe("topic")
	subscription, _ = b.EventSafe("topic", id)
	queue = subscription.Queue
	if err := b.PublishSafe("topic", nil); err != nil {
		t.Fatal(err)
	}
	if msg := <-queue; msg.Sequence != uint64(cap(queue)+2) {
		t.Fatalf("sequence reset after resubscription: %+v", msg)
	}
	otherID := b.AddSubscriberSafe("other")
	otherSubscription, _ := b.EventSafe("other", otherID)
	other := otherSubscription.Queue
	if err := b.PublishSafe("other", nil); err != nil {
		t.Fatal(err)
	}
	if msg := <-other; msg.Sequence != 0 {
		t.Fatalf("new topic should start at 0, got %d", msg.Sequence)
	}
}
