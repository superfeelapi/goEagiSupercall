package pubsub

import (
	"fmt"
	"sync"
	"time"
)

type Broker struct {
	topics map[string][]*Subscriber
	sync.RWMutex
}

func NewBroker() *Broker {
	return &Broker{
		topics: make(map[string][]*Subscriber, 0),
	}
}

func (b *Broker) Publish(topic string, data any) error {
	var exists bool
	var subs []*Subscriber

	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			return fmt.Errorf("topic[%s] does not exist", topic)

		default:
			b.RLock()
			{
				subs, exists = b.topics[topic]
			}
			b.RUnlock()

			if exists {
				for _, sub := range subs {
					sub.Signal(data)
				}
				return nil
			}

			time.Sleep(1 * time.Second)
		}
	}
}

func (b *Broker) Subscribe(topic string, s *Subscriber) {
	b.Lock()
	defer b.Unlock()
	{
		_, exists := b.topics[topic]
		if !exists {
			b.topics[topic] = make([]*Subscriber, 0)
		}

		b.topics[topic] = append(b.topics[topic], s)
	}
}

func (b *Broker) UnSubscribe(topic string, s *Subscriber) error {
	b.Lock()
	defer b.Unlock()
	{
		subs, exists := b.topics[topic]
		if !exists {
			return fmt.Errorf("topic[%s] does not exists", topic)
		}

		b.topics[topic] = removeFromSlice(subs, s)
		s.CloseChannel()
	}

	return nil
}

// =================================================================================================================

func removeFromSlice[T comparable](s []T, d T) []T {
	for i := range s {
		if s[i] == d {
			s[i] = s[len(s)-1]
			return s[:len(s)-1]
		}
	}
	return s
}
