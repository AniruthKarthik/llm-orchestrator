package events

import (
	"sync"
)

type EventHandler func(event Event)

type EventBus struct {
	subscribers map[EventType][]EventHandler
	mutex       sync.RWMutex
}

func NewEventBus() *EventBus {
	newEventBus := EventBus{
		subscribers: map[EventType][]EventHandler{},
	}
	return &newEventBus
}

func (b *EventBus) Subscribe(
	eventType EventType,
	handler EventHandler,
) {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	if b.subscribers[eventType] == nil {
		b.subscribers[eventType] = []EventHandler{}
	}

	b.subscribers[eventType] = append(b.subscribers[eventType], handler)
}

func (b *EventBus) Publish(
	event Event,
) {

	b.mutex.RLock()

	handlers := append([]EventHandler{}, b.subscribers[event.Type]...)

	b.mutex.RUnlock()

	for _, handler := range handlers {
		go handler(event)
	}
}
