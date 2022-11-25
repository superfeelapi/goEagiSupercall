package config

type Config struct {
	Eagi []Project `json:"config"`
}

type Project struct {
	ExtensionID string     `json:"extension_id"`
	Campaigns   []Campaign `json:"campaigns"`
}

type Campaign struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Inbound  Bound  `json:"inbound"`
	Outbound Bound  `json:"outbound"`
}

type Bound struct {
	LanguageCode  string            `json:"language_code"`
	Language      string            `json:"language"`
	SpeechContext map[string]string `json:"speech_context"`
}
