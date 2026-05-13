package handlers

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	webhooksmodule "zxmail/backend/internal/modules/webhooks"
)

type PostalWebhookHandler struct {
	signingSecret string
	service       PostalWebhookService
}

type PostalWebhookService interface {
	ProcessPostalEvent(ctx context.Context, payload []byte) (*webhooksmodule.ProcessResult, error)
}

func NewPostalWebhookHandler(signingSecret string, service PostalWebhookService) *PostalWebhookHandler {
	return &PostalWebhookHandler{signingSecret: signingSecret, service: service}
}

func (h *PostalWebhookHandler) Receive(c *gin.Context) {
	payload, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	signature := c.GetHeader("X-Webhook-Signature")
	if !validateHMAC(payload, signature, h.signingSecret) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid webhook signature"})
		return
	}

	result, err := h.service.ProcessPostalEvent(c.Request.Context(), payload)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to process webhook"})
		return
	}

	c.JSON(http.StatusAccepted, result)
}

func validateHMAC(payload []byte, signature string, secret string) bool {
	if secret == "" || signature == "" {
		return false
	}

	signature = strings.TrimSpace(signature)
	signature = strings.TrimPrefix(strings.ToLower(signature), "sha256=")
	provided, err := hex.DecodeString(signature)
	if err != nil {
		return false
	}

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	expected := mac.Sum(nil)
	return hmac.Equal(expected, provided)
}
