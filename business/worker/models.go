package worker

import (
	"github.com/gorilla/websocket"
	"github.com/superfeelapi/goEagi"
	"github.com/superfeelapi/goEagiSupercall/foundation/config"
	"github.com/superfeelapi/goEagiSupercall/foundation/external/goVad"
	"github.com/superfeelapi/goEagiSupercall/foundation/external/supercall"
	"go.uber.org/zap"
)

type Settings struct {
	Config
	Logger    *zap.SugaredLogger
	Eagi      *goEagi.Eagi
	Google    *goEagi.GoogleService
	Azure     *websocket.Conn
	Supercall *supercall.Polling
	GoVad     *goVad.Vad
	Campaign  config.Campaign
}

type Config struct {
	Actor                         string
	AgiID                         string
	ExtensionID                   string
	GooglePrivateKeyPath          string
	VoiceAnalysisApiKey           string
	VoiceAnalysisAgentEndpoint    string
	VoiceAnalysisCustomerEndpoint string
	TextAnalysisEndpoint          string
	VadAudioDir                   string
	VadAmplitudeThreshold         float64
	AsteriskAudioDirectory        string
}

// =====================================================================================================================

type AzureResult struct {
	Transcription string `json:"transcription"`
	IsFinal       bool   `json:"is_final"`
	Error         error  `json:"error"`
}
