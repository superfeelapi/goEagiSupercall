package voicebot

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"time"
)

const (
	apiTimeout = 10
)

func VoiceEmotion(apiEndpoint string, apiKey string, audioPath string) (Result, error) {
	ctx, cancel := context.WithTimeout(context.Background(), apiTimeout*time.Second)
	defer cancel()

	payload := bytes.Buffer{}
	writer := multipart.NewWriter(&payload)

	file, err := os.Open(audioPath)
	if err != nil {
		return Result{}, err
	}

	part, err := writer.CreateFormFile("voice", audioPath)
	if err != nil {
		return Result{}, err
	}

	_, err = io.Copy(part, file)
	if err != nil {
		return Result{}, err
	}
	writer.Close()

	req, err := http.NewRequest(http.MethodPost, apiEndpoint, &payload)
	if err != nil {
		return Result{}, err
	}

	req = req.WithContext(ctx)
	req.Header.Add("api-key", apiKey)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	client := http.Client{}

	resp, err := client.Do(req)
	if err != nil {
		return Result{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusInternalServerError {
		return Result{}, errors.New("internal server error 500")
	}

	bytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return Result{}, err
	}

	if resp.StatusCode != http.StatusOK {
		return Result{}, errors.New(string(bytes))
	}

	var r Result
	if err := json.Unmarshal(bytes, &r); err != nil {
		return Result{}, err
	}

	return r, nil
}
