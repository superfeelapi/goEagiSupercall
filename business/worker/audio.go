package worker

import (
	"context"

	"github.com/superfeelapi/goEagi"
)

func (w *Worker) audioStreamOperation() {
	w.logger.Infow("worker: audioStreamOperation: G started")
	defer w.logger.Infow("worker: audioStreamOperation: G completed")

	streamCh := goEagi.AudioStreaming(context.Background())

	w.logger.Infow("worker: audioStreamOperation: G listening")
	for {
		select {
		case audio := <-streamCh:
			if audio.Error != nil {
				w.Shutdown(audio.Error)
				return
			}
			w.toSpeechCh <- audio.Stream

		case <-w.shut:
			w.logger.Infow("worker: audioStreamOperation: received shut signal")
			return
		}
	}
}
