package sms

import "testing"

func TestNormalizeE164_Brazil(t *testing.T) {
	tests := []struct {
		input string
		want  string
		valid bool
	}{
		{"+5511988887777", "+5511988887777", true},
		{"11988887777", "+5511988887777", true},
		{"5511988887777", "+5511988887777", true},
	}

	SetDefaultRegion("BR")
	defer SetDefaultRegion("BR")

	for _, tt := range tests {
		got, ok := NormalizeE164(tt.input)
		if ok != tt.valid {
			t.Errorf("NormalizeE164(%q) valid = %v, want %v", tt.input, ok, tt.valid)
		}
		if got != tt.want {
			t.Errorf("NormalizeE164(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestNormalizeE164_US(t *testing.T) {
	tests := []struct {
		input  string
		region string
		want   string
		valid  bool
	}{
		{"+14155551234", "", "+14155551234", true},
		{"4155551234", "US", "+14155551234", true},
	}

	for _, tt := range tests {
		got, ok := NormalizeE164(tt.input, tt.region)
		if ok != tt.valid {
			t.Errorf("NormalizeE164(%q, %q) valid = %v, want %v", tt.input, tt.region, ok, tt.valid)
		}
		if ok && got != tt.want {
			t.Errorf("NormalizeE164(%q, %q) = %q, want %q", tt.input, tt.region, got, tt.want)
		}
	}
}

func TestNormalizeE164_Invalid(t *testing.T) {
	SetDefaultRegion("BR")
	defer SetDefaultRegion("BR")

	tests := []struct {
		input string
	}{
		{"abc"},
		{"123"},
		{""},
	}

	for _, tt := range tests {
		_, ok := NormalizeE164(tt.input)
		if ok {
			t.Errorf("NormalizeE164(%q) should be invalid", tt.input)
		}
	}
}

func TestIsValidE164(t *testing.T) {
	if !IsValidE164("+5511988887777") {
		t.Error("IsValidE164(+5511988887777) = false, want true")
	}
	if !IsValidE164("+14155551234") {
		t.Error("IsValidE164(+14155551234) = false, want true")
	}
	if IsValidE164("abc") {
		t.Error("IsValidE164(abc) = true, want false")
	}
}
