package supercall

type AgiData struct {
	Source      string `json:"source"`
	AgiId       string `json:"agi_id"`
	ExtensionId string `json:"extension_id"`
}

type TranscriptionData struct {
	Source        string `json:"source"`
	AgiId         string `json:"agi_id"`
	ExtensionId   string `json:"extension_id"`
	DataId        string `json:"data_id"`
	Transcription string `json:"transcription"`
	Interim       bool   `json:"interim"`
}

type TextEmotionData struct {
	Source                string  `json:"source"`
	AgiId                 string  `json:"agi_id"`
	ExtensionId           string  `json:"extension_id"`
	DataId                string  `json:"data_id"`
	TextEmotion           string  `json:"text_emotion"`
	TextEmotionConfidence float64 `json:"text_emotion_confidence"`
	TextContext           string  `json:"text_context"`
	TextContextConfidence float64 `json:"text_context_confidence"`
}

type VoiceEmotionData struct {
	Source                 string  `json:"source"`
	AgiId                  string  `json:"agi_id"`
	ExtensionId            string  `json:"extension_id"`
	DataId                 string  `json:"data_id"`
	VoiceEmotion           string  `json:"voice_emotion,omitempty"`
	VoiceEmotionConfidence float64 `json:"voice_emotion_confidence,omitempty"`
	VoiceAmplitude         string  `json:"voice_amplitude,omitempty"`
	VoicePace              string  `json:"voice_pace,omitempty"`
}
