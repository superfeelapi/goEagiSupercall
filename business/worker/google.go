package worker

import (
	"context"
	"fmt"
)

func (w *Worker) speech2TextOperation() {
	w.logger.Infow("worker: speech2TextOperation: G started")
	defer w.logger.Infow("worker: speech2TextOperation: G completed")

	defer close(w.interimTranscriptCh)
	defer close(w.fullTranscriptCh)

	errCh := w.google.StartStreaming(context.Background(), w.toGoogleCh)
	googleCh := w.google.SpeechToTextResponse(context.Background())

	w.logger.Infow("worker: speech2TextOperation: G listening")

	var transcriptionData string

	for {
		select {
		case google := <-googleCh:
			transcript := google.Result.Alternatives[0].Transcript
			transcriptionData = transcript
			isFinal := google.Result.IsFinal

			if google.Reinitialized {
				w.fullTranscriptCh <- transcriptionData
				w.logger.Infow("worker: speech2TextOperation: G reinitialized")
				continue
			}

			switch isFinal {
			case false:
				w.logger.Infow("worker: speech2TextOperation:", "transcription", transcript, "isFinal", isFinal)
				w.eagi.Verbose(fmt.Sprintf("INTERIM TRANSCRIPTION: %s", transcript))
				//w.interimTranscriptCh <- transcript

			case true:
				w.logger.Infow("worker: speech2TextOperation:", "transcription", transcript, "isFinal", isFinal)
				w.eagi.Verbose(fmt.Sprintf("FULL TRANSCRIPTION: %s", transcript))
				//w.fullTranscriptCh <- transcript
			}

		case err := <-errCh:
			w.Shutdown(err)
			return

		case <-w.shut:
			w.logger.Infow("worker: speech2TextOperation: received shut signal")
			return
		}
	}
}
