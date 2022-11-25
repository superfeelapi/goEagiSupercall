package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/ardanlabs/conf/v3"
	"github.com/superfeelapi/goEagi/v2"
	"github.com/superfeelapi/goVoicebot/business/worker"
	"github.com/superfeelapi/goVoicebot/foundation/config"
	"github.com/superfeelapi/goVoicebot/foundation/logger"
)

var (
	actor     string
	version   string
	buildTime string
)

func main() {

	// =================================================================================================================
	// Configuration

	cfg := struct {
		conf.Version
		Eagi struct {
			AgiID          string
			Actor          string
			ExtensionID    string
			BoundType      string
			CampaignID     string
			CampaignConfig config.Campaign
			ConfigFilePath string `conf:"default:/etc/asterisk/ami_server.json,noprint"`
		}
		Google struct {
			PrivateKeyPath string `conf:"default:/var/lib/asterisk/agi-bin/boxwood-pilot-299014-769b582bc376.json,noprint"`
		}
		GoVad struct {
			CertFilePath string `conf:"default:/var/lib/asterisk/agi-bin/grpc/selfsigned.crt,noprint"`
			GrpcAddress  string `conf:"default:18.139.35.176:50051,noprint"`
		}
		Supercall struct {
			ApiEndpoint string `conf:"default:https://ticket-api.superceed.com:9000/socket.io/?EIO=4&transport=polling,noprint"`
		}
		Voicebot struct {
			ApiKey                       string `conf:"default:777,noprint"`
			agentVoiceEmotionEndpoint    string `conf:"default:https://voicebotapi.superceed.com/v1/voice_analysis?model=none,noprint"`
			customerVoiceEmotionEndpoint string `conf:"default:https://voicebotapi.superceed.com/v1/voice_analysis?model=emotion,noprint"`
		}
		Wauchat struct {
			TextEmotionEndpoint string `conf:"default:http://bot.superheroes.ai:5000/predict_multi/,noprint"`
		}
		Logger struct {
			LogDirectory string `conf:"default:/var/log/goEagi/campaigns/"`
		}
		Vad struct {
			AudioDir           string  `conf:"default:/tmp/goEagi/"`
			AmplitudeThreshold float64 `conf:"default:-27.5"`
		}
	}{
		Version: conf.Version{
			Build: version,
			Desc:  buildTime,
		},
	}

	// =================================================================================================================
	// Set Actor and Displaying Purpose

	cfg.Eagi.Actor = actor

	displayVersion := flag.Bool("version", false, "Display version and exit")
	flag.Parse()

	if *displayVersion {
		fmt.Printf("Actor:\t%s\n", cfg.Eagi.Actor)
		fmt.Printf("Version:\t%s\n", version)
		fmt.Printf("Build time:\t%s\n", buildTime)
		os.Exit(0)
	}

	// =================================================================================================================
	// Eagi Environment Variables

	eagi, err := goEagi.New()
	if err != nil {
		eagi.Verbose(fmt.Sprintf("ERROR: %s\n", err.Error()))
		os.Exit(1)
	}

	cfg.Eagi.ExtensionID = eagi.Env["arg_1"]
	cfg.Eagi.AgiID = eagi.Env["arg_2"]
	cfg.Eagi.CampaignID = eagi.Env["arg_3"]
	cfg.Eagi.BoundType = eagi.Env["arg_4"]

	// =================================================================================================================
	// Application Logger

	log, err := logger.New(cfg.Logger.LogDirectory, cfg.Eagi.CampaignID, cfg.Eagi.Actor)
	if err != nil {
		eagi.Verbose(fmt.Sprintf("ERROR: %s\n", err.Error()))
		os.Exit(1)
	}
	defer log.Sync()

	// =================================================================================================================
	// Campaign Configuration

	cfg.Eagi.CampaignConfig, err = config.GetCampaign(cfg.Eagi.ConfigFilePath, cfg.Eagi.CampaignID, cfg.Eagi.BoundType)
	if err != nil {
		log.Panicw("startup", "ERROR", err)
	}

	// =================================================================================================================
	// Configuration Parsing and Stringify

	_, err = conf.Parse("", &cfg)
	if err != nil {
		log.Panicw("startup", "ERROR", err)
	}

	out, err := conf.String(&cfg)
	if err != nil {
		log.Panicw("startup", "ERROR", err)
	}
	log.Infow("startup", "config", out)

	// =================================================================================================================
	// Google Speech2Text

	languageCode := config.GetLanguageCode(cfg.Eagi.CampaignConfig, cfg.Eagi.BoundType)
	speechContext := config.GetSpeechContext(cfg.Eagi.CampaignConfig, cfg.Eagi.BoundType)

	google, err := goEagi.NewGoogleService(cfg.Google.PrivateKeyPath, languageCode, nil, speechContext)
	if err != nil {
		log.Panicw("startup", "ERROR", err)
	}

	// =================================================================================================================
	// Run Worker

	workerCh := worker.Run(worker.Settings{
		Logger: log,
		Google: google,
		Config: worker.Config{
			Actor:                    strings.ToLower(cfg.Eagi.Actor),
			AgiID:                    cfg.Eagi.AgiID,
			ExtensionID:              cfg.Eagi.ExtensionID,
			Language:                 config.GetLanguage(cfg.Eagi.CampaignConfig, cfg.Eagi.BoundType),
			GrpcAddress:              cfg.GoVad.GrpcAddress,
			GrpcCertFilePath:         cfg.GoVad.CertFilePath,
			SupercallApiEndpoint:     cfg.Supercall.ApiEndpoint,
			VoicebotApiKey:           cfg.Voicebot.ApiKey,
			VoicebotAgentEndpoint:    cfg.Voicebot.agentVoiceEmotionEndpoint,
			VoicebotCustomerEndpoint: cfg.Voicebot.customerVoiceEmotionEndpoint,
			WauchatEndpoint:          cfg.Wauchat.TextEmotionEndpoint,
			AudioDir:                 cfg.Vad.AudioDir,
			AmplitudeThreshold:       cfg.Vad.AmplitudeThreshold,
		},
	})

	// Blocking main and waiting for error or shutdown.
	err = <-workerCh

	log.Infow("shutdown", "status", "shutdown started")
	defer log.Infow("shutdown", "status", "shutdown complete")

	if err != nil {
		log.Panicw("shutdown", "ERROR", err)
	}
}
