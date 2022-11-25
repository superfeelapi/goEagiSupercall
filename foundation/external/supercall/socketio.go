package supercall

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type Event string

const (
	apiTimeout = 5

	AgiEvent        Event = "sendAgiData"
	TranscriptEvent Event = "sendTranscriptionApi"
	EmotionEvent    Event = "sendEmotionApi"
	KeepAliveEvent  Event = "keepAlive"
)

type Polling struct {
	sid         string
	apiEndpoint string
}

func New(apiEndpoint string) *Polling {
	return &Polling{
		apiEndpoint: apiEndpoint,
	}
}

func (p *Polling) GetSessionID() string {
	return p.sid
}

func (p *Polling) SendData(e Event, d interface{}) error {
	ctx, cancel := context.WithTimeout(context.Background(), apiTimeout*time.Second)
	defer cancel()

	b, err := json.Marshal(d)
	if err != nil {
		return err
	}

	payload := strings.NewReader(formatData(b, e))

	req, err := http.NewRequest(http.MethodPost, addQueryParams(p.apiEndpoint, p.sid), payload)
	if err != nil {
		return err
	}

	req = req.WithContext(ctx)
	client := http.Client{}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusInternalServerError {
		return errors.New("internal server error 500")
	}

	bytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return errors.New(string(bytes))
	}

	return nil
}

func (p *Polling) SetupConnection() error {
	if err := p.establishHandshake(); err != nil {
		return err
	}

	if err := p.upgradeWebsocket(); err != nil {
		return err
	}

	if err := p.keepConnection(); err != nil {
		return err
	}

	return nil
}

func (p *Polling) establishHandshake() error {
	ctx, cancel := context.WithTimeout(context.Background(), apiTimeout*time.Second)
	defer cancel()

	req, err := http.NewRequest(http.MethodGet, addQueryParams(p.apiEndpoint, ""), nil)
	if err != nil {
		return err
	}

	req = req.WithContext(ctx)
	client := http.Client{}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusInternalServerError {
		return errors.New("internal server error 500")
	}

	bytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return errors.New(string(bytes))
	}

	var r map[string]interface{}

	if err := json.Unmarshal(bytes[1:], &r); err != nil {
		return err
	}

	if sid, ok := r["sid"]; ok {
		p.sid = sid.(string)
	} else {
		return err
	}

	return nil
}

func (p *Polling) upgradeWebsocket() error {
	ctx, cancel := context.WithTimeout(context.Background(), apiTimeout*time.Second)
	defer cancel()

	payload := strings.NewReader(`40`)

	req, err := http.NewRequest(http.MethodPost, addQueryParams(p.apiEndpoint, p.sid), payload)
	if err != nil {
		return err
	}

	req = req.WithContext(ctx)
	client := http.Client{}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusInternalServerError {
		return errors.New("internal server error 500")
	}

	bytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return errors.New(string(bytes))
	}

	return nil
}

func (p *Polling) keepConnection() error {
	ctx, cancel := context.WithTimeout(context.Background(), apiTimeout*time.Second)
	defer cancel()

	req, err := http.NewRequest(http.MethodGet, addQueryParams(p.apiEndpoint, p.sid), nil)
	if err != nil {
		return err
	}

	req = req.WithContext(ctx)
	client := http.Client{}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusInternalServerError {
		return errors.New("internal server error 500")
	}

	bytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return errors.New(string(bytes))
	}

	return nil
}

// =========================================================================

func getTimestamp() string {
	now := time.Now()
	return strconv.FormatInt(now.Unix(), 10)
}

func addQueryParams(endpoint string, sid string) string {
	u, _ := url.Parse(endpoint)
	q, _ := url.ParseQuery(u.RawQuery)
	q.Add("t", getTimestamp())
	if sid != "" {
		q.Add("sid", sid)
	}
	u.RawQuery = q.Encode()

	return u.String()
}

func formatData(b []byte, e Event) string {
	return `42["` + string(e) + `", ` + string(b) + `]`
}
