package pubsub

type Subscriber struct {
	payload chan any
}

func NewSubscriber(channelCapacity int) *Subscriber {
	if channelCapacity > 0 {
		return &Subscriber{
			payload: make(chan any, channelCapacity),
		}
	}
	return &Subscriber{
		payload: make(chan any),
	}
}

func (s *Subscriber) Signal(data any) {
	s.payload <- data
}

func (s *Subscriber) GetChannel() <-chan any {
	return s.payload
}

func (s *Subscriber) CloseChannel() {
	close(s.payload)
}
