package google

import (
	"context"
	"fmt"
	"os"
	"time"

	"cloud.google.com/go/translate"
	"golang.org/x/text/language"
)

const translationTimeout = 3 * time.Second

type Translation struct {
	ApiKey    string
	TargetTag language.Tag
	Client    *translate.Client
}

func NewTranslation(apiKey, targetLanguageCode string) (*Translation, error) {
	if env := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS"); env == "" {
		os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", apiKey)
	}

	targetTag, err := language.Parse(targetLanguageCode)
	if err != nil {
		return nil, fmt.Errorf("incorrect target langauge code: %w", err)
	}

	client, err := translate.NewClient(context.Background())
	if err != nil {
		return nil, fmt.Errorf("unable to create google translate client: %w", err)
	}

	t := Translation{
		ApiKey:    apiKey,
		TargetTag: targetTag,
		Client:    client,
	}
	return &t, nil
}

func (t *Translation) Translate(text string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), translationTimeout)
	defer cancel()

	option := translate.Options{
		Format: "text",
	}

	resp, err := t.Client.Translate(ctx, []string{text}, t.TargetTag, &option)
	if err != nil {
		return "", fmt.Errorf("unable to translate text: %w", err)
	}
	return resp[0].Text, nil
}
