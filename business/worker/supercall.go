package worker

import (
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/superfeelapi/goVoicebot/foundation/external/supercall"
	"github.com/superfeelapi/goVoicebot/foundation/external/voicebot"
	"github.com/superfeelapi/goVoicebot/foundation/external/wauchat"
	"github.com/superfeelapi/goVoicebot/foundation/pubsub"
	"github.com/superfeelapi/goVoicebot/foundation/state"
)

func (w *Worker) supercallOperation() {
	w.logger.Infow("worker: supercallOperation: G started")
	defer w.logger.Infow("worker: supercallOperation: G completed")

	paceSub := pubsub.NewSubscriber(10)
	w.broker.Subscribe(transcriptionPaceTopic, paceSub)
	defer w.broker.UnSubscribe(transcriptionPaceTopic, paceSub)

	interimTranscriptionSub := pubsub.NewSubscriber(0)
	w.broker.Subscribe(interimTranscriptionToSupercallTopic, interimTranscriptionSub)
	defer w.broker.UnSubscribe(interimTranscriptionToSupercallTopic, interimTranscriptionSub)

	fullTranscriptionSub := pubsub.NewSubscriber(0)
	w.broker.Subscribe(fullTranscriptionToSupercallTopic, fullTranscriptionSub)
	defer w.broker.UnSubscribe(fullTranscriptionToSupercallTopic, fullTranscriptionSub)

	textEmotionSub := pubsub.NewSubscriber(10)
	w.broker.Subscribe(emotionFromWauchatTopic, textEmotionSub)
	defer w.broker.UnSubscribe(emotionFromWauchatTopic, textEmotionSub)

	voiceEmotionSub := pubsub.NewSubscriber(0)
	w.broker.Subscribe(emotionFromVoicebotTopic, voiceEmotionSub)
	defer w.broker.UnSubscribe(emotionFromVoicebotTopic, voiceEmotionSub)

	paceCh := paceSub.GetChannel()
	interimTranscriptionCh := interimTranscriptionSub.GetChannel()
	fullTranscriptionCh := fullTranscriptionSub.GetChannel()
	textEmotionCh := textEmotionSub.GetChannel()
	voiceEmotionCh := voiceEmotionSub.GetChannel()

	s := supercall.New(w.config.SupercallApiEndpoint)
	err := s.SetupConnection()
	if err != nil {
		w.Shutdown(err)
		return
	}

	err = w.broker.Publish(sessionIDFromSupercallTopic, s.GetSessionID())
	if err != nil {
		w.Shutdown(err)
		return
	}

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
	emotions := wauchat.NewQueue()

	for {
		select {
		case <-keepAlive.C:
			err := s.SendData(supercall.KeepAliveEvent, nil)
			if err != nil {
				w.Shutdown(err)
				return
			}

		case transcription := <-interimTranscriptionCh:
			go func() {
				err := s.SendData(supercall.TranscriptEvent, supercall.TranscriptionData{
					Source:        w.config.Actor,
					AgiId:         w.config.AgiID,
					ExtensionId:   w.config.ExtensionID,
					DataId:        dataID("transcription"),
					Transcription: transcription.(string),
					Interim:       true,
				})
				if err != nil {
					w.Shutdown(err)
					return
				}
			}()

		case transcription := <-fullTranscriptionCh:
			err := s.SendData(supercall.TranscriptEvent, supercall.TranscriptionData{
				Source:        w.config.Actor,
				AgiId:         w.config.AgiID,
				ExtensionId:   w.config.ExtensionID,
				DataId:        dataID("transcription"),
				Transcription: transcription.(string),
				Interim:       false,
			})
			if err != nil {
				w.Shutdown(err)
				return
			}

		case textEmotion := <-textEmotionCh:
			go func() {
				if w.state.Get(state.Voicebot) {
					emotions.Enqueue(textEmotion.(wauchat.Result))
				} else {
					err := s.SendData(supercall.EmotionEvent, supercall.TextAndVoiceEmotionData{
						Source:      w.config.Actor,
						AgiId:       w.config.AgiID,
						ExtensionId: w.config.ExtensionID,
						DataId:      dataID("emotion"),
						TextData: supercall.TextEmotionData{
							TextEmotion:           textEmotion.(wauchat.Result).Emotion.Result,
							TextEmotionConfidence: textEmotion.(wauchat.Result).Emotion.Confidence,
							TextContext:           textEmotion.(wauchat.Result).Context.Result,
							TextContextConfidence: textEmotion.(wauchat.Result).Context.Confidence,
						},
						VoiceData: nil,
					})
					if err != nil {
						w.Shutdown(err)
						return
					}
				}
			}()

		case voiceEmotion := <-voiceEmotionCh:
			go func() {
				pace := <-paceCh
				paceState := computePaceState(pace.(int), voiceEmotion.(voicebot.Result).AudioLength)

				if w.state.Get(state.Wauchat) {
					textEmotion, err := emotions.Dequeue()
					if err != nil {
						w.Shutdown(err)
						return
					}

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
								VoiceAmplitude: voiceEmotion.(voicebot.Result).Amplitude[0].State,
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
								VoiceEmotion:           voiceEmotion.(voicebot.Result).Emotion[0].Result,
								VoiceEmotionConfidence: voiceEmotion.(voicebot.Result).Emotion[0].Confidence,
								VoiceAmplitude:         voiceEmotion.(voicebot.Result).Amplitude[0].State,
								VoicePace:              paceState,
							},
						})
						if err != nil {
							w.Shutdown(err)
							return
						}
					}
				} else {
					switch w.config.Actor {
					case "agent":
						err := s.SendData(supercall.EmotionEvent, supercall.TextAndVoiceEmotionData{
							Source:      w.config.Actor,
							AgiId:       w.config.AgiID,
							ExtensionId: w.config.ExtensionID,
							DataId:      dataID("emotion"),
							TextData:    supercall.TextEmotionData{},
							VoiceData: &supercall.VoiceEmotionData{
								VoiceAmplitude: voiceEmotion.(voicebot.Result).Amplitude[0].State,
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
								VoiceEmotion:           voiceEmotion.(voicebot.Result).Emotion[0].Result,
								VoiceEmotionConfidence: voiceEmotion.(voicebot.Result).Emotion[0].Confidence,
								VoiceAmplitude:         voiceEmotion.(voicebot.Result).Amplitude[0].State,
								VoicePace:              paceState,
							},
						})
						if err != nil {
							w.Shutdown(err)
							return
						}
					}
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
