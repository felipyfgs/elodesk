package sms

import (
	"github.com/nyaruka/phonenumbers"

	"backend/internal/phone"
)

func SetDefaultRegion(region string) { phone.SetDefaultRegion(region) }

func DefaultRegion() string { return phone.DefaultRegion() }

func NormalizeE164(raw string, regions ...string) (string, bool) {
	return phone.NormalizeE164(raw, regions...)
}

func ParseRegion(raw string, region string) (*phonenumbers.PhoneNumber, error) {
	return phone.ParseRegion(raw, region)
}

func FormatE164(num *phonenumbers.PhoneNumber) string { return phone.FormatE164(num) }

func IsValidE164(e164 string) bool { return phone.IsValidE164(e164) }

func ParseAndNormalize(raw string) (string, bool) { return phone.ParseAndNormalize(raw) }

func CountryCode(e164 string) (int, error) { return phone.CountryCode(e164) }
