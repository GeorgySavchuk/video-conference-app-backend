package controllers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// CreateRoom returns a new unique room id for mediasoup sessions (no external VideoSDK).
func CreateRoom(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"roomId": uuid.New().String(),
	})
}
