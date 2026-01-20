package handler

import (
	"io"
	"net/http"

	"github.com/ayteuir/backend/internal/pkg/logger"
	"github.com/ayteuir/backend/internal/service"
)

type WebhookHandler struct {
	webhookService *service.WebhookService
}

func NewWebhookHandler(webhookService *service.WebhookService) *WebhookHandler {
	return &WebhookHandler{
		webhookService: webhookService,
	}
}

func (h *WebhookHandler) Verify(w http.ResponseWriter, r *http.Request) {
	mode := r.URL.Query().Get("hub.mode")
	token := r.URL.Query().Get("hub.verify_token")
	challenge := r.URL.Query().Get("hub.challenge")

	if mode == "" || token == "" || challenge == "" {
		logger.Warn().Msg("Webhook verification missing parameters")
		Error(w, http.StatusBadRequest, "MISSING_PARAMS", "Missing verification parameters")
		return
	}

	response, ok := h.webhookService.VerifyChallenge(mode, token, challenge)
	if !ok {
		logger.Warn().Str("mode", mode).Msg("Webhook verification failed")
		Error(w, http.StatusForbidden, "VERIFICATION_FAILED", "Invalid verify token")
		return
	}

	logger.Info().Msg("Webhook verified successfully")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(response))
}

func (h *WebhookHandler) Handle(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to read webhook body")
		Error(w, http.StatusBadRequest, "READ_ERROR", "Failed to read request body")
		return
	}

	signature := r.Header.Get("X-Hub-Signature-256")
	if signature == "" {
		logger.Warn().Msg("Missing webhook signature")
		Error(w, http.StatusUnauthorized, "MISSING_SIGNATURE", "Missing signature header")
		return
	}

	if !h.webhookService.VerifySignature(body, signature) {
		logger.Warn().Msg("Invalid webhook signature")
		Error(w, http.StatusUnauthorized, "INVALID_SIGNATURE", "Invalid signature")
		return
	}

	w.WriteHeader(http.StatusOK)

	go func() {
		if err := h.webhookService.ProcessWebhook(r.Context(), body); err != nil {
			logger.Error().Err(err).Msg("Failed to process webhook")
		}
	}()
}
