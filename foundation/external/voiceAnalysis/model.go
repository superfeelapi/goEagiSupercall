package voiceAnalysis

type Amplitude struct {
	Amplitude float64 `json:"amplitude"`
	State     string  `json:"state"`
}

type Pace struct {
	Pace  float64 `json:"pace"`
	State string  `json:"state"`
}

type Emotion struct {
	Confidence float64 `json:"confidence"`
	Result     string  `json:"result"`
}

type EmotionPercentage struct {
	Neutral   float64
	Happy     float64
	Calm      float64
	Sad       float64
	Angry     float64
	Fearful   float64
	Disgust   float64
	Surprised float64
}

type ErrorDetail struct {
	Message string `json:"message"`
}

type Result struct {
	Amplitude   []Amplitude         `json:"amplitude"`
	Pace        []Pace              `json:"pace"`
	Emotion     []Emotion           `json:"emotion"`
	Percentage  []EmotionPercentage `json:"percentage"`
	Error       ErrorDetail         `json:"detail"`
	AudioLength float64             `json:"audio_length_seconds"`
}
