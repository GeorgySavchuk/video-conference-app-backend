package controllers

import (
	"errors"
	"net/http"
	netmail "net/mail"
	"strings"

	"github.com/GeorgySavchuk/video-conference-app-backend/initializers"
	apimail "github.com/GeorgySavchuk/video-conference-app-backend/mail"
	"github.com/GeorgySavchuk/video-conference-app-backend/models"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// findMeetingByRoomID ищет встречу по room_id или по room id внутри поля link.
func findMeetingByRoomID(roomID string) (models.Meeting, error) {
	var meeting models.Meeting
	rid := strings.ToLower(strings.TrimSpace(roomID))
	err := initializers.DB.Where("LOWER(room_id) = ?", rid).First(&meeting).Error
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return meeting, err
		}
		p1 := "%/meeting/" + rid + "%"
		p2 := "%/room/" + rid + "%"
		err = initializers.DB.Where("LOWER(link) LIKE LOWER(?) OR LOWER(link) LIKE LOWER(?)", p1, p2).First(&meeting).Error
	}
	return meeting, err
}

// MeetingReminderStatus проверяет подписку на напоминание для email и комнаты.
// @Summary      Статус подписки на напоминание
// @Tags         meetings
// @Produce      json
// @Param        room_id  query  string  true  "ID комнаты"
// @Param        email    query  string  true  "Email"
// @Success      200  {object}  ReminderStatusSuccess
// @Failure      400  {object}  ErrorJSON
// @Failure      404  {object}  ErrorJSON
// @Failure      500  {object}  ErrorJSON
// @Router       /meetings/reminders/status [get]
func MeetingReminderStatus(c *gin.Context) {
	roomID := strings.TrimSpace(c.Query("room_id"))
	email := strings.TrimSpace(strings.ToLower(c.Query("email")))
	if roomID == "" || email == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Нужны room_id и email"})
		return
	}
	if _, err := netmail.ParseAddress(email); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Некорректный email"})
		return
	}

	meeting, err := findMeetingByRoomID(roomID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Встреча по этой ссылке не найдена"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка поиска встречи"})
		return
	}

	var count int64
	if err := initializers.DB.Model(&models.MeetingReminder{}).
		Where("meeting_id = ? AND email = ?", meeting.ID, email).
		Count(&count).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка проверки подписки"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"subscribed": count > 0})
}

// SubscribeMeetingReminder создаёт подписку на email-напоминание за ~15 минут до встречи.
// @Summary      Подписаться на напоминание
// @Tags         meetings
// @Accept       json
// @Produce      json
// @Param        body  body  SubscribeReminderBody  true  "room_id и email"
// @Success      200  {object}  SimpleMessageSuccess  "Уже подписан"
// @Success      201  {object}  SubscribeReminderCreated
// @Failure      400  {object}  ErrorJSON
// @Failure      404  {object}  ErrorJSON
// @Failure      500  {object}  ErrorJSON
// @Router       /meetings/reminders/subscribe [post]
func SubscribeMeetingReminder(c *gin.Context) {
	var body struct {
		RoomID string `json:"room_id" binding:"required"`
		Email  string `json:"email" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Нужны room_id и email"})
		return
	}
	roomID := strings.ToLower(strings.TrimSpace(body.RoomID))
	email := strings.TrimSpace(strings.ToLower(body.Email))
	if _, err := netmail.ParseAddress(email); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Некорректный email"})
		return
	}

	meeting, err := findMeetingByRoomID(roomID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Встреча по этой ссылке не найдена"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка поиска встречи"})
		return
	}

	var existing models.MeetingReminder
	err = initializers.DB.Where("meeting_id = ? AND email = ?", meeting.ID, email).First(&existing).Error
	if err == nil {
		c.JSON(http.StatusOK, gin.H{"message": "Вы уже подписаны на напоминания по этой почте"})
		return
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка проверки подписки"})
		return
	}

	rec := models.MeetingReminder{
		MeetingID: meeting.ID,
		Email:     email,
	}
	if err := initializers.DB.Create(&rec).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Не удалось сохранить подписку"})
		return
	}

	subj := "HellConf: напоминание о встрече включено"
	bodyTxt := "Вы подписались на напоминание за ~15 минут до начала встречи.\n\n"
	if strings.TrimSpace(meeting.Link) != "" {
		bodyTxt += "Ссылка:\n" + strings.TrimSpace(meeting.Link) + "\n"
	}
	if err := apimail.SendMeetingInvite(email, subj, bodyTxt); err != nil && err != apimail.ReminderDisabled {
		// подписка уже сохранена; ошибку SMTP пользователю не показываем как фатальную
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "За ~15 минут до начала встречи отправим напоминание на " + email,
	})
}
