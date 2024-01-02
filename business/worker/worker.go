package worker

import (
	"sync"

	"github.com/gorilla/websocket"
	"github.com/superfeelapi/goEagi"
	"github.com/superfeelapi/goEagiSupercall/foundation/config"
	"github.com/superfeelapi/goEagiSupercall/foundation/external/goVad"
	"github.com/superfeelapi/goEagiSupercall/foundation/external/google"
	"github.com/superfeelapi/goEagiSupercall/foundation/external/supercall"
	"github.com/superfeelapi/goEagiSupercall/foundation/external/textAnalysis"
	"github.com/superfeelapi/goEagiSupercall/foundation/external/voiceAnalysis"
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
	supercall   *supercall.Polling
	goVad       *goVad.Vad
	campaign    config.Campaign
	translation *google.Translation

	wg    sync.WaitGroup
	shut  chan struct{}
	error chan error

	toSpeechCh chan []byte
	toVadCh    chan []byte
	toGoVadCh  chan bool

	interimTranscriptCh     chan string
	fullTranscriptCh        chan string
	textEmotionTranscriptCh chan string

	paceTranscriptCh chan int
	audioPathCh      chan string
	textAnalysisCh   chan textAnalysis.Result
	voiceAnalysisCh  chan voiceAnalysis.Result
}

func Run(s Settings) <-chan error {
	w := &Worker{
		config:                  s.Config,
		state:                   state.NewState(),
		logger:                  s.Logger,
		eagi:                    s.Eagi,
		google:                  s.Google,
		azure:                   s.Azure,
		supercall:               s.Supercall,
		goVad:                   s.GoVad,
		campaign:                s.Campaign,
		shut:                    make(chan struct{}),
		error:                   make(chan error),
		toSpeechCh:              make(chan []byte, 4096),
		toVadCh:                 make(chan []byte),
		toGoVadCh:               make(chan bool, 10),
		interimTranscriptCh:     make(chan string, 10),
		fullTranscriptCh:        make(chan string),
		textEmotionTranscriptCh: make(chan string, 10),
		paceTranscriptCh:        make(chan int, 10),
		audioPathCh:             make(chan string),
		textAnalysisCh:          make(chan textAnalysis.Result),
		voiceAnalysisCh:         make(chan voiceAnalysis.Result),
	}

	// Translation
	if w.campaign.Translation.InUse {
		translation, err := google.NewTranslation(s.GooglePrivateKeyPath, w.campaign.Translation.Target)
		if err != nil {
			return w.error
		}
		w.translation = translation
	}

	// Operations
	operations := make([]func(), 0)

	if w.google != nil {
		operations = append(operations, w.googleOperation)
	}

	if w.azure != nil {
		// underlying azureOperation will spawn another 3 goroutines
		operations = append(operations, w.azureOperation)
	}

	operations = append(operations, []func(){
		w.vadOperation,
		w.goVadOperation,
		w.audioStreamOperation,
		w.supercallOperation,
		w.voiceEmotionOperation,
		w.textEmotionOperation,
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
