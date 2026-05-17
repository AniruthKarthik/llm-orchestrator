package events

import (
	"sync"
)

type EventHandler func(event Event)

type EventBus struct {
	subscribers map[EventType][]EventHandler
	mutex       sync.RWMutex
	eventChan   chan Event
	workerCount int
}

func NewEventBus(workerCount int) *EventBus {
	b := &EventBus{
		subscribers: map[EventType][]EventHandler{},
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
	handlers := append([]EventHandler{}, b.subscribers[event.Type]...)
	b.mutex.RUnlock()

	for _, handler := range handlers {
		handler(event)
	}
}

func (b *EventBus) Subscribe(
	eventType EventType,
	handler EventHandler,
) {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	b.subscribers[eventType] = append(b.subscribers[eventType], handler)
}

func (b *EventBus) Publish(
	event Event,
) {
	select {
	case b.eventChan <- event:
	default:
		// Drop event or handle overflow (e.g., log) if channel is full
	}
}
