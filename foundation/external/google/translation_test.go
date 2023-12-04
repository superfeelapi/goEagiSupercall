package google_test

import (
	"testing"

	"github.com/superfeelapi/goEagiSupercall/foundation/external/google"
)

const googleCred = "../../../boxwood-pilot-299014-769b582bc376.json"

func TestTranslation_Translate(t *testing.T) {
	text := "God bless you"
	translation, err := google.NewTranslation(googleCred, "en", "zh-HK")
	if err != nil {
		t.Fatal(err)
	}
	resp, err := translation.Translate(text)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(resp)
}
