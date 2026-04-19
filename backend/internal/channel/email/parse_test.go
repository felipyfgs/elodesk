package email_test

import (
	"strings"
	"testing"

	emailch "backend/internal/channel/email"
)

func TestParseMIME_PlainText(t *testing.T) {
	raw := "From: alice@example.com\r\nTo: bob@example.com\r\nSubject: Hello\r\nMessage-ID: <abc123@example.com>\r\n\r\nHello world"
	env, err := emailch.ParseMIME(strings.NewReader(raw))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if env.MessageID != "<abc123@example.com>" {
		t.Errorf("MessageID = %q, want <abc123@example.com>", env.MessageID)
	}
	if !strings.Contains(env.Text, "Hello world") {
		t.Errorf("Text = %q, want 'Hello world'", env.Text)
	}
}

func TestParseMIME_MultipartAlternative(t *testing.T) {
	boundary := "bound42"
	raw := "From: alice@example.com\r\nTo: bob@example.com\r\nSubject: =?UTF-8?Q?Ol=C3=A1?=\r\n" +
		"MIME-Version: 1.0\r\nContent-Type: multipart/alternative; boundary=\"" + boundary + "\"\r\n\r\n" +
		"--" + boundary + "\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\nPlain text part\r\n" +
		"--" + boundary + "\r\nContent-Type: text/html; charset=UTF-8\r\n\r\n<p>HTML part</p>\r\n" +
		"--" + boundary + "--\r\n"

	env, err := emailch.ParseMIME(strings.NewReader(raw))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(env.Text, "Plain text part") {
		t.Errorf("Text = %q, want plain text", env.Text)
	}
	if !strings.Contains(env.HTML, "HTML part") {
		t.Errorf("HTML = %q, want html", env.HTML)
	}
}

func TestParseMIME_ReferencesLimitedTo50(t *testing.T) {
	var refs []string
	for i := 0; i < 60; i++ {
		refs = append(refs, "<msg"+strings.Repeat("x", i+1)+"@example.com>")
	}
	raw := "From: a@b.com\r\nTo: c@d.com\r\nSubject: S\r\nReferences: " + strings.Join(refs, " ") + "\r\n\r\nbody"
	env, err := emailch.ParseMIME(strings.NewReader(raw))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(env.References) > 50 {
		t.Errorf("References = %d, want ≤50", len(env.References))
	}
}

func TestSplitMessageIDs(t *testing.T) {
	cases := []struct {
		input string
		count int
	}{
		{"", 0},
		{"<a@b.com>", 1},
		{"<a@b.com> <c@d.com>", 2},
		{"  <a@b.com>   <c@d.com>  <e@f.com>  ", 3},
	}
	for _, tc := range cases {
		env, _ := emailch.ParseMIME(strings.NewReader(
			"From: a@b.com\r\nTo: c@d.com\r\nSubject: S\r\nReferences: " + tc.input + "\r\n\r\nbody",
		))
		if env != nil && len(env.References) != tc.count {
			t.Errorf("input=%q: got %d refs, want %d", tc.input, len(env.References), tc.count)
		}
	}
}
