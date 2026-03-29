package bus

import (
	"context"
	"encoding/json"
	"sync"
	"time"
)

// Subscriber is a channel-based subscriber
type Subscriber struct {
	ch   chan *Event
	done chan struct{}
}

// Bus is an in-memory event bus using Go Channels
type Bus struct {
	mu          sync.RWMutex
	subscribers map[string][]*Subscriber
	ctx         context.Context
	cancel      context.CancelFunc
	wg          sync.WaitGroup
}

// New creates a new in-memory event bus
func New() *Bus {
	ctx, cancel := context.WithCancel(context.Background())
	return &Bus{
		subscribers: make(map[string][]*Subscriber),
		ctx:         ctx,
		cancel:      cancel,
	}
}

// Publish publishes an event to the bus synchronously
func (b *Bus) Publish(topic string, payload interface{}) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	event := &Event{
		Topic:   topic,
		Payload: json.RawMessage(data),
		Time:    time.Now(),
	}

	b.mu.RLock()
	subs, ok := b.subscribers[topic]
	b.mu.RUnlock()

	if !ok || len(subs) == 0 {
		return nil
	}

	// Fan-out to all subscribers
	for _, sub := range subs {
		select {
		case sub.ch <- event:
		default:
			// channel full, skip
		}
	}
	return nil
}

// Subscribe subscribes to a topic and returns a channel of events
// The caller must drain the channel to avoid blocking
func (b *Bus) Subscribe(topic string) (<-chan *Event, func()) {
	sub := &Subscriber{
		ch:   make(chan *Event, 100),
		done: make(chan struct{}),
	}

	b.mu.Lock()
	b.subscribers[topic] = append(b.subscribers[topic], sub)
	b.mu.Unlock()

	cleanup := func() {
		b.mu.Lock()
		subs := b.subscribers[topic]
		for i, s := range subs {
			if s == sub {
				b.subscribers[topic] = append(subs[:i], subs[i+1:]...)
				break
			}
		}
		b.mu.Unlock()
		close(sub.ch)
		close(sub.done)
	}

	return sub.ch, cleanup
}

// Request sends a request and waits for a reply on the reply topic
func (b *Bus) Request(ctx context.Context, topic string, replyTopic string, payload interface{}, timeout time.Duration) (*Event, error) {
	if err := b.Publish(topic, payload); err != nil {
		return nil, err
	}

	replyCh, cleanup := b.Subscribe(replyTopic)
	defer cleanup()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case reply := <-replyCh:
		return reply, nil
	case <-time.After(timeout):
		return nil, context.DeadlineExceeded
	}
}

// Close gracefully shuts down the bus
func (b *Bus) Close() {
	b.cancel()
	b.wg.Wait()
}
