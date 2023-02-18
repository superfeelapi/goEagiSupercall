package worker

import "sync"

type googleResponse struct {
	transcription string
	isFinal       bool
	sync.RWMutex
}

func newGoogleResponse() *googleResponse {
	return &googleResponse{}
}

func (g *googleResponse) getTranscription() string {
	g.RLock()
	defer g.RUnlock()
	return g.transcription
}

func (g *googleResponse) getIsFinal() bool {
	g.RLock()
	defer g.RUnlock()
	return g.isFinal
}

func (g *googleResponse) setTranscription(s string) {
	g.Lock()
	defer g.Unlock()
	g.transcription = s
}

func (g *googleResponse) setIsFinal(b bool) {
	g.Lock()
	defer g.Unlock()
	g.isFinal = b
}
