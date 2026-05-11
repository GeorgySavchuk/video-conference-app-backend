package controllers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// CreateRoom выдаёт новый UUID комнаты для mediasoup.
// @Summary      Создать идентификатор комнаты
// @Description  Только генерация roomId; медиасервер — отдельный сервис.
// @Tags         rooms
// @Produce      json
// @Success      200  {object}  CreateRoomResponse
// @Router       /rooms [post]
func CreateRoom(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"roomId": uuid.New().String(),
	})
}
