package worker

import (
	"fmt"

	"github.com/gorilla/websocket"
	"github.com/superfeelapi/goEagiSupercall/foundation/state"
)

func (w *Worker) azureOperation() {
	w.logger.Infow("worker: azureOperation: G started")
	defer w.logger.Infow("worker: azureOperation: G completed")

	azureResultCh := make(chan AzureResult, 10)

	// Read JSON
	go func(conn *websocket.Conn) {
		w.logger.Infow("worker: azureOperation: G started to listen for JSON")
		defer w.logger.Infow("worker: azureOperation: G completed to listen for JSON")

		for {
			var result AzureResult
			err := conn.ReadJSON(&result)
			if err != nil {
				w.Shutdown(fmt.Errorf("worker: azureOperation: G:json: conn.ReadJSON: %w", err))
				return
			}
			if result.Error != nil {
				w.Shutdown(fmt.Errorf("worker: azureOperation: G:json: result.Error: %w", result.Error))
				return
			}
			azureResultCh <- result
		}
	}(w.azure)

	// Receive Transcription
	go func() {
		w.logger.Infow("worker: azureOperation: G started to listen for TRANSCRIPTION")
		defer w.logger.Infow("worker: azureOperation: G completed to listen for TRANSCRIPTION")
		for {
			select {
			case <-w.shut:
				return

			case result := <-azureResultCh:
				switch result.IsFinal {
				case false:
					w.interimTranscriptCh <- result.Transcription
					w.logger.Infow("worker: googleOperation:", "transcription", result.Transcription, "isFinal", result.IsFinal)

				case true:
					w.fullTranscriptCh <- result.Transcription

					if w.state.Get(state.Wauchat) {
						w.textEmotionTranscriptCh <- result.Transcription
					}

					if w.state.Get(state.Voicebot) {
						w.paceTranscriptCh <- transcriptionLength(result.Transcription)
					}

					w.logger.Infow("worker: googleOperation:", "transcription", result.Transcription, "isFinal", result.IsFinal)
				}
			}
		}
	}()

	// Send Audio Streaming
	w.logger.Infow("worker: azureOperation: G listening")
	for {
		select {
		case <-w.shut:
			w.logger.Infow("worker: azureOperation: received shut signal")
			return

		case audio := <-w.toSpeechCh:
			if err := w.azure.WriteMessage(websocket.BinaryMessage, audio); err != nil {
				w.Shutdown(fmt.Errorf("worker: azureOperation: conn.WriteMessage: %w", err))
				return
			}
		}
	}
}
