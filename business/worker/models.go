package worker

import (
	"github.com/superfeelapi/goEagi"
	"go.uber.org/zap"
)

type Settings struct {
	Config
	Logger *zap.SugaredLogger
	Google *goEagi.GoogleService
}

type Config struct {
	Actor                    string
	AgiID                    string
	ExtensionID              string
	CampaignName             string
	Language                 string
	GrpcAddress              string
	GrpcCertFilePath         string
	SupercallApiEndpoint     string
	VoicebotApiKey           string
	VoicebotAgentEndpoint    string
	VoicebotCustomerEndpoint string
	WauchatEndpoint          string
	AudioDir                 string
	AmplitudeThreshold       float64
}
