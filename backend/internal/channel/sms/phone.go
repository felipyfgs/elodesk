package sms

import (
	"github.com/nyaruka/phonenumbers"

	"backend/internal/phone"
)

// Thin re-exports of backend/internal/phone so existing callers under
// channel/sms keep working after the helpers were promoted to a shared
// package (the move was driven by the need to call NormalizeE164 from
// service/, which couldn't import sms without an import cycle).

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
