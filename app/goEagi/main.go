package main

import (
	"flag"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/ardanlabs/conf/v3"
	"github.com/gorilla/websocket"
	"github.com/superfeelapi/goEagi"
	"github.com/superfeelapi/goEagiSupercall/business/worker"
	"github.com/superfeelapi/goEagiSupercall/foundation/config"
	"github.com/superfeelapi/goEagiSupercall/foundation/external/supercall"
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
			AgiID          string
			Actor          string
			ExtensionID    string
			BoundType      string
			CampaignID     string
			ConfigFilePath string `conf:"default:/etc/asterisk/ami_server.json,noprint"`
		}
		Google struct {
			PrivateKeyPath string `conf:"default:/var/lib/asterisk/agi-bin/boxwood-pilot-299014-769b582bc376.json,noprint"`
		}
		Websocket struct {
			Scheme string `conf:"default:ws"`
			Host   string `conf:"default:20.2.83.74:8080"`
			Path   string `conf:"default:/azure"`
			ApiKey string `conf:"default:cp132465"`
		}
		Supercall struct {
			ApiEndpoint string `conf:"default:https://ticket-api.superceed.com:9000/socket.io/?EIO=4&transport=polling,noprint"`
			ApiToken    string `conf:"default:TxbA20O4S0KO"`
		}
		Redis struct {
			Address              string `conf:"default:redis-10106.c252.ap-southeast-1-1.ec2.cloud.redislabs.com:10106"`
			Password             string `conf:"default:dq1BygKhg4rtpmTBRlG3Rt3uh4oG0uPu"`
			TranscriptionChannel string `conf:"default:scamBot:transcription"`
			ScamBotChannel       string `conf:"default:scamBot:"`
		}
		Asterisk struct {
			AudioDirectory string `conf:"default:/var/lib/asterisk/sounds/en/"`
		}
		Logger struct {
			LogDirectory string `conf:"default:/var/log/goEagi/campaigns/,noprint"`
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

	// =================================================================================================================
	// Configuration Stringify

	out, err := conf.String(&cfg)
	if err != nil {
		log.Errorw("startup", "ERROR", err)
	}
	log.Infow("startup", "config", out)

	// =================================================================================================================
	// Redis

	cfg.Redis.ScamBotChannel = fmt.Sprintf("%s%s", cfg.Redis.ScamBotChannel, cfg.Eagi.AgiID)

	redisClient, err := redis.New(cfg.Redis.Address, cfg.Redis.Password, cfg.Redis.TranscriptionChannel, cfg.Redis.ScamBotChannel, log)
	if err != nil {
		log.Errorw("startup", "ERROR", err)
	}

	// =================================================================================================================
	// Supercall

	superCall := supercall.New(cfg.Supercall.ApiEndpoint, cfg.Supercall.ApiToken)
	err = superCall.SetupConnection()
	if err != nil {
		log.Errorw("startup", "ERROR", err)
	}

	// =================================================================================================================
	// Speech2Text

	var google *goEagi.GoogleService
	var azure *websocket.Conn

	// Google Speech2Text
	googleInUse, err := campaignConfig.IsGoogleInUse(cfg.Eagi.BoundType)
	if err != nil {
		log.Errorw("startup", "ERROR", err)
	}
	if googleInUse {
		languageCode, err := campaignConfig.GetGoogleLanguageCode(cfg.Eagi.BoundType)
		if err != nil {
			log.Errorw("startup", "ERROR", err)
		}

		speechContext, err := campaignConfig.GetGoogleSpeechContext(cfg.Eagi.BoundType)
		if err != nil {
			log.Errorw("startup", "ERROR", err)
		}

		google, err = goEagi.NewGoogleService(cfg.Google.PrivateKeyPath, languageCode, speechContext)
		if err != nil {
			log.Errorw("startup", "ERROR", err)
		}
	}

	// Azure Speech2Text
	azureInUse, err := campaignConfig.IsAzureInUse(cfg.Eagi.BoundType)
	if err != nil {
		log.Errorw("startup", "ERROR", err)
	}
	if azureInUse {
		u := url.URL{
			Scheme: cfg.Websocket.Scheme,
			Host:   cfg.Websocket.Host,
			Path:   cfg.Websocket.Path,
		}

		azure, _, err = websocket.DefaultDialer.Dial(u.String(), http.Header{"api-key": []string{cfg.Websocket.ApiKey}})
		if err != nil {
			log.Errorw("startup", "ERROR", err)
		}

		languageCode, err := campaignConfig.GetAzureLanguageCode(cfg.Eagi.BoundType)
		if err != nil {
			log.Errorw("startup", "ERROR", err)
		}

		registerData := struct {
			LanguageCode []string
		}{
			LanguageCode: languageCode,
		}

		if err := azure.WriteJSON(registerData); err != nil {
			log.Errorw("startup", "ERROR", err)
		}
	}

	// =================================================================================================================
	// Run Worker

	workerCh := worker.Run(worker.Settings{
		Logger:    log,
		Eagi:      eagi,
		Google:    google,
		Azure:     azure,
		Redis:     redisClient,
		Supercall: superCall,
		Campaign:  campaignConfig,
		Config: worker.Config{
			Actor:                  strings.ToLower(cfg.Eagi.Actor),
			AgiID:                  cfg.Eagi.AgiID,
			ExtensionID:            cfg.Eagi.ExtensionID,
			GooglePrivateKeyPath:   cfg.Google.PrivateKeyPath,
			AsteriskAudioDirectory: cfg.Asterisk.AudioDirectory,
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
