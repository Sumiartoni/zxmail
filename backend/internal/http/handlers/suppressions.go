package handlers

import "github.com/gin-gonic/gin"

type SuppressionsHandler struct{}

func NewSuppressionsHandler() *SuppressionsHandler {
	return &SuppressionsHandler{}
}

func (h *SuppressionsHandler) List(c *gin.Context) {
	notImplemented(c, "suppression list")
}

func (h *SuppressionsHandler) Create(c *gin.Context) {
	notImplemented(c, "suppression create")
}

func (h *SuppressionsHandler) Release(c *gin.Context) {
	notImplemented(c, "suppression release")
}
