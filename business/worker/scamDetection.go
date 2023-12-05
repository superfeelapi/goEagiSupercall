package worker

import (
	"encoding/json"
	"fmt"
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
				w.scamCh <- true
				audioName := fmt.Sprintf("%s-%s", scamAudioName, w.config.SourceLanguageCode)

				switch data.Source {
				case agent:
					if w.config.Actor == agent {
						_, err := w.eagi.StreamFile(audioName, "en")
						if err != nil {
							w.logger.Errorw("worker: scamDetectOperation", "streamFile", err)
						}
					}

				case customer:
					if w.config.Actor == customer {
						_, err := w.eagi.StreamFile(audioName, "en")
						if err != nil {
							w.logger.Errorw("worker: scamDetectOperation", "streamFile", err)
						}
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
