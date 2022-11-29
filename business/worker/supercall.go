package worker

import (
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/superfeelapi/goEagiSupercall/foundation/external/supercall"
	"github.com/superfeelapi/goEagiSupercall/foundation/state"
)

func (w *Worker) supercallOperation() {
	w.logger.Infow("worker: supercallOperation: G started")
	defer w.logger.Infow("worker: supercallOperation: G completed")

	defer close(w.idCh)
	defer close(w.wauchatQueueCh)

	s := supercall.New(w.config.SupercallApiEndpoint)
	err := s.SetupConnection()
	if err != nil {
		w.Shutdown(err)
		return
	}

	w.idCh <- s.GetSessionID()

	err = s.SendData(supercall.AgiEvent, supercall.AgiData{
		Source:      w.config.Actor,
		AgiId:       w.config.AgiID,
		ExtensionId: w.config.ExtensionID,
	})
	if err != nil {
		w.Shutdown(err)
		return
	}

	keepAlive := time.NewTicker(10 * time.Second)
	defer keepAlive.Stop()

	dataID := createDataId()

	w.logger.Infow("worker: supercallOperation: G listening")
	for {
		select {
		case <-keepAlive.C:
			err := s.SendData(supercall.KeepAliveEvent, nil)
			if err != nil {
				w.Shutdown(err)
				return
			}

		case transcription := <-w.interimTranscriptCh:
			go func() {
				err := s.SendData(supercall.TranscriptEvent, supercall.TranscriptionData{
					Source:        w.config.Actor,
					AgiId:         w.config.AgiID,
					ExtensionId:   w.config.ExtensionID,
					DataId:        dataID("transcription"),
					Transcription: transcription,
					Interim:       false,
				})
				if err != nil {
					w.Shutdown(err)
					return
				}
				w.logger.Infow("worker: supercallOperation: sent interim transcription")
			}()

		case transcription := <-w.fullTranscriptCh:
			w.logger.Infow("worker: supercallOperation: sending full transcription")
			go func() {
				err := s.SendData(supercall.TranscriptEvent, supercall.TranscriptionData{
					Source:        w.config.Actor,
					AgiId:         w.config.AgiID,
					ExtensionId:   w.config.ExtensionID,
					DataId:        dataID("transcription"),
					Transcription: transcription,
					Interim:       true,
				})
				if err != nil {
					w.Shutdown(err)
					return
				}
				w.logger.Infow("worker: supercallOperation: sent full transcription")
			}()

		case textEmotion := <-w.wauchatCh:
			go func() {
				if w.state.Get(state.Voicebot) {
					w.wauchatQueueCh <- textEmotion
				} else {
					err := s.SendData(supercall.EmotionEvent, supercall.TextAndVoiceEmotionData{
						Source:      w.config.Actor,
						AgiId:       w.config.AgiID,
						ExtensionId: w.config.ExtensionID,
						DataId:      dataID("emotion"),
						TextData: supercall.TextEmotionData{
							TextEmotion:           textEmotion.Emotion.Result,
							TextEmotionConfidence: textEmotion.Emotion.Confidence,
							TextContext:           textEmotion.Context.Result,
							TextContextConfidence: textEmotion.Context.Confidence,
						},
						VoiceData: nil,
					})
					if err != nil {
						w.Shutdown(err)
						return
					}
					w.logger.Infow("worker: supercallOperation: sent text emotion")
				}
			}()

		case voiceEmotion := <-w.voicebotCh:
			w.logger.Infow("worker: supercallOperation: PROCESSING EMOTIONS")
			go func() {
				pace := <-w.paceTranscriptCh
				paceState := computePaceState(pace, voiceEmotion.AudioLength)

				if w.state.Get(state.Wauchat) {
					textEmotion := <-w.wauchatQueueCh

					w.logger.Infow("worker: supercallOperation: sending both text and voice emotions")
					switch w.config.Actor {
					case "agent":
						err := s.SendData(supercall.EmotionEvent, supercall.TextAndVoiceEmotionData{
							Source:      w.config.Actor,
							AgiId:       w.config.AgiID,
							ExtensionId: w.config.ExtensionID,
							DataId:      dataID("emotion"),
							TextData: supercall.TextEmotionData{
								TextEmotion:           textEmotion.Emotion.Result,
								TextEmotionConfidence: textEmotion.Emotion.Confidence,
								TextContext:           textEmotion.Context.Result,
								TextContextConfidence: textEmotion.Context.Confidence,
							},
							VoiceData: &supercall.VoiceEmotionData{
								VoiceAmplitude: voiceEmotion.Amplitude[0].State,
								VoicePace:      paceState,
							},
						})
						if err != nil {
							w.Shutdown(err)
							return
						}

					case "customer":
						err := s.SendData(supercall.EmotionEvent, supercall.TextAndVoiceEmotionData{
							Source:      w.config.Actor,
							AgiId:       w.config.AgiID,
							ExtensionId: w.config.ExtensionID,
							DataId:      dataID("emotion"),
							TextData: supercall.TextEmotionData{
								TextEmotion:           textEmotion.Emotion.Result,
								TextEmotionConfidence: textEmotion.Emotion.Confidence,
								TextContext:           textEmotion.Context.Result,
								TextContextConfidence: textEmotion.Context.Confidence,
							},
							VoiceData: &supercall.VoiceEmotionData{
								VoiceEmotion:           voiceEmotion.Emotion[0].Result,
								VoiceEmotionConfidence: voiceEmotion.Emotion[0].Confidence,
								VoiceAmplitude:         voiceEmotion.Amplitude[0].State,
								VoicePace:              paceState,
							},
						})
						if err != nil {
							w.Shutdown(err)
							return
						}
					}
					w.logger.Infow("worker: supercallOperation: sent both text and voice emotions")

				} else {
					w.logger.Infow("worker: supercallOperation: sending voice emotion")
					switch w.config.Actor {
					case "agent":
						err := s.SendData(supercall.EmotionEvent, supercall.TextAndVoiceEmotionData{
							Source:      w.config.Actor,
							AgiId:       w.config.AgiID,
							ExtensionId: w.config.ExtensionID,
							DataId:      dataID("emotion"),
							TextData:    supercall.TextEmotionData{},
							VoiceData: &supercall.VoiceEmotionData{
								VoiceAmplitude: voiceEmotion.Amplitude[0].State,
								VoicePace:      paceState,
							},
						})
						if err != nil {
							w.Shutdown(err)
							return
						}

					case "customer":
						err := s.SendData(supercall.EmotionEvent, supercall.TextAndVoiceEmotionData{
							Source:      w.config.Actor,
							AgiId:       w.config.AgiID,
							ExtensionId: w.config.ExtensionID,
							DataId:      dataID("emotion"),
							TextData:    supercall.TextEmotionData{},
							VoiceData: &supercall.VoiceEmotionData{
								VoiceEmotion:           voiceEmotion.Emotion[0].Result,
								VoiceEmotionConfidence: voiceEmotion.Emotion[0].Confidence,
								VoiceAmplitude:         voiceEmotion.Amplitude[0].State,
								VoicePace:              paceState,
							},
						})
						if err != nil {
							w.Shutdown(err)
							return
						}
					}
					w.logger.Infow("worker: supercallOperation: sent voice emotion")
				}
			}()

		case <-w.shut:
			w.logger.Infow("worker: supercallOperation: received shut signal")
			return
		}
	}
}

// =================================================================================================================

func createDataId() func(event string) string {
	var ids []string

	return func(event string) string {
		if event == "transcription" && len(ids) > 0 {
			return ids[0]
		}

		if event == "transcription" {
			generateId := uuid.New().String()
			ids = append(ids, generateId)
			return generateId
		}

		getID := ids[0]
		ids = ids[1:]

		return getID
	}
}

// computePaceState returns word per minute and state which based on speech rate guidelines.
func computePaceState(transcriptionLength int, audioSecond float64) string {
	wpm := wordPerMinute(transcriptionLength, audioSecond)
	if wpm > 180 {
		return "fast"
	} else if wpm < 110 {
		return "slow"
	} else {
		return "normal"
	}
}

// wordPerMinute computes word per minute.
func wordPerMinute(transcriptionLength int, audioSecond float64) float64 {
	wpm := float64(transcriptionLength) / (audioSecond / 60)
	return math.Ceil(wpm*100) / 100
}
