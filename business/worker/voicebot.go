package worker

import (
	"github.com/superfeelapi/goEagiSupercall/foundation/external/voicebot"
	"github.com/superfeelapi/goEagiSupercall/foundation/state"
)

func (w *Worker) voicebotOperation() {
	w.logger.Infow("worker: voicebotOperation: G started")
	defer w.logger.Infow("worker: voicebotOperation: G completed")

	defer w.state.Set(state.Voicebot, false)
	defer close(w.voicebotCh)

	var apiEndpoint string
	if w.config.Actor == "agent" {
		apiEndpoint = w.config.VoicebotAgentEndpoint
	} else {
		apiEndpoint = w.config.VoicebotCustomerEndpoint
	}

	w.logger.Infow("worker: voicebotOperation: G listening")
	for {
		select {
		case audioPath := <-w.audioPathCh:
			if !w.state.Get(state.Voicebot) {
				return
			}
			go func() {
				resp, err := voicebot.VoiceEmotion(apiEndpoint, w.config.VoicebotApiKey, audioPath)
				if err != nil {
					w.logger.Errorw("worker: voicebotOperation", "ERROR", err)
					return
				}
				w.voicebotCh <- resp
				w.logger.Infow("worker: voicebotOperation:", "voice emotion", resp)
			}()

		case <-w.shut:
			w.logger.Infow("worker: voicebotOperation: received shut signal")
			return
		}
	}
}
