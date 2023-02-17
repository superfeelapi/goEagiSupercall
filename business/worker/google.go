package worker

import (
	"context"
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/superfeelapi/goEagi"
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

	var transcription string
	var isFinal bool
	var m sync.RWMutex

	for {
		select {
		case google := <-googleCh:
			go func(google goEagi.GoogleResult, m *sync.RWMutex) {
				if google.Error != nil {
					w.Shutdown(google.Error)
				}

				if google.Info != "" && !google.Reinitialized {
					w.logger.Infow("worker: speech2TextOperation:", "agiID", w.config.AgiID, "info", google.Info)
					return
				}

				if google.Result != nil && google.Result.Alternatives != nil {
					m.Lock()
					transcription = google.Result.Alternatives[0].Transcript
					isFinal = google.Result.IsFinal
					m.Unlock()
				}

				if google.Reinitialized {
					w.logger.Infow("worker: speech2TextOperation:", "agiID", w.config.AgiID, "info[Reinitialization]", google.Info)

					if !isFinal {
						if isStringNotEmpty(transcription) {
							w.logger.Infow("worker: speech2TextOperation:", "transcription", transcription, "isFinal", true)
							w.fullTranscriptCh <- transcription

							if w.state.Get(state.Wauchat) {
								w.wauchatTranscriptCh <- transcription
							}

							if w.state.Get(state.Voicebot) {
								w.paceTranscriptCh <- transcriptionLength(w.config.Language, transcription)
							}
						}
					}

				} else {
					switch google.Result.IsFinal {
					case false:
						w.logger.Infow("worker: speech2TextOperation:", "transcription", transcription, "isFinal", google.Result.IsFinal)
						w.interimTranscriptCh <- transcription

					case true:
						w.logger.Infow("worker: speech2TextOperation:", "transcription", transcription, "isFinal", google.Result.IsFinal)
						w.fullTranscriptCh <- transcription

						if w.state.Get(state.Wauchat) {
							w.wauchatTranscriptCh <- transcription
						}

						if w.state.Get(state.Voicebot) {
							w.paceTranscriptCh <- transcriptionLength(w.config.Language, transcription)
						}
					}
				}
			}(google, &m)

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

func isStringNotEmpty(input string) bool {
	for _, char := range input {
		if char != ' ' && char != '\t' && char != '\n' {
			return true
		}
	}
	return false
}
