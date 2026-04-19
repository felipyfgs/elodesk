package sms

import (
	"encoding/json"
	"fmt"
)

type ProviderConfig struct {
	Twilio    *TwilioConfig    `json:"twilio,omitempty"`
	Bandwidth *BandwidthConfig `json:"bandwidth,omitempty"`
	Zenvia    *ZenviaConfig    `json:"zenvia,omitempty"`
}

type TwilioConfig struct {
	AccountSID          string `json:"accountSid"`
	AuthToken           string `json:"authToken"`
	MessagingServiceSID string `json:"messagingServiceSid,omitempty"`
}

type BandwidthConfig struct {
	AccountID     string `json:"accountId"`
	ApplicationID string `json:"applicationId"`
	BasicAuthUser string `json:"basicAuthUser"`
	BasicAuthPass string `json:"basicAuthPass"`
}

type ZenviaConfig struct {
	APIToken string `json:"apiToken"`
}

func ParseProviderConfig(provider string, rawConfig string) (*ProviderConfig, error) {
	if rawConfig == "" {
		return nil, fmt.Errorf("sms: provider config is empty")
	}

	pc := &ProviderConfig{}
	if err := json.Unmarshal([]byte(rawConfig), pc); err != nil {
		return nil, fmt.Errorf("sms: unmarshal provider config: %w", err)
	}

	switch provider {
	case "twilio":
		if pc.Twilio == nil || pc.Twilio.AccountSID == "" || pc.Twilio.AuthToken == "" {
			return nil, fmt.Errorf("sms: twilio config requires accountSid and authToken")
		}
	case "bandwidth":
		if pc.Bandwidth == nil || pc.Bandwidth.AccountID == "" || pc.Bandwidth.ApplicationID == "" || pc.Bandwidth.BasicAuthUser == "" || pc.Bandwidth.BasicAuthPass == "" {
			return nil, fmt.Errorf("sms: bandwidth config requires accountId, applicationId, basicAuthUser, and basicAuthPass")
		}
	case "zenvia":
		if pc.Zenvia == nil || pc.Zenvia.APIToken == "" {
			return nil, fmt.Errorf("sms: zenvia config requires apiToken")
		}
	default:
		return nil, fmt.Errorf("sms: unknown provider %q", provider)
	}

	return pc, nil
}

func (pc *ProviderConfig) Serialize() (string, error) {
	b, err := json.Marshal(pc)
	if err != nil {
		return "", fmt.Errorf("sms: serialize provider config: %w", err)
	}
	return string(b), nil
}

func ConfigForProvider(provider string, pc *ProviderConfig) (interface{}, error) {
	switch provider {
	case "twilio":
		if pc.Twilio == nil {
			return nil, fmt.Errorf("sms: no twilio config")
		}
		return pc.Twilio, nil
	case "bandwidth":
		if pc.Bandwidth == nil {
			return nil, fmt.Errorf("sms: no bandwidth config")
		}
		return pc.Bandwidth, nil
	case "zenvia":
		if pc.Zenvia == nil {
			return nil, fmt.Errorf("sms: no zenvia config")
		}
		return pc.Zenvia, nil
	default:
		return nil, fmt.Errorf("sms: unknown provider %q", provider)
	}
}
