package worker

import (
	"github.com/superfeelapi/goVoicebot/foundation/external/wauchat"
	"github.com/superfeelapi/goVoicebot/foundation/pubsub"
	"github.com/superfeelapi/goVoicebot/foundation/state"
)

func (w *Worker) wauchatOperation() {
	w.logger.Infow("worker: wauchatOperation: G started")
	defer w.logger.Infow("worker: wauchatOperation: G completed")
	defer w.state.Set(state.Wauchat, false)

	sub := pubsub.NewSubscriber(0)
	w.broker.Subscribe(transcriptionToWauchatTopic, sub)
	defer w.broker.UnSubscribe(transcriptionToWauchatTopic, sub)

	dataCh := sub.GetChannel()

	for {
		select {
		case transcription := <-dataCh:
			if !w.state.Get(state.Wauchat) {
				return
			}
			go func() {
				w.logger.Infow("worker: wauchatOperation: requesting wauchat API")

				resp, err := wauchat.TextEmotion(w.config.WauchatEndpoint, transcription.(string))
				if err != nil {
					w.logger.Errorw("worker: wauchatOperation", "ERROR", err)
					return
				}
				w.logger.Infow("worker: wauchatOperation: requested wauchat API", "response", resp)

				err = w.broker.Publish(emotionFromWauchatTopic, resp)
				if err != nil {
					w.Shutdown(err)
					return
				}
			}()

		case <-w.shut:
			w.logger.Infow("worker: wauchatOperation: received shut signal")
			return
		}
	}
}
