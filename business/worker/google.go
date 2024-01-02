package worker

import (
	"context"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/abadojack/whatlanggo"
	"github.com/superfeelapi/goEagiSupercall/foundation/state"
)

func (w *Worker) googleOperation() {
	w.logger.Infow("worker: googleOperation: G started")
	defer w.logger.Infow("worker: googleOperation: G completed")

	errCh := w.google.StartStreaming(context.Background(), w.toSpeechCh)
	googleCh := w.google.SpeechToTextResponse(context.Background())

	var transcriptionData string

	w.logger.Infow("worker: googleOperation: G listening")
	for {
		select {
		case google := <-googleCh:
			transcript := google.Result.Alternatives[0].Transcript
			transcriptionData = transcript
			isFinal := google.Result.IsFinal

			if google.Reinitialized {
				w.fullTranscriptCh <- transcriptionData

				if w.state.Get(state.Wauchat) {
					w.textEmotionTranscriptCh <- transcript
				}

				if w.state.Get(state.Voicebot) {
					w.paceTranscriptCh <- transcriptionLength(transcript)
				}

				w.logger.Infow("worker: googleOperation: G reinitialized")
				continue
			}

			switch isFinal {
			case false:
				w.logger.Infow("worker: googleOperation:", "transcription", transcript, "isFinal", isFinal)
				w.interimTranscriptCh <- transcript

			case true:
				w.logger.Infow("worker: googleOperation:", "transcription", transcript, "isFinal", isFinal)
				w.fullTranscriptCh <- transcript

				if w.state.Get(state.Wauchat) {
					w.textEmotionTranscriptCh <- transcript
				}

				if w.state.Get(state.Voicebot) {
					w.paceTranscriptCh <- transcriptionLength(transcript)
				}
			}

		case err := <-errCh:
			w.Shutdown(fmt.Errorf("worker: googleOperation: %w", err))
			return

		case <-w.shut:
			w.logger.Infow("worker: googleOperation: received shut signal")
			return
		}
	}
}

// =================================================================================================================

func transcriptionLength(text string) int {
	info := whatlanggo.Detect(text)

	switch whatlanggo.Scripts[info.Script] {
	case "Latin":
		return len(strings.Fields(text))

	case "Han", "Arabic":
		return utf8.RuneCountInString(text)
	}

	switch info.Lang.String() {
	case "Japanese":
		return utf8.RuneCountInString(text)
	}

	return utf8.RuneCountInString(text)
}
