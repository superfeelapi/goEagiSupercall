package config

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
)

func GetCampaign(eagiConfigPath string, campaignID, boundType string) (Campaign, error) {
	file, err := os.Open(eagiConfigPath)
	if err != nil {
		return Campaign{}, err
	}
	defer file.Close()

	bytes, err := io.ReadAll(file)
	if err != nil {
		return Campaign{}, err
	}

	var config Config

	if err := json.Unmarshal(bytes, &config); err != nil {
		return Campaign{}, err
	}

	campaign, exists := campaignExists(config.Eagi, campaignID)
	if !exists {
		return Campaign{}, fmt.Errorf("campaign[%s] does not exist", campaignID)
	}

	return campaign, nil
}

func GetLanguage(c Campaign, boundType string) string {
	switch boundType {
	case "inbound":
		return c.Inbound.Language

	case "outbound":
		return c.Outbound.Language
	}
	return ""
}

func GetLanguageCode(c Campaign, boundType string) string {
	switch boundType {
	case "inbound":
		return c.Inbound.LanguageCode

	case "outbound":
		return c.Outbound.LanguageCode
	}
	return ""
}

func GetSpeechContext(c Campaign, boundType string) []string {
	var scMap map[string]string
	switch boundType {
	case "inbound":
		scMap = c.Inbound.SpeechContext

	case "outbound":
		scMap = c.Outbound.SpeechContext
	}

	scSlice := make([]string, 0, len(scMap))

	for _, v := range scMap {
		scSlice = append(scSlice, v)
	}

	return scSlice
}

func campaignExists(p []Project, campaignID string) (Campaign, bool) {
	for _, v := range p {
		for _, campaign := range v.Campaigns {
			if campaign.ID == campaignID {
				return campaign, true
			}
		}
	}
	return Campaign{}, false
}
