package worker

import (
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/superfeelapi/goEagiSupercall/foundation/external/supercall"
)

const (
	interimTranscriptionID = "interimTranscription"
	fullTranscriptionID    = "fullTranscription"
	TextEmotionID          = "textEmotion"
	voiceEmotionID         = "voiceEmotion"
)

func (w *Worker) supercallOperation() {
	w.logger.Infow("worker: supercallOperation: G started")
	defer w.logger.Infow("worker: supercallOperation: G completed")

	err := w.supercall.SendData(supercall.AgiEvent, supercall.AgiData{
		Source:      w.config.Actor,
		AgiId:       w.config.AgiID,
		ExtensionId: w.config.ExtensionID,
	})
	if err != nil {
		w.Shutdown(err)
		return
	}

	// Keeping the connection alive
	keepAlive := time.NewTicker(10 * time.Second)
	defer keepAlive.Stop()

	// DataID generation
	dataID := createDataID()

	w.logger.Infow("worker: supercallOperation: G listening")
	for {
		select {
		case <-keepAlive.C:
			err := w.supercall.SendData(supercall.KeepAliveEvent, nil)
			if err != nil {
				w.Shutdown(err)
				return
			}

		case transcription := <-w.interimTranscriptCh:
			go func() {
				var translatedTranscription string
				var translationEnabled bool

				// Check if translation is enabled
				if w.translation != nil {
					translationEnabled = true
					var err error

					translatedTranscription, err = w.translation.Translate(transcription)
					if err != nil {
						w.logger.Errorw("worker: supercallOperation: translation", "ERROR", err)
					}
				}

				err := w.supercall.SendData(supercall.TranscriptEvent, supercall.TranscriptionData{
					Source:                  w.config.Actor,
					AgiId:                   w.config.AgiID,
					ExtensionId:             w.config.ExtensionID,
					DataId:                  dataID(interimTranscriptionID),
					Transcription:           transcription,
					TranslationEnabled:      translationEnabled,
					TranslatedTranscription: translatedTranscription,
					IsFinal:                 false,
				})
				if err != nil {
					w.Shutdown(err)
					return
				}
			}()

		case transcription := <-w.fullTranscriptCh:
			w.logger.Infow("worker: supercallOperation: sending full transcription")
			go func() {
				var translatedTranscription string
				var translationEnabled bool

				// Check if translation is enabled
				if w.translation != nil {
					translationEnabled = true
					var err error

					translatedTranscription, err = w.translation.Translate(transcription)
					if err != nil {
						w.logger.Errorw("worker: supercallOperation: translation", "ERROR", err)
					}
				}

				data := supercall.TranscriptionData{
					Source:                  w.config.Actor,
					AgiId:                   w.config.AgiID,
					ExtensionId:             w.config.ExtensionID,
					DataId:                  dataID(fullTranscriptionID),
					Transcription:           transcription,
					TranslationEnabled:      translationEnabled,
					TranslatedTranscription: translatedTranscription,
					IsFinal:                 true,
				}

				err := w.supercall.SendData(supercall.TranscriptEvent, data)
				if err != nil {
					w.Shutdown(err)
					return
				}
				w.logger.Infow("worker: supercallOperation: sent full transcription")
			}()

		case textEmotion := <-w.textAnalysisCh:
			w.logger.Infow("worker: supercallOperation: sending text emotion")
			go func() {
				err := w.supercall.SendData(supercall.TextEmotionEvent, supercall.TextEmotionData{
					Source:      w.config.Actor,
					AgiId:       w.config.AgiID,
					ExtensionId: w.config.ExtensionID,
					DataId:      dataID(TextEmotionID),
					TextEmotion: textEmotion.TextEmotion,
				})
				if err != nil {
					w.Shutdown(err)
					return
				}
				w.logger.Infow("worker: supercallOperation: sent text emotion")
			}()

		case voiceEmotion := <-w.voiceAnalysisCh:
			w.logger.Infow("worker: supercallOperation: sending voice emotion")
			go func() {
				pace := <-w.paceTranscriptCh
				paceState := computePaceState(pace, voiceEmotion.AudioLength)

				switch w.config.Actor {
				case "agent":
					err := w.supercall.SendData(supercall.VoiceEmotionEvent, supercall.VoiceEmotionData{
						Source:         w.config.Actor,
						AgiId:          w.config.AgiID,
						ExtensionId:    w.config.ExtensionID,
						DataId:         dataID(voiceEmotionID),
						VoiceAmplitude: voiceEmotion.Amplitude[0].State,
						VoicePace:      paceState,
					})
					if err != nil {
						w.Shutdown(err)
						return
					}

				case "customer":
					err := w.supercall.SendData(supercall.VoiceEmotionEvent, supercall.VoiceEmotionData{
						Source:                 w.config.Actor,
						AgiId:                  w.config.AgiID,
						ExtensionId:            w.config.ExtensionID,
						DataId:                 dataID(voiceEmotionID),
						VoiceAmplitude:         voiceEmotion.Amplitude[0].State,
						VoicePace:              paceState,
						VoiceEmotion:           voiceEmotion.Emotion[0].Result,
						VoiceEmotionConfidence: voiceEmotion.Emotion[0].Confidence,
					})
					if err != nil {
						w.Shutdown(err)
						return
					}
				}
				w.logger.Infow("worker: supercallOperation: sent voice emotion")
			}()

		case <-w.shut:
			w.logger.Infow("worker: supercallOperation: received shut signal")
			return
		}
	}
}

// =================================================================================================================

func createDataID() func(event string) string {
	ids := NewDataIDs()

	return func(event string) string {
		switch event {
		case "interimTranscription":
			return ids.Peek(event)

		case "fullTranscription":
			id := ids.Dequeue(event)
			_ = ids.Dequeue("interimTranscription")
			ids.CreateNewID()
			return id

		default:
			return ids.Dequeue(event)
		}
	}
}

type DataIDs struct {
	elements map[string][]string
}

func NewDataIDs() *DataIDs {
	generateId := uuid.New().String()
	d := DataIDs{
		elements: map[string][]string{
			interimTranscriptionID: {generateId},
			fullTranscriptionID:    {generateId},
			TextEmotionID:          {generateId},
			voiceEmotionID:         {generateId},
		},
	}
	return &d
}

func (d *DataIDs) CreateNewID() {
	generateId := uuid.New().String()
	for i, _ := range d.elements {
		d.elements[i] = append(d.elements[i], generateId)
	}
}

func (d *DataIDs) Dequeue(event string) string {
	getElement := d.elements[event][0]
	d.elements[event] = d.elements[event][1:]
	return getElement
}
func (d *DataIDs) Peek(event string) string {
	return d.elements[event][0]
}

// =================================================================================================================

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
