package handlers

import "github.com/gin-gonic/gin"

type AdminHandler struct{}

func NewAdminHandler() *AdminHandler {
	return &AdminHandler{}
}

func (h *AdminHandler) Overview(c *gin.Context) {
	notImplemented(c, "admin dashboard overview")
}
