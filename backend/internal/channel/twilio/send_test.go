package twilio

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"backend/internal/model"
)

// sendTestServer captures form params from /Accounts/{sid}/Messages.json POSTs.
type sendTestServer struct {
	srv    *httptest.Server
	form   map[string]string
	status int
}

func newSendTestServer(t *testing.T, status int, responseSID string) *sendTestServer {
	t.Helper()
	s := &sendTestServer{status: status, form: map[string]string{}}
	s.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		for k, v := range r.PostForm {
			if len(v) > 0 {
				s.form[k] = v[0]
			}
		}
		w.WriteHeader(s.status)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"sid":    responseSID,
			"status": "queued",
		})
	}))
	t.Cleanup(s.srv.Close)
	return s
}

func TestSend_WhatsappPrefixesBothEnds(t *testing.T) {
	ts := newSendTestServer(t, http.StatusOK, "SMabc")
	origBase := APIBaseOverride
	APIBaseOverride = ts.srv.URL
	defer func() { APIBaseOverride = origBase }()

	phone := "+14155552671"
	ch := &model.ChannelTwilio{
		ID:         1,
		AccountID:  10,
		AccountSID: "ACxxx",
		Medium:     model.TwilioMediumWhatsApp,
		PhoneNumber: &phone,
	}
	client := NewClient(nil)
	sid, err := Send(context.Background(), client, OutboundInput{
		Channel:   ch,
		AuthToken: "tok",
		To:        "+5511988887777",
		Body:      "oi",
	})
	if err != nil {
		t.Fatalf("Send failed: %v", err)
	}
	if sid != "SMabc" {
		t.Fatalf("expected SID, got %q", sid)
	}
	if ts.form["From"] != "whatsapp:"+phone {
		t.Fatalf("From should carry whatsapp: prefix, got %q", ts.form["From"])
	}
	if ts.form["To"] != "whatsapp:+5511988887777" {
		t.Fatalf("To should carry whatsapp: prefix, got %q", ts.form["To"])
	}
}

func TestSend_MessagingServiceSIDWins(t *testing.T) {
	ts := newSendTestServer(t, http.StatusOK, "SMxyz")
	origBase := APIBaseOverride
	APIBaseOverride = ts.srv.URL
	defer func() { APIBaseOverride = origBase }()

	mss := "MG123"
	ch := &model.ChannelTwilio{
		ID:                  1,
		AccountID:           10,
		AccountSID:          "ACxxx",
		Medium:              model.TwilioMediumSMS,
		MessagingServiceSID: &mss,
	}
	client := NewClient(nil)
	_, err := Send(context.Background(), client, OutboundInput{
		Channel:   ch,
		AuthToken: "tok",
		To:        "+14155551234",
		Body:      "hi",
	})
	if err != nil {
		t.Fatalf("Send failed: %v", err)
	}
	if ts.form["MessagingServiceSid"] != mss {
		t.Fatalf("MessagingServiceSid should be forwarded, got %q", ts.form["MessagingServiceSid"])
	}
	if _, ok := ts.form["From"]; ok {
		t.Fatalf("From must NOT be set when messaging_service_sid is present")
	}
}

func TestSend_ContentVariablesMarshalled(t *testing.T) {
	ts := newSendTestServer(t, http.StatusOK, "SM1")
	origBase := APIBaseOverride
	APIBaseOverride = ts.srv.URL
	defer func() { APIBaseOverride = origBase }()

	phone := "+14155550001"
	ch := &model.ChannelTwilio{
		ID: 1, AccountID: 1, AccountSID: "ACxxx",
		Medium: model.TwilioMediumSMS, PhoneNumber: &phone,
	}
	_, err := Send(context.Background(), NewClient(nil), OutboundInput{
		Channel:          ch,
		AuthToken:        "tok",
		To:               "+14155550002",
		ContentSID:       "HX1",
		ContentVariables: map[string]string{"1": "Jo", "2": "ão"},
	})
	if err != nil {
		t.Fatalf("Send failed: %v", err)
	}
	if ts.form["ContentSid"] != "HX1" {
		t.Fatalf("ContentSid should be forwarded")
	}
	if !strings.Contains(ts.form["ContentVariables"], `"1":"Jo"`) {
		t.Fatalf("ContentVariables must contain JSON-marshalled map, got %q", ts.form["ContentVariables"])
	}
}

func TestSend_AuthErrorPropagates(t *testing.T) {
	ts := newSendTestServer(t, http.StatusUnauthorized, "")
	origBase := APIBaseOverride
	APIBaseOverride = ts.srv.URL
	defer func() { APIBaseOverride = origBase }()

	phone := "+1"
	ch := &model.ChannelTwilio{ID: 1, AccountID: 1, AccountSID: "AC", Medium: model.TwilioMediumSMS, PhoneNumber: &phone}
	_, err := Send(context.Background(), NewClient(nil), OutboundInput{
		Channel: ch, AuthToken: "bad", To: "+2", Body: "x",
	})
	if !IsAuthError(err) {
		t.Fatalf("expected auth error, got %v", err)
	}
}
