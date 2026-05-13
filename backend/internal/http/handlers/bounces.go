package handlers

import "github.com/gin-gonic/gin"

type BouncesHandler struct{}

func NewBouncesHandler() *BouncesHandler {
	return &BouncesHandler{}
}

func (h *BouncesHandler) List(c *gin.Context) {
	notImplemented(c, "bounce handling list")
}
