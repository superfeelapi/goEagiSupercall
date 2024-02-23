package textAnalysis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

const (
	apiTimeout = 15
	apiKey     = "a4b98363-e598-4550-97fb-a8ec138fcf38"
)

func TextEmotion(apiEndpoint string, text string) (Result, error) {
	ctx, cancel := context.WithTimeout(context.Background(), apiTimeout*time.Second)
	defer cancel()

	u, err := url.Parse(apiEndpoint)
	if err != nil {
		return Result{}, fmt.Errorf("failed to parse endpoint: %w", err)
	}

	q := u.Query()
	q.Add("text", text)
	u.RawQuery = q.Encode()

	req, err := http.NewRequest(http.MethodPost, u.String(), nil)
	if err != nil {
		return Result{}, err
	}
	req.Header.Add("api-key", apiKey)

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
