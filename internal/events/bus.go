package events

import (
	"sync"
)

type EventHandler func(event Event)

// SubscriptionID identifies a subscription so it can be cancelled.
type SubscriptionID uint64

type subscription struct {
	id      SubscriptionID
	handler EventHandler
}

type EventBus struct {
	subscribers map[EventType][]subscription
	mutex       sync.RWMutex
	eventChan   chan Event
	workerCount int
	nextID      SubscriptionID
}

func NewEventBus(workerCount int) *EventBus {
	b := &EventBus{
		subscribers: map[EventType][]subscription{},
		eventChan:   make(chan Event, 1000), // Buffered channel
		workerCount: workerCount,
	}
	b.start()
	return b
}

func (b *EventBus) start() {
	for i := 0; i < b.workerCount; i++ {
		go func() {
			for event := range b.eventChan {
				b.dispatch(event)
			}
		}()
	}
}

func (b *EventBus) dispatch(event Event) {
	b.mutex.RLock()
	subs := append([]subscription{}, b.subscribers[event.Type]...)
	b.mutex.RUnlock()

	for _, sub := range subs {
		sub.handler(event)
	}
}

// Subscribe registers a handler and returns a SubscriptionID that can be used
// to Unsubscribe later. This is critical for WebSocket connections to avoid leaks.
func (b *EventBus) Subscribe(eventType EventType, handler EventHandler) SubscriptionID {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	b.nextID++
	id := b.nextID
	b.subscribers[eventType] = append(b.subscribers[eventType], subscription{id: id, handler: handler})
	return id
}

// Unsubscribe removes the subscription with the given ID from the given event type.
func (b *EventBus) Unsubscribe(eventType EventType, id SubscriptionID) {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	subs := b.subscribers[eventType]
	out := subs[:0]
	for _, s := range subs {
		if s.id != id {
			out = append(out, s)
		}
	}
	b.subscribers[eventType] = out
}

func (b *EventBus) Publish(event Event) {
	select {
	case b.eventChan <- event:
	default:
		// Channel full — drop event to avoid blocking callers. Logged in production.
	}
}
