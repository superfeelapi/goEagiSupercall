package worker

import (
	"time"

	"github.com/superfeelapi/goEagi"
	"github.com/superfeelapi/goEagiSupercall/foundation/state"
)

func (w *Worker) vadOperation() {
	w.logger.Infow("worker: vadOperation: G started")
	defer w.logger.Infow("worker: vadOperation: G completed")

	defer close(w.audioPathCh)
	defer close(w.grpcCh)

	audioDir := w.config.AudioDir + w.config.CampaignName + "/"

	var latestFrame []byte
	var speechFrame []byte
	var isSpeech bool

	startTime := time.Now()
	endTime := time.Duration(1) * time.Second

	w.logger.Infow("worker: vadOperation: G listening")
	for {
		select {
		case audio := <-w.toVadCh:
			latestFrame = append(latestFrame, audio...)

			if time.Since(startTime) > endTime {
				amp, err := goEagi.ComputeAmplitude(latestFrame)
				if err != nil {
					w.Shutdown(err)
					return
				}

				switch amp > w.config.AmplitudeThreshold {

				case true:
					if w.state.Get(state.GoVad) {
						w.grpcCh <- true
					}
					isSpeech = true
					speechFrame = append(speechFrame, latestFrame...)

				case false:
					if w.state.Get(state.GoVad) {
						w.grpcCh <- false
					}
					if isSpeech {
						if w.state.Get(state.Voicebot) {
							audioFile := createAudioFile(w.config.AgiID)
							audioFilepath, err := goEagi.GenerateAudio(speechFrame, audioDir, audioFile)
							if err != nil {
								w.Shutdown(err)
								return
							}

							w.audioPathCh <- audioFilepath
							w.logger.Infow("worker: vadOperation: SENT AUDIO FILE")
						}
						speechFrame = nil
						isSpeech = false
					}
				}
				latestFrame = nil
				startTime = time.Now()
			}

		case <-w.shut:
			w.logger.Infow("worker: vadOperation: received shut signal")
			return
		}
	}
}

// =================================================================================================================

func createAudioFile(agiID string) string {
	const layout = "2006-01-02-15:04:05"
	t := time.Now()

	return agiID + "-" + t.Format(layout) + ".wav"
}
