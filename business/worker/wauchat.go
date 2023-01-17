package worker

import (
	"github.com/superfeelapi/goEagiSupercall/foundation/external/wauchat"
	"github.com/superfeelapi/goEagiSupercall/foundation/state"
)

func (w *Worker) textEmotionOperation() {
	w.logger.Infow("worker: wauchatOperation: G started")
	defer w.logger.Infow("worker: wauchatOperation: G completed")

	defer w.state.Set(state.Wauchat, false)

	w.logger.Infow("worker: wauchatOperation: G listening")
	for {
		select {
		case transcription := <-w.wauchatTranscriptCh:
			if !w.state.Get(state.Wauchat) {
				return
			}

			resp, err := wauchat.TextEmotion(w.config.WauchatEndpoint, transcription)
			if err != nil {
				w.logger.Errorw("worker: wauchatOperation", "ERROR", err)
				return
			}
			w.wauchatCh <- resp
			w.logger.Infow("worker: wauchatOperation:", "text emotion", resp)

		case <-w.shut:
			w.logger.Infow("worker: wauchatOperation: received shut signal")
			return
		}
	}
}
