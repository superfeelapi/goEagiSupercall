package worker

import (
	"sync"

	"github.com/superfeelapi/goEagi/v2"
	"github.com/superfeelapi/goVoicebot/foundation/pubsub"
	"github.com/superfeelapi/goVoicebot/foundation/state"
	"go.uber.org/zap"
)

const (
	audioTopic                           = "audio"
	transcriptionPaceTopic               = "transcription-pace"
	transcriptionToWauchatTopic          = "transcription-wauchat"
	interimTranscriptionToSupercallTopic = "interim-transcription-supercall"
	fullTranscriptionToSupercallTopic    = "full-transcription-supercall"
	emotionFromWauchatTopic              = "emotion-wauchat"
	emotionFromVoicebotTopic             = "emotion-voicebot"
	audioPathFromVadTopic                = "audio-path"
	vadToGrpcTopic                       = "vad-grpc"
	sessionIDFromSupercallTopic          = "id-supercall"
)

type Worker struct {
	config Config
	state  *state.State
	broker *pubsub.Broker
	logger *zap.SugaredLogger

	google *goEagi.GoogleService

	wg    sync.WaitGroup
	shut  chan struct{}
	error chan error
}

func Run(s Settings) <-chan error {
	w := &Worker{
		google: s.Google,
		state:  state.NewState(),
		logger: s.Logger,
		broker: pubsub.NewBroker(),
		shut:   make(chan struct{}),
		error:  make(chan error),
	}

	operations := []func(){
		w.vadOperation,
		w.goVadOperation,
		w.speech2TextOperation,
		w.voicebotOperation,
		w.wauchatOperation,
		w.supercallOperation,
		w.audioStreamOperation,
	}

	g := len(operations)
	w.wg.Add(g)

	hasStarted := make(chan bool)

	for _, op := range operations {
		go func(op func()) {
			defer w.wg.Done()
			hasStarted <- true
			op()
		}(op)
	}

	for i := 0; i < g; i++ {
		<-hasStarted
	}

	return w.error
}

func (w *Worker) Shutdown(err error) {
	w.logger.Infow("worker: shutdown: started")
	defer w.logger.Infow("worker: shutdown: completed")

	w.logger.Infow("worker: shutdown: terminate goroutines")
	close(w.shut)

	w.wg.Wait()

	if err != nil {
		w.error <- err
	}
}
