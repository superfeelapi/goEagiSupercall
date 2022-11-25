package worker

import (
	"context"

	"github.com/superfeelapi/goEagi/v2"
)

func (w *Worker) audioStreamOperation() {
	w.logger.Infow("worker: audioStreamOperation: G started")
	defer w.logger.Infow("worker: audioStreamOperation: G completed")

	streamCh := goEagi.AudioStreaming(context.Background())

	for {
		select {
		case audio := <-streamCh:
			if audio.Error != nil {
				w.Shutdown(audio.Error)
				return
			}
			if err := w.broker.Publish(audioTopic, audio.Stream); err != nil {
				w.Shutdown(err)
				return
			}

		case <-w.shut:
			w.logger.Infow("worker: audioStreamOperation: received shut signal")
			return
		}
	}
}
