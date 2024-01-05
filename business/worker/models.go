package worker

import (
	"github.com/gorilla/websocket"
	"github.com/superfeelapi/goEagi"
	"github.com/superfeelapi/goEagiSupercall/foundation/config"
	"github.com/superfeelapi/goEagiSupercall/foundation/external/supercall"
	"github.com/superfeelapi/goEagiSupercall/foundation/redis"
	"go.uber.org/zap"
)

type Settings struct {
	Config
	Logger    *zap.SugaredLogger
	Eagi      *goEagi.Eagi
	Google    *goEagi.GoogleService
	Azure     *websocket.Conn
	Redis     *redis.Redis
	Supercall *supercall.Polling
	Campaign  config.Campaign
}

type Config struct {
	Actor                  string
	AgiID                  string
	ExtensionID            string
	GooglePrivateKeyPath   string
	AsteriskAudioDirectory string
}

// =====================================================================================================================

type AzureResult struct {
	Transcription string `json:"transcription"`
	IsFinal       bool   `json:"is_final"`
	Error         error  `json:"error"`
}

// =====================================================================================================================

type ScamData struct {
	Source string `json:"source"`
	AgiId  string `json:"agi_id"`
	IsScam bool   `json:"is_scam"`
}
