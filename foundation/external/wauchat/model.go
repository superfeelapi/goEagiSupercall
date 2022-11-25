package wauchat

type Emotion struct {
	Confidence float64 `json:"confidence"`
	Result     string  `json:"result"`
}

type Context struct {
	Confidence float64 `json:"confidence"`
	Result     string  `json:"result"`
}

type Result struct {
	Context Context `json:"context"`
	Emotion Emotion `json:"emotion"`
}
