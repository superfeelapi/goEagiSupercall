// Package goEagi of google.go provides a simplified interface
// for calling Google's speech to text service.

package goEagi

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"

	speech "cloud.google.com/go/speech/apiv1"
	speechpb "google.golang.org/genproto/googleapis/cloud/speech/v1"
)

const (
	sampleRate  = 8000
	domainModel = "phone_call"
)

type GoogleResult struct {
	Result *speechpb.StreamingRecognitionResult
	Info   string
	Error  error
}

// GoogleService provides information to Google Speech Recognizer
// and speech to text methods.
type GoogleService struct {
	languageCode   string
	privateKeyPath string
	enhancedMode   bool
	speechContext  []string
	client         speechpb.Speech_StreamingRecognizeClient

	sync.Mutex
}

// NewGoogleService is a constructor of GoogleService,
// it takes a privateKeyPath to set it in environment with key GOOGLE_APPLICATION_CREDENTIALS,
// a languageCode, example ["en-GB", "en-US", "ch", ...], see (https://cloud.google.com/speech-to-text/docs/languages),
// and a speech context, see (https://cloud.google.com/speech-to-text/docs/speech-adaptation).
func NewGoogleService(privateKeyPath string, languageCode string, speechContext []string) (*GoogleService, error) {
	if len(strings.TrimSpace(privateKeyPath)) == 0 {
		return nil, errors.New("private key path is empty")
	}

	err := os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", privateKeyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to set Google credential's env: %v\n", err)
	}

	g := GoogleService{
		languageCode:   languageCode,
		privateKeyPath: privateKeyPath,
		enhancedMode:   false,
		speechContext:  speechContext,
	}

	for _, v := range supportedEnhancedMode() {
		if v == languageCode {
			g.enhancedMode = true
			break
		}
	}

	ctx := context.Background()

	client, err := speech.NewClient(ctx)
	if err != nil {
		return nil, err
	}

	g.client, err = client.StreamingRecognize(ctx)
	if err != nil {
		return nil, err
	}

	sc := &speechpb.SpeechContext{Phrases: speechContext}

	if err := g.client.Send(&speechpb.StreamingRecognizeRequest{
		StreamingRequest: &speechpb.StreamingRecognizeRequest_StreamingConfig{
			StreamingConfig: &speechpb.StreamingRecognitionConfig{
				Config: &speechpb.RecognitionConfig{
					Encoding:        speechpb.RecognitionConfig_LINEAR16,
					SampleRateHertz: sampleRate,
					LanguageCode:    g.languageCode,
					Model:           domainModel,
					UseEnhanced:     g.enhancedMode,
					SpeechContexts:  []*speechpb.SpeechContext{sc},
				},
				InterimResults: true,
			},
		},
	}); err != nil {
		return nil, err
	}

	return &g, nil
}

// StartStreaming takes a reading channel of audio stream and sends it
// as a gRPC request to Google service through the initialized client.
// Caller should run it in a goroutine.
func (g *GoogleService) StartStreaming(ctx context.Context, stream <-chan []byte) <-chan error {
	startStream := make(chan error)

	go func() {
		defer close(startStream)

		for {
			select {
			case <-ctx.Done():
				return

			case s := <-stream:
				g.Lock()
				if err := g.client.Send(&speechpb.StreamingRecognizeRequest{
					StreamingRequest: &speechpb.StreamingRecognizeRequest_AudioContent{
						AudioContent: s,
					},
				}); err != nil {
					startStream <- fmt.Errorf("streaming error: %v\n", err)
					return
				}
				g.Unlock()
			}
		}
	}()

	return startStream
}

// SpeechToTextResponse sends the transcription response from Google's SpeechToText.
func (g *GoogleService) SpeechToTextResponse(ctx context.Context) <-chan GoogleResult {
	googleResultStream := make(chan GoogleResult, 5)
	//t := time.Now()

	go func() {
		defer close(googleResultStream)

		for {
			select {
			case <-ctx.Done():
				return

			default:
				resp, err := g.client.Recv()

				if err == io.EOF {
					return
				}

				if err != nil {
					googleResultStream <- GoogleResult{Error: fmt.Errorf("failed to stream results: %v", err)}
					return
				}

				//if time.Since(t).Seconds() > 10 {
				//	g.Lock()
				//
				//	googleResultStream <- GoogleResult{Info: fmt.Sprintf("before client address: %p", g.client)}
				//
				//	if err := g.client.CloseSend(); err != nil {
				//		googleResultStream <- GoogleResult{Error: fmt.Errorf("failed to close streaming client: %v", err)}
				//		g.Unlock()
				//		return
				//	}
				//
				//	googleResultStream <- GoogleResult{Info: "Reinitializing Google's client"}
				//
				//	ctx := context.Background()
				//
				//	client, err := speech.NewClient(ctx)
				//	if err != nil {
				//		googleResultStream <- GoogleResult{Error: fmt.Errorf("failed to reinitialize streaming client: %v", err)}
				//		g.Unlock()
				//		return
				//	}
				//
				//	g.client, err = client.StreamingRecognize(ctx)
				//	if err != nil {
				//		googleResultStream <- GoogleResult{Error: fmt.Errorf("failed to reinitialize streaming client: %v", err)}
				//		g.Unlock()
				//		return
				//	}
				//
				//	sc := &speechpb.SpeechContext{Phrases: g.speechContext}
				//
				//	if err := g.client.Send(&speechpb.StreamingRecognizeRequest{
				//		StreamingRequest: &speechpb.StreamingRecognizeRequest_StreamingConfig{
				//			StreamingConfig: &speechpb.StreamingRecognitionConfig{
				//				Config: &speechpb.RecognitionConfig{
				//					Encoding:        speechpb.RecognitionConfig_LINEAR16,
				//					SampleRateHertz: sampleRate,
				//					LanguageCode:    g.languageCode,
				//					Model:           domainModel,
				//					UseEnhanced:     g.enhancedMode,
				//					SpeechContexts:  []*speechpb.SpeechContext{sc},
				//				},
				//				InterimResults: true,
				//			},
				//		},
				//	}); err != nil {
				//		googleResultStream <- GoogleResult{Error: fmt.Errorf("failed to reinitialize streaming client: %v", err)}
				//		g.Unlock()
				//		return
				//	}
				//
				//	googleResultStream <- GoogleResult{Info: "Reinitialized!"}
				//
				//	googleResultStream <- GoogleResult{Info: fmt.Sprintf("after client address: %p", g.client)}
				//	g.Unlock()
				//
				//	t = time.Now()
				//}

				if err := resp.Error; err != nil {
					if err.Code == 3 || err.Code == 11 {
						g.Lock()
						googleResultStream <- GoogleResult{Info: resp.Error.Message}
						googleResultStream <- GoogleResult{Info: "Reinitializing Google's client"}

						if err := g.ReinitializeClient(); err != nil {
							googleResultStream <- GoogleResult{Error: fmt.Errorf("failed to reinitialize streaming client: %v", err)}
							g.Unlock()
							return
						}

						googleResultStream <- GoogleResult{Info: "Reinitialized!"}

						g.Unlock()
						//t = time.Now()
					}
				}

				for _, result := range resp.Results {
					googleResultStream <- GoogleResult{Result: result}
				}
			}
		}
	}()

	return googleResultStream
}

func (g *GoogleService) ReinitializeClient() error {
	ctx := context.Background()

	client, err := speech.NewClient(ctx)
	if err != nil {
		return err
	}

	g.client, err = client.StreamingRecognize(ctx)
	if err != nil {
		return err
	}

	sc := &speechpb.SpeechContext{Phrases: g.speechContext}

	if err := g.client.Send(&speechpb.StreamingRecognizeRequest{
		StreamingRequest: &speechpb.StreamingRecognizeRequest_StreamingConfig{
			StreamingConfig: &speechpb.StreamingRecognitionConfig{
				Config: &speechpb.RecognitionConfig{
					Encoding:        speechpb.RecognitionConfig_LINEAR16,
					SampleRateHertz: sampleRate,
					LanguageCode:    g.languageCode,
					Model:           domainModel,
					UseEnhanced:     g.enhancedMode,
					SpeechContexts:  []*speechpb.SpeechContext{sc},
				},
				InterimResults: true,
			},
		},
	}); err != nil {
		return err
	}

	return nil
}

func supportedEnhancedMode() []string {
	return []string{"es-US", "en-GB", "en-US", "fr-FR", "ja-JP", "pt-BR", "ru-RU"}
}
