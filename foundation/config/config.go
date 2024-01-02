package config

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
)

func GetCampaign(eagiConfigPath string, campaignID string) (Campaign, error) {
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

	campaign, exists := campaignExists(config.Projects, campaignID)
	if !exists {
		return Campaign{}, fmt.Errorf("campaign[%s] does not exist", campaignID)
	}

	return campaign, nil
}

func (c Campaign) IsGoogleInUse(boundType string) (bool, error) {
	switch boundType {
	case "inbound":
		return c.Inbound.Google.InUse, nil

	case "outbound":
		return c.Inbound.Google.InUse, nil

	default:
		return false, fmt.Errorf("bound type[%s] does not exist", boundType)
	}
}

func (c Campaign) IsAzureInUse(boundType string) (bool, error) {
	switch boundType {
	case "inbound":
		return c.Inbound.Azure.InUse, nil

	case "outbound":
		return c.Inbound.Azure.InUse, nil

	default:
		return false, fmt.Errorf("bound type[%s] does not exist", boundType)
	}
}

func (c Campaign) GetGoogleLanguageCode(boundType string) (string, error) {
	switch boundType {
	case "inbound":
		return c.Inbound.Google.LanguageCode, nil

	case "outbound":
		return c.Inbound.Google.LanguageCode, nil

	default:
		return "", fmt.Errorf("bound type[%s] does not exist", boundType)
	}
}

func (c Campaign) GetAzureLanguageCode(boundType string) ([]string, error) {
	switch boundType {
	case "inbound":
		return c.Inbound.Azure.LanguageCode, nil

	case "outbound":
		return c.Inbound.Azure.LanguageCode, nil

	default:
		return nil, fmt.Errorf("bound type[%s] does not exist", boundType)
	}
}

func (c Campaign) GetGoogleSpeechContext(boundType string) ([]string, error) {
	speechContext := make([]string, 0)
	switch boundType {
	case "inbound":
		for _, v := range c.Inbound.Google.SpeechContext {
			speechContext = append(speechContext, v)
		}
		return speechContext, nil

	case "outbound":
		for _, v := range c.Outbound.Google.SpeechContext {
			speechContext = append(speechContext, v)
		}
		return speechContext, nil

	default:
		return nil, fmt.Errorf("bound type[%s] does not exist", boundType)
	}
}

func campaignExists(projects []Project, campaignID string) (Campaign, bool) {
	for _, project := range projects {
		for _, campaign := range project.Campaigns {
			if campaign.ID == campaignID {
				return campaign, true
			}
		}
	}
	return Campaign{}, false
}
