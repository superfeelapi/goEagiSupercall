package worker

import (
	"github.com/superfeelapi/goVoicebot/foundation/external/voicebot"
	"github.com/superfeelapi/goVoicebot/foundation/pubsub"
	"github.com/superfeelapi/goVoicebot/foundation/state"
)

func (w *Worker) voicebotOperation() {
	w.logger.Infow("worker: voicebotOperation: G started")
	defer w.logger.Infow("worker: voicebotOperation: G completed")
	defer w.state.Set(state.Voicebot, false)

	sub := pubsub.NewSubscriber(0)
	w.broker.Subscribe(audioPathFromVadTopic, sub)
	defer w.broker.UnSubscribe(audioPathFromVadTopic, sub)

	dataCh := sub.GetChannel()

	var apiEndpoint string
	if w.config.Actor == "agent" {
		apiEndpoint = w.config.VoicebotAgentEndpoint
	} else {
		apiEndpoint = w.config.VoicebotCustomerEndpoint
	}

	for {
		select {
		case audioPath := <-dataCh:
			if !w.state.Get(state.Voicebot) {
				return
			}
			go func() {
				w.logger.Infow("worker: voicebotOperation: requesting voicebot API")

				resp, err := voicebot.VoiceEmotion(apiEndpoint, w.config.VoicebotApiKey, audioPath.(string))
				if err != nil {
					w.logger.Errorw("worker: voicebotOperation", "ERROR", err)
					return
				}
				w.logger.Infow("worker: voicebotOperation: requested voicebot API", "response", resp)

				err = w.broker.Publish(emotionFromVoicebotTopic, resp)
				if err != nil {
					w.Shutdown(err)
					return
				}
			}()

		case <-w.shut:
			w.logger.Infow("worker: voicebotOperation: received shut signal")
			return
		}
	}
}
