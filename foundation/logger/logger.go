package logger

import (
	"os"
	"path/filepath"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func New(logDirectory string, campaignID string, actor string) (*zap.SugaredLogger, error) {
	logCampaignDirectory := filepath.Join(logDirectory, campaignID)
	logPath := filepath.Join(logCampaignDirectory, actor+".log")

	if _, err := os.Stat(logCampaignDirectory); os.IsNotExist(err) {
		if err := os.MkdirAll(logCampaignDirectory, os.ModePerm); err != nil {
			return nil, err
		}
	}

	_, err := os.OpenFile(logPath, os.O_CREATE|os.O_RDWR, os.ModePerm)
	if err != nil {
		return nil, err
	}

	config := zap.NewProductionConfig()
	config.OutputPaths = []string{logPath}
	config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	config.DisableStacktrace = false

	log, err := config.Build()
	if err != nil {
		return nil, err
	}

	return log.Sugar(), nil
}
