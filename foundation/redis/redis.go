package redis

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

type Redis struct {
	Client               *redis.Client
	Logger               *zap.SugaredLogger
	TranscriptionChannel string
	ScamBotChannel       string
}

func New(host, password, transcriptionChannel, scamBotChannel string, logger *zap.SugaredLogger) (*Redis, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     host,
		Password: password,
	})

	_, err := client.Ping(context.Background()).Result()
	if err != nil {
		return nil, fmt.Errorf("redis connection failed: %w", err)
	}

	return &Redis{
		Client:               client,
		Logger:               logger,
		TranscriptionChannel: transcriptionChannel,
		ScamBotChannel:       scamBotChannel,
	}, nil
}

func (r *Redis) ConsumeScamBotChannel() <-chan *redis.Message {
	msgCh := r.Client.Subscribe(context.Background(), r.ScamBotChannel).Channel()
	return msgCh
}

func (r *Redis) Produce(data interface{}) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		fmt.Println(err)
		return err
	}

	err = r.Client.Publish(context.Background(), r.TranscriptionChannel, jsonData).Err()
	if err != nil {
		return err
	}

	r.Logger.Infow("redis: Produce", "channel", r.TranscriptionChannel, "data", data)

	return nil
}
