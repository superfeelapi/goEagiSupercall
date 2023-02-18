package worker

import (
	"context"
	"strings"
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

	g := newGoogleResponse()

	w.logger.Infow("worker: speech2TextOperation: G listening")
	for {
		select {
		case google := <-googleCh:
			go func(google goEagi.GoogleResult, g *googleResponse) {
				if google.Error != nil {
					w.Shutdown(google.Error)
				}

				if google.Info != "" && !google.Reinitialized {
					w.logger.Infow("worker: speech2TextOperation:", "agiID", w.config.AgiID, "info", google.Info)
					return
				}

				if google.Reinitialized {
					w.logger.Infow("worker: speech2TextOperation:", "agiID", w.config.AgiID, "info[Reinitialization]", google.Info)
					transcrpt := g.getTranscription()
					isFnl := g.getIsFinal()

					if !isFnl {
						if isStringNotEmpty(transcrpt) {
							w.logger.Infow("worker: speech2TextOperation:", "transcription", transcrpt, "isFinal", true)
							w.fullTranscriptCh <- transcrpt

							if w.state.Get(state.Wauchat) {
								w.wauchatTranscriptCh <- transcrpt
							}

							if w.state.Get(state.Voicebot) {
								w.paceTranscriptCh <- transcriptionLength(w.config.Language, transcrpt)
							}
						}
					}
				} else {
					transcrpt := google.Result.Alternatives[0].Transcript
					isFnl := google.Result.IsFinal

					g.setTranscription(transcrpt)
					g.setIsFinal(isFnl)

					switch isFnl {
					case false:
						w.logger.Infow("worker: speech2TextOperation:", "transcription", transcrpt, "isFinal", isFnl)
						w.interimTranscriptCh <- transcrpt

					case true:
						w.logger.Infow("worker: speech2TextOperation:", "transcription", transcrpt, "isFinal", isFnl)
						w.fullTranscriptCh <- transcrpt

						if w.state.Get(state.Wauchat) {
							w.wauchatTranscriptCh <- transcrpt
						}

						if w.state.Get(state.Voicebot) {
							w.paceTranscriptCh <- transcriptionLength(w.config.Language, transcrpt)
						}
					}
				}
			}(google, g)

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
