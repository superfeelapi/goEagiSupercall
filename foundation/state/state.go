package state

import "sync"

type Service int

const (
	Redis Service = iota
)

type State struct {
	sync.RWMutex
	Redis bool
}

func NewState() *State {
	return &State{
		Redis: true,
	}
}

func (s *State) Get(svc Service) bool {
	s.RLock()
	defer s.RUnlock()
	{
		switch svc {
		case Redis:
			return s.Redis
		}
	}
	return false
}

func (s *State) Set(svc Service, state bool) {
	s.Lock()
	defer s.Unlock()
	{
		switch svc {
		case Redis:
			s.Redis = state
		}
	}
}
