package controllers

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"unicode"

	"github.com/GeorgySavchuk/video-conference-app-backend/initializers"
	"github.com/GeorgySavchuk/video-conference-app-backend/models"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func roomChatMessageJSON(m *models.RoomChatMessage) gin.H {
	peerID := ""
	if m.UserID != 0 {
		peerID = "user-" + strconv.FormatUint(uint64(m.UserID), 10)
	} else if strings.TrimSpace(m.GuestPeerID) != "" {
		peerID = strings.TrimSpace(m.GuestPeerID)
	} else {
		peerID = "guest-" + strconv.FormatUint(uint64(m.ID), 10)
	}
	return gin.H{
		"id":          strconv.FormatUint(uint64(m.ID), 10),
		"peerId":      peerID,
		"displayName": m.DisplayName,
		"text":        m.Body,
		"ts":          m.CreatedAt.UnixMilli(),
		"userId":      m.UserID,
	}
}

// ListRoomChat GET /rooms/:roomId/chat — история доступна всем (гости без «прочитано»).
func ListRoomChat(c *gin.Context) {
	roomID := strings.TrimSpace(c.Param("roomId"))
	if roomID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Не указан идентификатор комнаты"})
		return
	}

	var msgs []models.RoomChatMessage
	if err := initializers.DB.Where("room_id = ?", roomID).Order("id ASC").Limit(500).Find(&msgs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Не удалось загрузить сообщения"})
		return
	}

	out := make([]gin.H, 0, len(msgs))
	for i := range msgs {
		out = append(out, roomChatMessageJSON(&msgs[i]))
	}

	lastReadID := uint(0)
	if u, ok := c.Get("user"); ok {
		user := u.(models.User)
		var read models.RoomChatRead
		if err := initializers.DB.Where("user_id = ? AND room_id = ?", user.ID, roomID).First(&read).Error; err == nil {
			lastReadID = read.LastReadMessageID
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"messages":          out,
		"lastReadMessageId": strconv.FormatUint(uint64(lastReadID), 10),
	})
}

// PutRoomChatRead PUT /rooms/:roomId/chat/read — обновить курсор «прочитано до сообщения id».
func PutRoomChatRead(c *gin.Context) {
	roomID := strings.TrimSpace(c.Param("roomId"))
	if roomID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Не указан идентификатор комнаты"})
		return
	}

	u, ok := c.Get("user")
	if !ok {
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}
	user := u.(models.User)

	var body struct {
		LastReadMessageID uint `json:"lastReadMessageId"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверное тело запроса"})
		return
	}

	if body.LastReadMessageID > 0 {
		var cnt int64
		initializers.DB.Model(&models.RoomChatMessage{}).
			Where("room_id = ? AND id = ?", roomID, body.LastReadMessageID).
			Count(&cnt)
		if cnt == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Сообщение не найдено в этой комнате"})
			return
		}
	}

	var read models.RoomChatRead
	err := initializers.DB.Where("user_id = ? AND room_id = ?", user.ID, roomID).First(&read).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		read = models.RoomChatRead{
			UserID:            user.ID,
			RoomID:            roomID,
			LastReadMessageID: body.LastReadMessageID,
		}
		if err := initializers.DB.Create(&read).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Не удалось сохранить отметку"})
			return
		}
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка БД"})
		return
	} else {
		if body.LastReadMessageID > read.LastReadMessageID {
			read.LastReadMessageID = body.LastReadMessageID
			if err := initializers.DB.Save(&read).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Не удалось обновить отметку"})
				return
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"lastReadMessageId": strconv.FormatUint(uint64(read.LastReadMessageID), 10),
	})
}

func isValidGuestPeerID(s string) bool {
	t := strings.TrimSpace(s)
	if len(t) < 8 || len(t) > 64 {
		return false
	}
	for _, r := range t {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' {
			continue
		}
		return false
	}
	return true
}

// PostRoomChat POST /rooms/:roomId/chat — залогиненный по cookie; иначе гость с guestPeerId + displayName.
func PostRoomChat(c *gin.Context) {
	roomID := strings.TrimSpace(c.Param("roomId"))
	if roomID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Не указан идентификатор комнаты"})
		return
	}

	var raw struct {
		Text          string `json:"text"`
		DisplayName   string `json:"displayName"`
		GuestPeerID   string `json:"guestPeerId"`
	}
	if err := c.ShouldBindJSON(&raw); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверное тело запроса"})
		return
	}

	text := strings.TrimSpace(raw.Text)
	if text == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Пустое сообщение"})
		return
	}
	if len(text) > 2000 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Сообщение слишком длинное"})
		return
	}

	if u, ok := c.Get("user"); ok {
		user := u.(models.User)
		msg := models.RoomChatMessage{
			RoomID:      roomID,
			UserID:      user.ID,
			DisplayName: user.Name,
			Body:        text,
		}
		if err := initializers.DB.Create(&msg).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Не удалось сохранить сообщение"})
			return
		}
		c.JSON(http.StatusCreated, gin.H{"message": roomChatMessageJSON(&msg)})
		return
	}

	dn := strings.TrimSpace(raw.DisplayName)
	guestPeer := strings.TrimSpace(raw.GuestPeerID)
	if dn == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Укажите имя (displayName)"})
		return
	}
	if len(dn) > 128 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Имя слишком длинное"})
		return
	}
	if guestPeer == "" || !isValidGuestPeerID(guestPeer) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Некорректный guestPeerId"})
		return
	}

	msg := models.RoomChatMessage{
		RoomID:      roomID,
		UserID:      0,
		GuestPeerID: guestPeer,
		DisplayName: dn,
		Body:        text,
	}
	if err := initializers.DB.Create(&msg).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Не удалось сохранить сообщение"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": roomChatMessageJSON(&msg)})
}
