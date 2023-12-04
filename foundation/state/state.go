package state

import "sync"

type Service int

const (
	Voicebot Service = iota
	Wauchat
	GoVad
	Redis
)

type State struct {
	sync.RWMutex

	Voicebot bool
	Wauchat  bool
	VadGrpc  bool
	Redis    bool
}

func NewState() *State {
	return &State{
		Voicebot: true,
		Wauchat:  true,
		VadGrpc:  true,
		Redis:    true,
	}
}

func (s *State) Get(svc Service) bool {
	s.RLock()
	defer s.RUnlock()
	{
		switch svc {
		case Voicebot:
			return s.Voicebot

		case Wauchat:
			return s.Wauchat

		case GoVad:
			return s.VadGrpc

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
		case Voicebot:
			s.Voicebot = state

		case Wauchat:
			s.Wauchat = state

		case GoVad:
			s.VadGrpc = state

		case Redis:
			s.Redis = state
		}
	}
}
