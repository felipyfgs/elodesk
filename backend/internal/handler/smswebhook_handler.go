package handler

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/gofiber/fiber/v2"

	"backend/internal/channel/sms"
	"backend/internal/logger"
	"backend/internal/repo"
	"backend/internal/service"
)

type SMSWebhookHandler struct {
	channelSMSRepo *repo.ChannelSMSRepo
	messageRepo    *repo.MessageRepo
	registry       *sms.Registry
	ingestSvc      *sms.IngestService
	messageSvc     *service.MessageService
}

func NewSMSWebhookHandler(
	channelSMSRepo *repo.ChannelSMSRepo,
	messageRepo *repo.MessageRepo,
	registry *sms.Registry,
	ingestSvc *sms.IngestService,
	messageSvc *service.MessageService,
) *SMSWebhookHandler {
	return &SMSWebhookHandler{
		channelSMSRepo: channelSMSRepo,
		messageRepo:    messageRepo,
		registry:       registry,
		ingestSvc:      ingestSvc,
		messageSvc:     messageSvc,
	}
}

func (h *SMSWebhookHandler) Receive(c *fiber.Ctx) error {
	provider := c.Params("provider")
	identifier := c.Params("identifier")

	if provider == "" || identifier == "" {
		return c.SendStatus(404)
	}

	ch, err := h.channelSMSRepo.FindByWebhookIdentifier(c.Context(), identifier)
	if err != nil {
		logger.Warn().Str("component", "sms_webhook").Str("identifier", identifier).Msg("channel not found")
		return c.SendStatus(200)
	}

	if ch.Provider != provider {
		logger.Warn().Str("component", "sms_webhook").
			Str("expected", ch.Provider).
			Str("got", provider).
			Msg("provider mismatch")
		return c.SendStatus(404)
	}

	prov, err := h.registry.Get(provider)
	if err != nil {
		logger.Error().Str("component", "sms_webhook").Err(err).Msg("provider not registered")
		return c.SendStatus(500)
	}

	httpReq := fiberToHTTPRequest(c)
	if err := prov.VerifyWebhook(httpReq, ch); err != nil {
		logger.Warn().Str("component", "sms_webhook").Str("provider", provider).Err(err).Msg("signature verification failed")
		return c.SendStatus(401)
	}

	inbound, err := prov.ParseInbound(httpReq)
	if err != nil {
		logger.Warn().Str("component", "sms_webhook").Err(err).Msg("parse inbound failed")
		return c.SendStatus(200)
	}

	if err := h.ingestSvc.IngestInbound(c.Context(), ch, inbound); err != nil {
		logger.Error().Str("component", "sms_webhook").Err(err).Msg("ingest inbound failed")
	}

	return c.SendStatus(200)
}

func (h *SMSWebhookHandler) Status(c *fiber.Ctx) error {
	provider := c.Params("provider")
	identifier := c.Params("identifier")

	if provider == "" || identifier == "" {
		return c.SendStatus(404)
	}

	ch, err := h.channelSMSRepo.FindByWebhookIdentifier(c.Context(), identifier)
	if err != nil {
		logger.Warn().Str("component", "sms_webhook_status").Str("identifier", identifier).Msg("channel not found")
		return c.SendStatus(200)
	}

	if ch.Provider != provider {
		return c.SendStatus(404)
	}

	prov, err := h.registry.Get(provider)
	if err != nil {
		return c.SendStatus(500)
	}

	httpReq := fiberToHTTPRequest(c)
	if err := prov.VerifyWebhook(httpReq, ch); err != nil {
		logger.Warn().Str("component", "sms_webhook_status").Str("provider", provider).Err(err).Msg("signature verification failed")
		return c.SendStatus(401)
	}

	cb, err := prov.ParseDeliveryStatus(httpReq)
	if err != nil {
		logger.Warn().Str("component", "sms_webhook_status").Err(err).Msg("parse delivery status failed")
		return c.SendStatus(200)
	}

	msg, err := h.messageRepo.FindBySourceID(c.Context(), cb.SourceID, ch.AccountID)
	if err != nil {
		logger.Info().Str("component", "sms_webhook_status").Str("sourceId", cb.SourceID).Msg("message not found")
		return c.SendStatus(200)
	}

	var extErr *string
	if cb.ExternalError != "" {
		extErr = &cb.ExternalError
	}

	if _, err := h.messageSvc.UpdateStatus(c.Context(), msg.ID, ch.AccountID, cb.Status, extErr); err != nil {
		logger.Error().Str("component", "sms_webhook_status").Err(err).Msg("update message status failed")
		return c.SendStatus(200)
	}

	return c.SendStatus(200)
}

func fiberToHTTPRequest(c *fiber.Ctx) *http.Request {
	protocol := c.Protocol()
	host := c.Hostname()
	fullURL := fmt.Sprintf("%s://%s%s", protocol, host, c.Path())

	headers := make(http.Header)
	c.Request().Header.VisitAll(func(key, value []byte) {
		headers.Set(string(key), string(value))
	})

	form := make(url.Values)
	c.Request().PostArgs().VisitAll(func(key, value []byte) {
		form.Add(string(key), string(value))
	})

	body := c.Body()
	r := &http.Request{
		Method: c.Method(),
		URL:    mustParseURL(fullURL),
		Header: headers,
		Form:   form,
	}
	r = r.WithContext(c.Context())

	if len(body) > 0 {
		r.Body = io.NopCloser(bytes.NewReader(body))
	} else {
		r.Body = http.NoBody
	}

	return r
}

func mustParseURL(raw string) *url.URL {
	u, err := url.Parse(raw)
	if err != nil {
		return &url.URL{}
	}
	return u
}
