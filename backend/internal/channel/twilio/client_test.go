package twilio

import (
	"errors"
	"net/http"
	"testing"
)

func TestIsAuthError(t *testing.T) {
	cases := map[string]struct {
		err  error
		want bool
	}{
		"401":    {&APIError{StatusCode: http.StatusUnauthorized}, true},
		"403":    {&APIError{StatusCode: http.StatusForbidden}, true},
		"429":    {&APIError{StatusCode: http.StatusTooManyRequests}, false},
		"500":    {&APIError{StatusCode: http.StatusInternalServerError}, false},
		"nilErr": {nil, false},
		"plain":  {errors.New("boom"), false},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			if got := IsAuthError(tc.err); got != tc.want {
				t.Fatalf("IsAuthError(%v) = %v want %v", tc.err, got, tc.want)
			}
		})
	}
}

func TestIsRateLimited(t *testing.T) {
	if !IsRateLimited(&APIError{StatusCode: http.StatusTooManyRequests}) {
		t.Fatalf("429 must be rate-limited")
	}
	if IsRateLimited(&APIError{StatusCode: http.StatusUnauthorized}) {
		t.Fatalf("401 must not be rate-limited")
	}
	if IsRateLimited(errors.New("boom")) {
		t.Fatalf("plain errors must not be rate-limited")
	}
}

func TestBasicAuthUser_PrefersAPIKey(t *testing.T) {
	if basicAuthUser("AC123", "SK456") != "SK456" {
		t.Fatalf("API key SID should take precedence")
	}
	if basicAuthUser("AC123", "") != "AC123" {
		t.Fatalf("Account SID used as fallback")
	}
}

func TestAPIError_Error(t *testing.T) {
	e := &APIError{StatusCode: 418, Body: "teapot"}
	if got, want := e.Error(), "twilio api error: status=418 body=teapot"; got != want {
		t.Fatalf("Error() = %q want %q", got, want)
	}
}
