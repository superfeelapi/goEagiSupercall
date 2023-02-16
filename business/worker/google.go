package worker

import (
	"context"
	"strings"
	"unicode/utf8"

	"github.com/superfeelapi/goEagiSupercall/foundation/state"
)

func (w *Worker) speech2TextOperation() {
	w.logger.Infow("worker: speech2TextOperation: G started")
	defer w.logger.Infow("worker: speech2TextOperation: G completed")

	defer close(w.interimTranscriptCh)
	defer close(w.fullTranscriptCh)
	defer close(w.wauchatTranscriptCh)
	defer close(w.paceTranscriptCh)

	errCh := w.google.StartStreaming(context.Background(), w.toGoogleCh)
	googleCh := w.google.SpeechToTextResponse(context.Background())

	w.logger.Infow("worker: speech2TextOperation: G listening")
	for {
		select {
		case google := <-googleCh:
			go func() {
				if google.Error != nil {
					w.Shutdown(google.Error)
				}

				if google.Info != "" {
					w.logger.Infow("worker: speech2TextOperation:", "agiID", w.config.AgiID, "info", google.Info)
				}

				transcription := google.Result.Alternatives[0].Transcript
				w.logger.Infow("worker: speech2TextOperation:", "transcription", transcription, "isFinal", google.Result.IsFinal)

				switch google.Result.IsFinal {
				case false:
					w.interimTranscriptCh <- transcription

				case true:
					w.fullTranscriptCh <- transcription

					if w.state.Get(state.Wauchat) {
						w.wauchatTranscriptCh <- transcription
					}

					if w.state.Get(state.Voicebot) {
						w.paceTranscriptCh <- transcriptionLength(w.config.Language, transcription)
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
