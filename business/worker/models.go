package worker

import (
	"github.com/superfeelapi/goEagi"
	"github.com/superfeelapi/goEagiSupercall/foundation/external/supercall"
	"github.com/superfeelapi/goEagiSupercall/foundation/redis"
	"go.uber.org/zap"
)

type Settings struct {
	Config
	Logger    *zap.SugaredLogger
	Google    *goEagi.GoogleService
	Redis     *redis.Redis
	Eagi      *goEagi.Eagi
	Supercall *supercall.Polling
}

type Config struct {
	Actor                  string
	AgiID                  string
	ExtensionID            string
	CampaignName           string
	Language               string
	Translation            bool
	SourceLanguageCode     string
	TargetLanguageCode     string
	GooglePrivateKeyPath   string
	SupercallApiEndpoint   string
	SupercallSessionID     string
	AsteriskAudioDirectory string
}

// =====================================================================================================================

type ScamData struct {
	Source string `json:"source"`
	AgiId  string `json:"agi_id"`
	IsScam bool   `json:"is_scam"`
}
