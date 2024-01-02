package worker

import (
	"github.com/superfeelapi/goEagiSupercall/foundation/external/textAnalysis"
	"github.com/superfeelapi/goEagiSupercall/foundation/state"
)

func (w *Worker) textEmotionOperation() {
	w.logger.Infow("worker: textEmotionOperation: G started")
	defer w.logger.Infow("worker: textEmotionOperation: G completed")

	defer w.state.Set(state.Wauchat, false)

	w.logger.Infow("worker: textEmotionOperation: G listening")
	for {
		select {
		case transcription := <-w.textEmotionTranscriptCh:
			if !w.state.Get(state.Wauchat) {
				return
			}
			go func() {
				resp, err := textAnalysis.TextEmotion(w.config.TextAnalysisEndpoint, transcription)
				if err != nil {
					w.logger.Errorw("worker: textEmotionOperation", "ERROR", err)
					return
				}
				w.textAnalysisCh <- resp
				w.logger.Infow("worker: textEmotionOperation:", "text emotion", resp)
			}()

		case <-w.shut:
			w.logger.Infow("worker: textEmotionOperation: received shut signal")
			return
		}
	}
}
