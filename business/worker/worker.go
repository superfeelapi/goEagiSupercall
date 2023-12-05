package worker

import (
	"sync"

	"github.com/superfeelapi/goEagi"
	"github.com/superfeelapi/goEagiSupercall/foundation/external/google"
	"github.com/superfeelapi/goEagiSupercall/foundation/external/voicebot"
	"github.com/superfeelapi/goEagiSupercall/foundation/external/wauchat"
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

	wg    sync.WaitGroup
	shut  chan struct{}
	error chan error

	isTranslationEnabled bool

	toGoogleCh          chan []byte
	toVadCh             chan []byte
	interimTranscriptCh chan string
	fullTranscriptCh    chan string
	wauchatTranscriptCh chan string
	paceTranscriptCh    chan int
	audioPathCh         chan string
	wauchatCh           chan wauchat.Result
	wauchatQueueCh      chan wauchat.Result
	voicebotCh          chan voicebot.Result
	grpcCh              chan bool
	idCh                chan string
	scamCh              chan bool
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
		shut:                 make(chan struct{}),
		error:                make(chan error),
		toGoogleCh:           make(chan []byte, 1000),
		toVadCh:              make(chan []byte),
		interimTranscriptCh:  make(chan string, 10),
		fullTranscriptCh:     make(chan string),
		wauchatTranscriptCh:  make(chan string, 10),
		paceTranscriptCh:     make(chan int, 10),
		audioPathCh:          make(chan string),
		wauchatCh:            make(chan wauchat.Result),
		wauchatQueueCh:       make(chan wauchat.Result, 10),
		voicebotCh:           make(chan voicebot.Result),
		grpcCh:               make(chan bool, 10),
		idCh:                 make(chan string),
		scamCh:               make(chan bool),
	}

	if w.isTranslationEnabled {
		translation, err := google.NewTranslation(s.GooglePrivateKeyPath, w.config.SourceLanguageCode, w.config.TargetLanguageCode)
		if err != nil {
			return w.error
		}
		w.translation = translation
	}

	operations := make([]func(), 8)

	if w.state.Get(state.Redis) {
		operations = append(operations, w.scamDetectOperation)
	}

	operations = append(operations, []func(){
		w.vadOperation,
		w.goVadOperation,
		w.speech2TextOperation,
		w.voiceEmotionOperation,
		w.textEmotionOperation,
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
