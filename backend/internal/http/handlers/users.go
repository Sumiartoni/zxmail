package handlers

import "github.com/gin-gonic/gin"

type UsersHandler struct{}

func NewUsersHandler() *UsersHandler {
	return &UsersHandler{}
}

func (h *UsersHandler) List(c *gin.Context) {
	notImplemented(c, "users list")
}
