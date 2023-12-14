package worker

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const (
	agent    = "agent"
	customer = "customer"

	scamAudioName = "scamDetected"
)

func (w *Worker) scamDetectOperation() {
	w.logger.Infow("worker: scamDetectOperation: G started")
	defer w.logger.Infow("worker: scamDetectOperation: G completed")
	defer w.redis.Client.Close()

	msgCh := w.redis.ConsumeScamBotChannel()

	w.logger.Infow("worker: scamDetectOperation: G listening")
	for {
		select {
		case message := <-msgCh:
			var data ScamData
			if err := json.Unmarshal([]byte(message.Payload), &data); err != nil {
				w.Shutdown(err)
				return
			}
			if data.IsScam {
				w.logger.Infow("worker: scamDetectOperation: SCAM DETECTED", "data", data)
				//w.scamCh <- true
				audioName := fmt.Sprintf("%s-%s", scamAudioName, w.config.SourceLanguageCode)

				if !checkIfFileExists(w.config.AsteriskAudioDirectory, audioName) {
					audioName = fmt.Sprintf("%s-%s", scamAudioName, w.config.Language)
				}

				w.logger.Infow("worker: scamDetectOperation", "audioName", audioName, "source", data.Source, "isScam", data.IsScam)

				switch data.Source {
				case agent:
					if w.config.Actor == agent {
						_, err := w.eagi.StreamFile(audioName, "1")
						if err != nil {
							w.logger.Errorw("worker: scamDetectOperation", "streamFile", err)
						}
						w.logger.Infow("worker: scamDetectOperation", "streamFile", audioName, "source", agent)
					}

				case customer:
					if w.config.Actor == customer {
						_, err := w.eagi.StreamFile(audioName, "1")
						if err != nil {
							w.logger.Errorw("worker: scamDetectOperation", "streamFile", err)
						}
						w.logger.Infow("worker: scamDetectOperation", "streamFile", audioName, "source", customer)
					}

				default:
					w.logger.Errorw("worker: scamDetectOperation: unknown source", "source", data.Source)
				}
			}

		case <-w.shut:
			w.logger.Infow("worker: scamDetectOperation: received shut signal")
			return
		}
	}
}

// =====================================================================================================================

func checkIfFileExists(audioDirectory, audioName string) bool {
	audioName = fmt.Sprintf("%s.wav", audioName)
	audiopath := filepath.Join(audioDirectory, audioName)
	if _, err := os.Stat(audiopath); os.IsNotExist(err) {
		return false
	}
	return true
}