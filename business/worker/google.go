package worker

import (
	"context"
)

func (w *Worker) googleOperation() {
	w.logger.Infow("worker: googleOperation: G started")
	defer w.logger.Infow("worker: googleOperation: G completed")

	errCh := w.google.StartStreaming(context.Background(), w.toSpeechCh)
	googleCh := w.google.SpeechToTextResponse(context.Background())

	w.logger.Infow("worker: googleOperation: G listening")

	var transcriptionData string

	for {
		select {
		case google := <-googleCh:
			transcript := google.Result.Alternatives[0].Transcript
			transcriptionData = transcript
			isFinal := google.Result.IsFinal

			if google.Reinitialized {
				w.fullTranscriptCh <- transcriptionData
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
			}

		case err := <-errCh:
			w.Shutdown(err)
			return

		case <-w.shut:
			w.logger.Infow("worker: googleOperation: received shut signal")
			return
		}
	}
}
