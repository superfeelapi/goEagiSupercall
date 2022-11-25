package wauchat

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	apiTimeout = 3
)

func TextEmotion(apiEndpoint string, text string) (Result, error) {
	ctx, cancel := context.WithTimeout(context.Background(), apiTimeout*time.Second)
	defer cancel()

	preprocessEndpoint := apiEndpoint + preprocessText(text)

	req, err := http.NewRequest(http.MethodGet, preprocessEndpoint, nil)
	if err != nil {
		return Result{}, err
	}

	req = req.WithContext(ctx)
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

func preprocessText(text string) string {
	return strings.Replace(text, " ", "%20", -1)
}
