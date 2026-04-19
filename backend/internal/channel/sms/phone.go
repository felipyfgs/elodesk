package sms

import (
	"fmt"

	"github.com/nyaruka/phonenumbers"
)

var defaultRegion = "BR"

func SetDefaultRegion(region string) {
	defaultRegion = region
}

func DefaultRegion() string {
	return defaultRegion
}

func NormalizeE164(raw string, regions ...string) (string, bool) {
	region := defaultRegion
	if len(regions) > 0 && regions[0] != "" {
		region = regions[0]
	}

	num, err := phonenumbers.Parse(raw, region)
	if err != nil {
		return raw, false
	}

	if !phonenumbers.IsValidNumber(num) {
		return raw, false
	}

	return phonenumbers.Format(num, phonenumbers.E164), true
}

func ParseRegion(raw string, region string) (*phonenumbers.PhoneNumber, error) {
	return phonenumbers.Parse(raw, region)
}

func FormatE164(num *phonenumbers.PhoneNumber) string {
	return phonenumbers.Format(num, phonenumbers.E164)
}

func IsValidE164(e164 string) bool {
	num, err := phonenumbers.Parse(e164, "")
	if err != nil {
		return false
	}
	return phonenumbers.IsValidNumber(num)
}

func ParseAndNormalize(raw string) (e164 string, valid bool) {
	for _, region := range []string{defaultRegion, "US", ""} {
		num, err := phonenumbers.Parse(raw, region)
		if err != nil {
			continue
		}
		if phonenumbers.IsValidNumber(num) {
			return phonenumbers.Format(num, phonenumbers.E164), true
		}
	}
	return raw, false
}

func CountryCode(e164 string) (int, error) {
	num, err := phonenumbers.Parse(e164, "")
	if err != nil {
		return 0, fmt.Errorf("parse phone: %w", err)
	}
	return int(num.GetCountryCode()), nil
}
