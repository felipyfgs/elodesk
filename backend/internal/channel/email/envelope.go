package email

import "github.com/jhillyerd/enmime"

// Attachment is a single file part extracted from a MIME message.
type Attachment struct {
	Filename    string
	ContentType string
	Inline      bool
	Data        []byte
}

// Envelope is the parsed representation of an inbound email message.
type Envelope struct {
	MessageID   string
	InReplyTo   string
	References  []string // up to 50 IDs, oldest first
	From        string   // "Name <addr>" or just "addr"
	To          []string
	Cc          []string
	Subject     string
	Text        string // plain-text body
	HTML        string // HTML body (raw, not sanitized)
	Attachments []Attachment
}

// fromEnmime converts an enmime.Envelope into our internal Envelope.
// References are limited to the last 50 IDs to match Chatwoot behaviour.
func fromEnmime(e *enmime.Envelope) *Envelope {
	refs := splitMessageIDs(e.GetHeader("References"))
	if len(refs) > 50 {
		refs = refs[len(refs)-50:]
	}

	var attachments []Attachment
	for _, p := range e.Attachments {
		attachments = append(attachments, Attachment{
			Filename:    p.FileName,
			ContentType: p.ContentType,
			Inline:      false,
			Data:        p.Content,
		})
	}
	for _, p := range e.Inlines {
		attachments = append(attachments, Attachment{
			Filename:    p.FileName,
			ContentType: p.ContentType,
			Inline:      true,
			Data:        p.Content,
		})
	}

	env := &Envelope{
		MessageID:   e.GetHeader("Message-ID"),
		InReplyTo:   e.GetHeader("In-Reply-To"),
		References:  refs,
		From:        e.GetHeader("From"),
		Subject:     e.GetHeader("Subject"),
		Text:        e.Text,
		HTML:        e.HTML,
		Attachments: attachments,
	}

	if toAddrs, _ := e.AddressList("To"); toAddrs != nil {
		for _, addr := range toAddrs {
			env.To = append(env.To, addr.Address)
		}
	}
	if ccAddrs, _ := e.AddressList("Cc"); ccAddrs != nil {
		for _, addr := range ccAddrs {
			env.Cc = append(env.Cc, addr.Address)
		}
	}
	return env
}
