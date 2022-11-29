package worker

import (
	"context"

	"github.com/superfeelapi/goEagi"
	"github.com/superfeelapi/goEagiSupercall/foundation/state"
)

func (w *Worker) audioStreamOperation() {
	w.logger.Infow("worker: audioStreamOperation: G started")
	defer w.logger.Infow("worker: audioStreamOperation: G completed")

	defer close(w.toGoogleCh)

	streamCh := goEagi.AudioStreaming(context.Background())

	w.logger.Infow("worker: audioStreamOperation: G listening")
	for {
		select {
		case audio := <-streamCh:
			if audio.Error != nil {
				w.Shutdown(audio.Error)
				return
			}
			w.toGoogleCh <- audio.Stream

			if w.state.Get(state.Voicebot) || w.state.Get(state.GoVad) {
				w.toVadCh <- audio.Stream
			}

		case <-w.shut:
			w.logger.Infow("worker: audioStreamOperation: received shut signal")
			return
		}
	}
}
