package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func notImplemented(c *gin.Context, feature string) {
	c.JSON(http.StatusNotImplemented, gin.H{
		"message": "scaffold created; implementation pending",
		"feature": feature,
	})
}
