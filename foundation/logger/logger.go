package logger

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func New(logDirectory string, campaignID string, actor string) (*zap.SugaredLogger, error) {
	logPath := logDirectory + campaignID + "/" + actor + ".log"

	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		if err := os.MkdirAll(logPath, 0755); err != nil {
			return nil, err
		}
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
