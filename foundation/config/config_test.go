package config_test

import (
	"log"
	"testing"

	"github.com/superfeelapi/goEagiSupercall/foundation/config"
)

const (
	filepath   = "config.json"
	campaignID = "1"
)

func TestGetCampaign(t *testing.T) {
	t.Run("campaign exists", func(t *testing.T) {
		t.Parallel()
		campaign, err := config.GetCampaign(filepath, campaignID)
		if err != nil {
			t.Fatal(err)
		}
		log.Printf("%+v\n", campaign)
	})

	t.Run("campaign does not exist", func(t *testing.T) {
		t.Parallel()
		_, err := config.GetCampaign(filepath, "0")
		if err == nil {
			t.Fatal("unexpected error")
		}
	})
}
