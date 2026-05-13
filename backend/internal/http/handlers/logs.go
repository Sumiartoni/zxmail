package handlers

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"

	"zxmail/backend/internal/http/middleware"
	authmodule "zxmail/backend/internal/modules/auth"
	logsmodule "zxmail/backend/internal/modules/logs"
)

type LogsService interface {
	List(ctx context.Context, actor authmodule.AuthenticatedUser, filters logsmodule.Filters) ([]logsmodule.Entry, error)
}

type LogsHandler struct {
	service LogsService
}

func NewLogsHandler(service LogsService) *LogsHandler {
	return &LogsHandler{service: service}
}

func (h *LogsHandler) List(c *gin.Context) {
	actor, ok := middleware.ActorFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing actor"})
		return
	}

	entries, err := h.service.List(c.Request.Context(), actor, logsmodule.ParseFilters(c.Request.URL.Query()))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list logs"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"logs": entries})
}
