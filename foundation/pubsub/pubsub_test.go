package pubsub_test

import (
	"log"
	"sync"
	"testing"
	"time"

	"github.com/superfeelapi/goVoicebot/foundation/pubsub"
)

func TestBroker(t *testing.T) {
	b := pubsub.NewBroker()
	s1 := pubsub.NewSubscriber(0)
	s2 := pubsub.NewSubscriber(0)
	s3 := pubsub.NewSubscriber(0)
	s4 := pubsub.NewSubscriber(0)

	b.Subscribe("transcription", s1)
	b.Subscribe("transcription", s2)
	b.Subscribe("integer", s3)
	b.Subscribe("integer", s4)

	subs := map[int]*pubsub.Subscriber{1: s1, 2: s2, 3: s3, 4: s4}

	var wg sync.WaitGroup
	wg.Add(len(subs))

	f := func(wg *sync.WaitGroup, s *pubsub.Subscriber, i int) {
		defer wg.Done()
		defer log.Printf("%d goroutine: exit\n", i)
		ch := s.GetChannel()
		ticker := time.NewTicker(3 * time.Second)

		for {
			select {
			case out := <-ch:
				switch out.(type) {
				case string:
					log.Printf("%d goroutine: received: %s\n", i, out)
				case int:
					log.Printf("%d goroutine: received: %v\n", i, out)
				}

			case <-ticker.C:
				return
			}
		}
	}

	for i, sub := range subs {
		go f(&wg, sub, i)
	}

	log.Println("main goroutine: sending", "hello world")
	if err := b.Publish("transcription", "hello world"); err != nil {
		log.Fatalln(err)
	}

	log.Println("main goroutine: sending", "hello gophers")
	if err := b.Publish("transcription", "hello gophers"); err != nil {
		log.Fatalln(err)

	}

	log.Println("main goroutine: sending", 17)
	if err := b.Publish("integer", 17); err != nil {
		log.Fatalln(err)
	}

	log.Println("main goroutine: sending", 12)
	if err := b.Publish("integer", 12); err != nil {
		log.Fatalln(err)
	}

	wg.Wait()
}
