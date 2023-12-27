package config

type Config struct {
	Projects []Project `json:"projects"`
}

type Project struct {
	ExtensionID string     `json:"extension_id"`
	Campaigns   []Campaign `json:"campaigns"`
}

type Campaign struct {
	ID          string      `json:"id"`
	Name        string      `json:"name"`
	Kind        string      `json:"kind"`
	InUse       bool        `json:"in_use"`
	Translation Translation `json:"translation"`
	Scam        Scam        `json:"scam"`
	Inbound     Bound       `json:"inbound"`
	Outbound    Bound       `json:"outbound"`
}

type Translation struct {
	InUse  bool   `json:"in_use"`
	Source string `json:"source"`
	Target string `json:"target"`
}

type Scam struct {
	InUse     bool   `json:"in_use"`
	AudioPath string `json:"audio_path"`
}

type Bound struct {
	Google Google `json:"google"`
	Azure  Azure  `json:"azure"`
}

type Google struct {
	InUse         bool              `json:"in_use"`
	LanguageCode  string            `json:"language_code"`
	SpeechContext map[string]string `json:"speech_context"`
}

type Azure struct {
	InUse        bool     `json:"in_use"`
	LanguageCode []string `json:"language_code"`
}
