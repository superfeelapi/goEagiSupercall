package textAnalysis_test

import (
	"testing"

	"github.com/superfeelapi/goEagiSupercall/foundation/external/textAnalysis"
)

func TestTextEmotion(t *testing.T) {
	r, err := textAnalysis.TextEmotion("https://chatgptreq.superceed.com/text_emotion", "hello there how are you?")
	if err != nil {
		t.Fatal(err)
	}
	t.Log(r)
}
