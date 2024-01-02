package textAnalysis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	apiTimeout = 15
)

func TextEmotion(apiEndpoint string, text string) (Result, error) {
	ctx, cancel := context.WithTimeout(context.Background(), apiTimeout*time.Second)
	defer cancel()

	params := url.Values{}
	params.Add("text", text)

	payload := strings.NewReader(params.Encode())

	req, err := http.NewRequest(http.MethodPost, apiEndpoint, payload)
	if err != nil {
		return Result{}, err
	}
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	req = req.WithContext(ctx)
	client := http.Client{}

	resp, err := client.Do(req)
	if err != nil {
		return Result{}, err
	}
	defer resp.Body.Close()

	bytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return Result{}, err
	}

	if resp.StatusCode == http.StatusInternalServerError {
		return Result{}, errors.New(fmt.Sprintf("internal server error 500: %s", string(bytes)))
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
