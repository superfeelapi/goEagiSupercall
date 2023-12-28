package worker

import (
	"encoding/json"
	"fmt"
	"path/filepath"
)

const (
	agent    = "agent"
	customer = "customer"
)

var audioFilenamePattern = "scamDetected-%s.wav"

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
				w.toScamCh <- true

				w.logger.Infow("worker: scamDetectOperation", "audioName", w.campaign.Scam.AudioPath, "source", data.Source, "isScam", data.IsScam)

				switch data.Source {
				case agent:
					if w.config.Actor == agent {
						if w.campaign.Scam.InUse {
							_, err := w.eagi.StreamFile(w.campaign.Scam.AudioPath, "1")
							if err != nil {
								w.logger.Errorw("worker: scamDetectOperation", "streamFile", err)
							}
							w.logger.Infow("worker: scamDetectOperation", "streamFile", w.campaign.Scam.AudioPath, "source", agent)

						} else {
							var audioPath string
							if w.campaign.Inbound.Azure.InUse {
								languageCode := w.campaign.Inbound.Azure.LanguageCode[0]
								audioName := fmt.Sprintf(audioFilenamePattern, languageCode)
								filepath.Join(w.config.AsteriskAudioDirectory, audioName)
							}
							if w.campaign.Inbound.Google.InUse {
								languageCode := w.campaign.Inbound.Google.LanguageCode
								audioName := fmt.Sprintf(audioFilenamePattern, languageCode)
								filepath.Join(w.config.AsteriskAudioDirectory, audioName)
							}
							_, err := w.eagi.StreamFile(audioPath, "1")
							if err != nil {
								w.logger.Errorw("worker: scamDetectOperation", "streamFile", err)
							}
							w.logger.Infow("worker: scamDetectOperation", "streamFile", audioPath, "source", agent)
						}
					}

				case customer:
					if w.config.Actor == customer {
						if w.campaign.Scam.InUse {
							_, err := w.eagi.StreamFile(w.campaign.Scam.AudioPath, "1")
							if err != nil {
								w.logger.Errorw("worker: scamDetectOperation", "streamFile", err)
							}
							w.logger.Infow("worker: scamDetectOperation", "streamFile", w.campaign.Scam.AudioPath, "source", customer)

						} else {
							var audioPath string
							if w.campaign.Inbound.Azure.InUse {
								languageCode := w.campaign.Inbound.Azure.LanguageCode[0]
								audioName := fmt.Sprintf(audioFilenamePattern, languageCode)
								filepath.Join(w.config.AsteriskAudioDirectory, audioName)
							}
							if w.campaign.Inbound.Google.InUse {
								languageCode := w.campaign.Inbound.Google.LanguageCode
								audioName := fmt.Sprintf(audioFilenamePattern, languageCode)
								filepath.Join(w.config.AsteriskAudioDirectory, audioName)
							}
							_, err := w.eagi.StreamFile(audioPath, "1")
							if err != nil {
								w.logger.Errorw("worker: scamDetectOperation", "streamFile", err)
							}
							w.logger.Infow("worker: scamDetectOperation", "streamFile", audioPath, "source", customer)
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
