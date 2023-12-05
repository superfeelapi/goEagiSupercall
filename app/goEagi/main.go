package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/ardanlabs/conf/v3"
	"github.com/superfeelapi/goEagi"
	"github.com/superfeelapi/goEagiSupercall/business/worker"
	"github.com/superfeelapi/goEagiSupercall/foundation/config"
	"github.com/superfeelapi/goEagiSupercall/foundation/logger"
	"github.com/superfeelapi/goEagiSupercall/foundation/redis"
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
			AgiID              string
			Actor              string
			ExtensionID        string
			BoundType          string
			CampaignID         string
			CampaignName       string
			Language           string
			LanguageCode       string
			TargetLanguageCode string
			Translation        bool
			SpeechContext      []string
			ConfigFilePath     string `conf:"default:/etc/asterisk/ami_server.json,noprint"`
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
			AgentVoiceEmotionEndpoint    string `conf:"default:https://voicebotapi.superceed.com/v1/voice_analysis?model=none,noprint"`
			CustomerVoiceEmotionEndpoint string `conf:"default:https://voicebotapi.superceed.com/v1/voice_analysis?model=emotion,noprint"`
		}
		Wauchat struct {
			TextEmotionEndpoint string `conf:"default:http://bot.superheroes.ai:4848/emotions,noprint"`
		}
		Redis struct {
			Address              string `conf:"default:redis-10106.c252.ap-southeast-1-1.ec2.cloud.redislabs.com:10106"`
			Password             string `conf:"default:dq1BygKhg4rtpmTBRlG3Rt3uh4oG0uPu"`
			TranscriptionChannel string `conf:"default:scamBot:transcription"`
			ScamBotChannel       string `conf:"default:scamBot:"`
		}
		Logger struct {
			LogDirectory string `conf:"default:/var/log/goEagi/campaigns/,noprint"`
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

	// Configuration Parsing
	_, err := conf.Parse("", &cfg)
	if err != nil {
		os.Exit(1)
	}

	// =================================================================================================================
	// Set Actor and Version Checking Support

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

	cfg.Eagi.ExtensionID = strings.TrimSpace(eagi.Env["arg_1"])
	cfg.Eagi.AgiID = strings.TrimSpace(eagi.Env["arg_2"])
	cfg.Eagi.CampaignID = strings.TrimSpace(eagi.Env["arg_3"])
	cfg.Eagi.BoundType = strings.TrimSpace(eagi.Env["arg_4"])

	// =================================================================================================================
	// Application Logger

	log, err := logger.New(cfg.Logger.LogDirectory, cfg.Eagi.CampaignID, cfg.Eagi.Actor)
	if err != nil {
		eagi.Verbose(fmt.Sprintf("ERROR: %s\n", err.Error()))
		os.Exit(1)
	}
	defer log.Sync()

	// =================================================================================================================
	// Set Campaign Configuration

	campaignConfig, err := config.GetCampaign(cfg.Eagi.ConfigFilePath, cfg.Eagi.CampaignID)
	if err != nil {
		log.Errorw("startup", "ERROR", err)
	}

	cfg.Eagi.CampaignName = config.GetCampaignName(campaignConfig)
	cfg.Eagi.Language = config.GetLanguage(campaignConfig, cfg.Eagi.BoundType)
	cfg.Eagi.LanguageCode = config.GetLanguageCode(campaignConfig, cfg.Eagi.BoundType)
	cfg.Eagi.SpeechContext = config.GetSpeechContext(campaignConfig, cfg.Eagi.BoundType)
	cfg.Eagi.TargetLanguageCode = config.GetTargetLanguageCode(campaignConfig, cfg.Eagi.BoundType)
	cfg.Eagi.Translation = config.IsTranslationEnabled(campaignConfig, cfg.Eagi.BoundType)

	// =================================================================================================================
	// Configuration Stringify

	out, err := conf.String(&cfg)
	if err != nil {
		log.Errorw("startup", "ERROR", err)
	}
	log.Infow("startup", "config", out)

	// =================================================================================================================
	// Google Speech2Text

	google, err := goEagi.NewGoogleService(cfg.Google.PrivateKeyPath, cfg.Eagi.LanguageCode, cfg.Eagi.SpeechContext)
	if err != nil {
		log.Errorw("startup", "ERROR", err)
	}

	// =================================================================================================================
	// Redis

	cfg.Redis.ScamBotChannel = fmt.Sprintf("%s%s", cfg.Redis.ScamBotChannel, cfg.Eagi.AgiID)

	redisClient, err := redis.New(cfg.Redis.Address, cfg.Redis.Password, cfg.Redis.TranscriptionChannel, cfg.Redis.ScamBotChannel, log)
	if err != nil {
		log.Errorw("startup", "ERROR", err)
	}

	// =================================================================================================================
	// Run Worker

	workerCh := worker.Run(worker.Settings{
		Logger: log,
		Google: google,
		Redis:  redisClient,
		Eagi:   eagi,
		Config: worker.Config{
			Actor:                    strings.ToLower(cfg.Eagi.Actor),
			AgiID:                    cfg.Eagi.AgiID,
			ExtensionID:              cfg.Eagi.ExtensionID,
			CampaignName:             cfg.Eagi.CampaignName,
			Language:                 cfg.Eagi.Language,
			Translation:              cfg.Eagi.Translation,
			SourceLanguageCode:       cfg.Eagi.LanguageCode,
			TargetLanguageCode:       cfg.Eagi.TargetLanguageCode,
			GooglePrivateKeyPath:     cfg.Google.PrivateKeyPath,
			GrpcAddress:              cfg.GoVad.GrpcAddress,
			GrpcCertFilePath:         cfg.GoVad.CertFilePath,
			SupercallApiEndpoint:     cfg.Supercall.ApiEndpoint,
			VoicebotApiKey:           cfg.Voicebot.ApiKey,
			VoicebotAgentEndpoint:    cfg.Voicebot.AgentVoiceEmotionEndpoint,
			VoicebotCustomerEndpoint: cfg.Voicebot.CustomerVoiceEmotionEndpoint,
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
		log.Errorw("shutdown", "ERROR", err)
	}
}
