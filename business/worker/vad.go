package worker

import (
	"time"

	"github.com/superfeelapi/goEagi/v2"
	"github.com/superfeelapi/goVoicebot/foundation/pubsub"
	"github.com/superfeelapi/goVoicebot/foundation/state"
)

func (w *Worker) vadOperation() {
	w.logger.Infow("worker: vadOperation: G started")
	defer w.logger.Infow("worker: vadOperation: G completed")

	sub := pubsub.NewSubscriber(0)
	w.broker.Subscribe(audioTopic, sub)
	defer w.broker.UnSubscribe(audioTopic, sub)

	dataCh := sub.GetChannel()

	var latestFrame []byte
	var speechFrame []byte
	var isSpeech bool

	timer := time.Now()

	for {
		select {
		case audio := <-dataCh:
			latestFrame = append(latestFrame, audio.([]byte)...)

			if time.Since(timer).Seconds() > 1 {
				amp, err := goEagi.ComputeAmplitude(latestFrame)
				if err != nil {
					w.Shutdown(err)
					return
				}

				switch amp > w.config.AmplitudeThreshold {

				case true:
					isSpeech = true
					speechFrame = append(speechFrame, latestFrame...)

					if w.state.Get(state.GoVad) {
						err = w.broker.Publish(vadToGrpcTopic, true)
						if err != nil {
							w.Shutdown(err)
							return
						}
					}

				case false:
					err = w.broker.Publish(vadToGrpcTopic, false)
					if err != nil {
						w.Shutdown(err)
						return
					}

					if isSpeech {
						if w.state.Get(state.Voicebot) {
							audioFile := createAudioFile(w.config.AgiID)
							audioFilepath, err := goEagi.GenerateAudio(speechFrame, w.config.AudioDir, audioFile)
							if err != nil {
								w.Shutdown(err)
								return
							}

							err = w.broker.Publish(audioPathFromVadTopic, audioFilepath)
							if err != nil {
								w.Shutdown(err)
								return
							}

							speechFrame = nil
							isSpeech = false
						}
					}
				}
				latestFrame = nil
				timer = time.Now()
			}

		case <-w.shut:
			w.logger.Infow("worker: vadOperation: received shut signal")
			return
		}
	}
}

// =================================================================================================================

func createAudioFile(agiID string) string {
	const layout = "2006-01-02-15:04:05"
	t := time.Now()

	return agiID + "-" + t.Format(layout) + ".wav"
}
