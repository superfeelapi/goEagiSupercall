package worker

import (
	"sync"

	"github.com/gorilla/websocket"
	"github.com/superfeelapi/goEagi"
	"github.com/superfeelapi/goEagiSupercall/foundation/config"
	"github.com/superfeelapi/goEagiSupercall/foundation/external/google"
	"github.com/superfeelapi/goEagiSupercall/foundation/external/supercall"
	"github.com/superfeelapi/goEagiSupercall/foundation/redis"
	"github.com/superfeelapi/goEagiSupercall/foundation/state"
	"go.uber.org/zap"
)

type Worker struct {
	config Config
	state  *state.State

	logger      *zap.SugaredLogger
	eagi        *goEagi.Eagi
	google      *goEagi.GoogleService
	azure       *websocket.Conn
	redis       *redis.Redis
	supercall   *supercall.Polling
	campaign    config.Campaign
	translation *google.Translation

	wg    sync.WaitGroup
	shut  chan struct{}
	error chan error

	toSpeechCh          chan []byte
	toScamCh            chan bool
	interimTranscriptCh chan string
	fullTranscriptCh    chan string
}

func Run(s Settings) <-chan error {
	w := &Worker{
		config:              s.Config,
		state:               state.NewState(),
		logger:              s.Logger,
		eagi:                s.Eagi,
		google:              s.Google,
		azure:               s.Azure,
		redis:               s.Redis,
		supercall:           s.Supercall,
		campaign:            s.Campaign,
		shut:                make(chan struct{}),
		error:               make(chan error),
		toSpeechCh:          make(chan []byte, 4096),
		interimTranscriptCh: make(chan string, 10),
		fullTranscriptCh:    make(chan string),
	}

	if w.campaign.Translation.InUse {
		translation, err := google.NewTranslation(s.GooglePrivateKeyPath, w.campaign.Translation.Target)
		if err != nil {
			return w.error
		}
		w.translation = translation
	}

	operations := make([]func(), 0)

	operations = append(operations, []func(){
		w.supercallOperation,
		w.audioStreamOperation,
	}...)

	if w.campaign.Scam.InUse {
		w.logger.Infow("worker: scam: enabled")
		operations = append(operations, w.scamDetectOperation)
		w.toScamCh = make(chan bool)
	}

	if w.google != nil {
		w.logger.Infow("worker: google: enabled")
		operations = append(operations, w.googleOperation)
	}

	if w.azure != nil {
		w.logger.Infow("worker: azure: enabled")
		// underlying azureOperation will spawn another 3 goroutines
		operations = append(operations, w.azureOperation)
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

	w.logger.Errorw("worker: shutdown", "ERROR", err)
	w.logger.Infow("worker: shutdown: terminate goroutines")
	close(w.shut)

	w.wg.Wait()

	if err != nil {
		w.error <- err
	}
}
