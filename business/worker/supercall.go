package worker

import (
	"time"

	"github.com/google/uuid"
	"github.com/superfeelapi/goEagiSupercall/foundation/external/supercall"
	"github.com/superfeelapi/goEagiSupercall/foundation/state"
)

const (
	interimTranscriptionID = "interimTranscription"
	fullTranscriptionID    = "fullTranscription"
)

func (w *Worker) supercallOperation() {
	w.logger.Infow("worker: supercallOperation: G started")
	defer w.logger.Infow("worker: supercallOperation: G completed")

	defer close(w.idCh)

	// Initialize Supercall's connection
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

	// Keeping the connection alive
	keepAlive := time.NewTicker(10 * time.Second)
	defer keepAlive.Stop()

	// DataID generation
	dataID := createDataID()

	w.logger.Infow("worker: supercallOperation: G listening")
	for {
		select {
		//case <-w.scamCh:
		//	w.logger.Infow("worker: supercallOperation: sending scam detected")
		//	err := s.SendData(supercall.ScamEvent, supercall.ScamData{
		//		Source:      w.config.Actor,
		//		AgiId:       w.config.AgiID,
		//		ExtensionId: w.config.ExtensionID,
		//		IsScam:      true,
		//	})
		//	if err != nil {
		//		w.logger.Errorw("worker: supercallOperation: sending scam detected", "ERROR", err)
		//	}
		//	w.logger.Infow("worker: supercallOperation: sent scam detected")

		case <-keepAlive.C:
			err := s.SendData(supercall.KeepAliveEvent, nil)
			if err != nil {
				w.Shutdown(err)
				return
			}

		case transcription := <-w.interimTranscriptCh:
			go func(transcription string) {
				var translatedTranscription string
				if w.isTranslationEnabled {
					var err error

					translatedTranscription, err = w.translation.Translate(transcription)
					if err != nil {
						w.logger.Errorw("worker: supercallOperation: translation", "ERROR", err)
					}
				}
				err := s.SendData(supercall.TranscriptEvent, supercall.TranscriptionData{
					Source:                  w.config.Actor,
					AgiId:                   w.config.AgiID,
					ExtensionId:             w.config.ExtensionID,
					DataId:                  dataID(interimTranscriptionID),
					Transcription:           transcription,
					Language:                w.config.Language,
					TranslationEnabled:      w.isTranslationEnabled,
					TranslatedTranscription: translatedTranscription,
					IsFinal:                 false,
				})
				if err != nil {
					w.Shutdown(err)
					return
				}
			}(transcription)

		case transcription := <-w.fullTranscriptCh:
			w.logger.Infow("worker: supercallOperation: sending full transcription")
			go func(transcription string) {
				var translatedTranscription string
				if w.isTranslationEnabled {
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
					Language:                w.config.Language,
					TranslationEnabled:      w.isTranslationEnabled,
					TranslatedTranscription: translatedTranscription,
					IsFinal:                 true,
				}

				err := s.SendData(supercall.TranscriptEvent, data)
				if err != nil {
					w.Shutdown(err)
					return
				}
				w.logger.Infow("worker: supercallOperation: sent full transcription")

				// ScamBot
				if w.state.Get(state.Redis) {
					if err := w.redis.Produce(data); err != nil {
						w.state.Set(state.Redis, false)
						w.logger.Errorw("worker: supercallOperation: redis", "ERROR", err)
					}
				}
			}(transcription)

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
