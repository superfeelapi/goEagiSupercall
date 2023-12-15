package worker

import (
	"sync"

	"github.com/superfeelapi/goEagi"
	"github.com/superfeelapi/goEagiSupercall/foundation/external/google"
	"github.com/superfeelapi/goEagiSupercall/foundation/external/supercall"
	"github.com/superfeelapi/goEagiSupercall/foundation/redis"
	"github.com/superfeelapi/goEagiSupercall/foundation/state"
	"go.uber.org/zap"
)

type Worker struct {
	config Config
	state  *state.State
	logger *zap.SugaredLogger

	google      *goEagi.GoogleService
	translation *google.Translation
	redis       *redis.Redis
	eagi        *goEagi.Eagi
	supercall   *supercall.Polling

	wg    sync.WaitGroup
	shut  chan struct{}
	error chan error

	isTranslationEnabled bool

	toGoogleCh          chan []byte
	interimTranscriptCh chan string
	fullTranscriptCh    chan string
}

func Run(s Settings) <-chan error {
	w := &Worker{
		config:               s.Config,
		state:                state.NewState(),
		logger:               s.Logger,
		google:               s.Google,
		isTranslationEnabled: s.Translation,
		redis:                s.Redis,
		eagi:                 s.Eagi,
		supercall:            s.Supercall,
		shut:                 make(chan struct{}),
		error:                make(chan error),
		toGoogleCh:           make(chan []byte, 4096),
		interimTranscriptCh:  make(chan string, 10),
		fullTranscriptCh:     make(chan string),
	}

	if w.isTranslationEnabled {
		translation, err := google.NewTranslation(s.GooglePrivateKeyPath, w.config.SourceLanguageCode, w.config.TargetLanguageCode)
		if err != nil {
			return w.error
		}
		w.translation = translation
	}

	operations := make([]func(), 0)

	if w.state.Get(state.Redis) {
		operations = append(operations, w.scamDetectOperation)
	}

	operations = append(operations, []func(){
		w.speech2TextOperation,
		w.supercallOperation,
		w.audioStreamOperation,
	}...)

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

	w.logger.Errorw("worker: shutdown", "ERROR", err)
	w.logger.Infow("worker: shutdown: terminate goroutines")
	close(w.shut)

	w.wg.Wait()

	if err != nil {
		w.error <- err
	}
}
