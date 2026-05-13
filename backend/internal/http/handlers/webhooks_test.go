package handlers

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	webhooksmodule "zxmail/backend/internal/modules/webhooks"
)

type stubWebhookService struct {
	processFn func(ctx context.Context, payload []byte) (*webhooksmodule.ProcessResult, error)
}

func (s *stubWebhookService) ProcessPostalEvent(ctx context.Context, payload []byte) (*webhooksmodule.ProcessResult, error) {
	return s.processFn(ctx, payload)
}

func TestPostalWebhookRejectsInvalidSignature(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := NewPostalWebhookHandler("secret", &stubWebhookService{
		processFn: func(ctx context.Context, payload []byte) (*webhooksmodule.ProcessResult, error) {
			return &webhooksmodule.ProcessResult{}, nil
		},
	})

	router := gin.New()
	router.POST("/webhooks/postal/event", handler.Receive)

	payload := []byte(`{"event":"accepted"}`)
	request := httptest.NewRequest(http.MethodPost, "/webhooks/postal/event", bytes.NewReader(payload))
	request.Header.Set("X-Webhook-Signature", "bad-signature")
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", response.Code)
	}
}

func TestPostalWebhookAcceptsValidSignature(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := NewPostalWebhookHandler("secret", &stubWebhookService{
		processFn: func(ctx context.Context, payload []byte) (*webhooksmodule.ProcessResult, error) {
			return &webhooksmodule.ProcessResult{
				Status:        "accepted",
				Duplicate:     false,
				PostalMessage: "msg-1",
			}, nil
		},
	})

	router := gin.New()
	router.POST("/webhooks/postal/event", handler.Receive)

	payload := []byte(`{"event":"accepted","message":{"id":"msg-1"}}`)
	request := httptest.NewRequest(http.MethodPost, "/webhooks/postal/event", bytes.NewReader(payload))
	request.Header.Set("X-Webhook-Signature", signPayload(payload, "secret"))
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d", response.Code)
	}
}

func TestPostalWebhookAcceptsSha256PrefixedSignature(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := NewPostalWebhookHandler("secret", &stubWebhookService{
		processFn: func(ctx context.Context, payload []byte) (*webhooksmodule.ProcessResult, error) {
			return &webhooksmodule.ProcessResult{
				Status:        "accepted",
				Duplicate:     false,
				PostalMessage: "msg-1",
			}, nil
		},
	})

	router := gin.New()
	router.POST("/webhooks/postal/event", handler.Receive)

	payload := []byte(`{"event":"accepted","message":{"id":"msg-1"}}`)
	request := httptest.NewRequest(http.MethodPost, "/webhooks/postal/event", bytes.NewReader(payload))
	request.Header.Set("X-Webhook-Signature", "sha256="+signPayload(payload, "secret"))
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d", response.Code)
	}
}

func signPayload(payload []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	return hex.EncodeToString(mac.Sum(nil))
}
