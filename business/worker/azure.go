package worker

import (
	"fmt"

	"github.com/gorilla/websocket"
)

func (w *Worker) azureOperation() {
	w.logger.Infow("worker: azureOperation: G started")
	defer w.logger.Infow("worker: azureOperation: G completed")

	azureResultCh := make(chan AzureResult)

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
			azureResultCh <- result
		}
	}(w.azure)

	go func(conn *websocket.Conn) {
		w.logger.Infow("worker: azureOperation: G started to listen for MESSAGE")
		defer w.logger.Infow("worker: azureOperation: G completed to listen for MESSAGE")

		for {
			messageType, message, err := conn.ReadMessage()
			if err != nil {
				w.Shutdown(fmt.Errorf("worker: azureOperation: G:message: conn.ReadMessage: %w", err))
				return
			}
			switch messageType {
			case websocket.CloseMessage:
				w.Shutdown(fmt.Errorf("worker: azureOperation: G:message: received close message: %s", string(message)))
				return

			case websocket.PongMessage:
				w.logger.Infow("worker: azureOperation: G:message: received pong message: %s", string(message))
			}
		}
	}(w.azure)

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
					w.logger.Infow("worker: googleOperation:", "transcription", result.Transcription, "isFinal", result.IsFinal)
				}
			}
		}
	}()

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
