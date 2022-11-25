package worker

import (
	"context"
	"strings"
	"unicode/utf8"

	"github.com/superfeelapi/goVoicebot/foundation/pubsub"
	"github.com/superfeelapi/goVoicebot/foundation/state"
)

func (w *Worker) speech2TextOperation() {
	w.logger.Infow("worker: speech2TextOperation: G started")
	defer w.logger.Infow("worker: speech2TextOperation: G completed")

	sub := pubsub.NewSubscriber(0)
	w.broker.Subscribe(audioTopic, sub)
	defer w.broker.UnSubscribe(audioTopic, sub)

	dataCh := sub.GetChannel()
	toGoogleCh := make(chan []byte)

	errCh := w.google.StartStreaming(context.Background(), toGoogleCh)
	googleCh := w.google.SpeechToTextResponse(context.Background())

	for {
		select {
		case audio := <-dataCh:
			toGoogleCh <- audio.([]byte)

		case google := <-googleCh:
			go func() {
				transcription := google.Result.Alternatives[0].Transcript

				switch google.Result.IsFinal {
				case false:
					if err := w.broker.Publish(interimTranscriptionToSupercallTopic, transcription); err != nil {
						w.Shutdown(err)
						return
					}

				case true:
					if err := w.broker.Publish(fullTranscriptionToSupercallTopic, transcription); err != nil {
						w.Shutdown(err)
						return
					}

					if w.state.Get(state.Wauchat) {
						if err := w.broker.Publish(transcriptionToWauchatTopic, transcription); err != nil {
							w.Shutdown(err)
							return
						}
					}
					if err := w.broker.Publish(transcriptionPaceTopic, transcriptionLength(w.config.Language, transcription)); err != nil {
						w.Shutdown(err)
						return
					}
				}
			}()

		case err := <-errCh:
			w.Shutdown(err)
			return

		case <-w.shut:
			w.logger.Infow("worker: speech2TextOperation: received shut signal")
			return
		}
	}
}

// =================================================================================================================

func transcriptionLength(language string, s string) int {
	switch language {
	case "english":
		return len(strings.Split(s, " "))
	case "chinese":
		return utf8.RuneCountInString(s)
	case "japan":
		return utf8.RuneCountInString(s)
	default:
		return len(strings.Split(s, " "))
	}
}
