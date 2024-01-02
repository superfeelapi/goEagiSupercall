package worker

import (
	"github.com/superfeelapi/goEagiSupercall/foundation/external/voiceAnalysis"
	"github.com/superfeelapi/goEagiSupercall/foundation/state"
)

func (w *Worker) voiceEmotionOperation() {
	w.logger.Infow("worker: voiceEmotionOperation: G started")
	defer w.logger.Infow("worker: voiceEmotionOperation: G completed")

	defer w.state.Set(state.Voicebot, false)

	var apiEndpoint string
	if w.config.Actor == "agent" {
		apiEndpoint = w.config.VoiceAnalysisAgentEndpoint
	} else {
		apiEndpoint = w.config.VoiceAnalysisCustomerEndpoint
	}

	w.logger.Infow("worker: voiceEmotionOperation: G listening")
	for {
		select {
		case audioPath := <-w.audioPathCh:
			if !w.state.Get(state.Voicebot) {
				return
			}
			go func() {
				resp, err := voiceAnalysis.VoiceEmotion(apiEndpoint, w.config.VoiceAnalysisApiKey, audioPath)
				if err != nil {
					w.logger.Errorw("worker: voiceEmotionOperation", "ERROR", err)
					return
				}
				w.voiceAnalysisCh <- resp
				w.logger.Infow("worker: voiceEmotionOperation:", "voice emotion", resp)
			}()

		case <-w.shut:
			w.logger.Infow("worker: voiceEmotionOperation: received shut signal")
			return
		}
	}
}
