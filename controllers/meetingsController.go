package controllers

import (
	"errors"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/GeorgySavchuk/video-conference-app-backend/initializers"
	"github.com/GeorgySavchuk/video-conference-app-backend/models"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// CreateMeeting создаёт встречу и опционально рассылает приглашения.
// @Summary      Создать встречу
// @Description  Дата ДД.ММ.ГГГГ, время ЧЧ:ММ. При пересечении слотов — 409.
// @Tags         meetings
// @Accept       json
// @Produce      json
// @Param        body  body      CreateMeetingBody  true  "Данные встречи"
// @Success      200   {object}  CreateMeetingSuccess
// @Failure      400   {object}  ErrorJSON
// @Failure      409   {object}  ErrorJSON
// @Failure      500   {object}  ErrorJSON
// @Router       /meetings [post]
func CreateMeeting(c *gin.Context) {
	var body struct {
		CreatorID    string   `json:"creator_id" binding:"required"`
		RoomID       string   `json:"room_id"`
		Date         string   `json:"date" binding:"required"`
		StartTime    string   `json:"start_time" binding:"required"`
		Duration     int      `json:"duration" binding:"required"`
		Description  string   `json:"description"`
		Link         string   `json:"link"`
		InviteEmails []string `json:"invite_emails"`
	}

	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Неверные данные запроса: " + err.Error(),
		})
		return
	}

	if _, err := time.Parse("02.01.2006", body.Date); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Неверный формат даты. Ожидается ДД.ММ.ГГГГ",
		})
		return
	}

	if _, err := time.Parse("15:04", body.StartTime); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Неверный формат времени. Ожидается ЧЧ:ММ",
		})
		return
	}

	var conflictingMeetings int64
	startTime, _ := time.Parse("15:04", body.StartTime)
	endTime := startTime.Add(time.Minute * time.Duration(body.Duration))
	endTimeStr := endTime.Format("15:04")

	initializers.DB.Model(&models.Meeting{}).
		Where("creator_id = ?", body.CreatorID).
		Where("date = ?", body.Date).
		Where("start_time < ? AND (start_time::time + (duration * interval '1 minute')::interval)::text > ?", endTimeStr, body.StartTime).
		Count(&conflictingMeetings)

	if conflictingMeetings > 0 {
		c.JSON(http.StatusConflict, gin.H{
			"error": "Встреча пересекается с существующими встречами пользователя",
		})
		return
	}

	meeting := models.Meeting{
		CreatorID:   body.CreatorID,
		RoomID:      strings.TrimSpace(body.RoomID),
		Date:        body.Date,
		StartTime:   body.StartTime,
		Duration:    body.Duration,
		Description: body.Description,
		Link:        body.Link,
	}

	if err := initializers.DB.Create(&meeting).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Не удалось создать встречу",
		})
		return
	}

	sent, failed := AfterCreateMeetingInvites(meeting, body.InviteEmails)

	c.JSON(http.StatusOK, gin.H{
		"message":                "Встреча успешно создана",
		"data":                   meeting,
		"invite_emails_sent":     sent,
		"invite_emails_failed":   failed,
	})
}

// GetAllUpcomingMeetings возвращает встречи создателя, у которых конец слота ещё не прошёл.
// @Summary      Предстоящие встречи
// @Tags         meetings
// @Produce      json
// @Param        creator_id  query  string  true  "ID создателя (UUID)"
// @Success      200  {object}  MeetingsUpcomingResponse
// @Failure      400  {object}  ErrorJSON
// @Failure      500  {object}  ErrorJSON
// @Router       /meetings/upcoming [get]
func GetAllUpcomingMeetings(c *gin.Context) {
	creatorID := c.Query("creator_id")
	if creatorID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Необходимо указать creator_id",
		})
		return
	}

	// Список «предстоящих»: слот ещё не закончился (start + duration > now).
	// Сравнение через meetingStartLocal совпадает с письмами/напоминаниями и учитывает duration.
	var all []models.Meeting
	if err := initializers.DB.Where("creator_id = ?", creatorID).Find(&all).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Не удалось получить список встреч",
		})
		return
	}

	now := time.Now()
	meetings := make([]models.Meeting, 0, len(all))
	for _, m := range all {
		start, err := meetingStartLocal(m)
		if err != nil {
			continue
		}
		dur := m.Duration
		if dur < 0 {
			dur = 0
		}
		end := start.Add(time.Duration(dur) * time.Minute)
		if end.After(now) {
			meetings = append(meetings, m)
		}
	}

	sort.Slice(meetings, func(i, j int) bool {
		si, errI := meetingStartLocal(meetings[i])
		sj, errJ := meetingStartLocal(meetings[j])
		if errI != nil || errJ != nil {
			return false
		}
		return si.Before(sj)
	})

	c.JSON(http.StatusOK, gin.H{
		"meetings": meetings,
	})
}

// GetCurrentMeeting возвращает активную на сейчас встречу или пустой meeting.
// @Summary      Текущая встреча
// @Description  Встреча на сегодня в интервале [start, start+duration).
// @Tags         meetings
// @Produce      json
// @Param        creator_id  query  string  true  "ID создателя"
// @Success      200  {object}  CurrentMeetingResponse
// @Failure      400  {object}  ErrorJSON
// @Failure      500  {object}  ErrorJSON
// @Router       /meetings/current [get]
func GetCurrentMeeting(c *gin.Context) {
	creatorID := c.Query("creator_id")
	if creatorID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Необходимо указать creator_id",
		})
		return
	}

	now := time.Now()
	currentDate := now.Format("02.01.2006")
	currentTime := now.Format("15:04")

	var meeting models.Meeting

	// Сравнение через ::time, не через ::text — иначе границы слота и «хвост» вида :ss ломают порядок.
	err := initializers.DB.
		Where("creator_id = ?", creatorID).
		Where("date = ?", currentDate).
		Where("start_time::time <= ?::time", currentTime).
		Where("(start_time::time + (duration * interval '1 minute'))::time > ?::time", currentTime).
		First(&meeting).
		Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusOK, gin.H{
				"message": "Сейчас нет активных встреч",
				"meeting": nil,
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Ошибка при поиске текущей встречи",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"meeting": meeting,
	})
}

// UpdateMeeting обновляет поля встречи при совпадении creator_id.
// @Summary      Обновить встречу
// @Tags         meetings
// @Accept       json
// @Produce      json
// @Param        id    path      string             true  "ID встречи"
// @Param        body  body      UpdateMeetingBody  true  "Поля для обновления"
// @Success      200   {object}  UpdateMeetingSuccess
// @Failure      400   {object}  ErrorJSON
// @Failure      404   {object}  ErrorJSON
// @Failure      409   {object}  ErrorJSON
// @Failure      500   {object}  ErrorJSON
// @Router       /meetings/{id} [put]
func UpdateMeeting(c *gin.Context) {
	id := c.Param("id")

	var body struct {
		CreatorID   string `json:"creator_id" binding:"required"`
		Date        string `json:"date"`
		StartTime   string `json:"start_time"`
		Duration    int    `json:"duration"`
		Description string `json:"description"`
		Link        string `json:"link"`
	}

	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Неверный формат данных",
		})
		return
	}

	var meeting models.Meeting
	if err := initializers.DB.Where("creator_id = ?", body.CreatorID).First(&meeting, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Встреча не найдена или у вас нет прав на её изменение",
		})
		return
	}

	if body.Date != "" || body.StartTime != "" || body.Duration != 0 {
		date := body.Date
		if date == "" {
			date = meeting.Date
		}

		startTime := body.StartTime
		if startTime == "" {
			startTime = meeting.StartTime
		}

		duration := body.Duration
		if duration == 0 {
			duration = meeting.Duration
		}

		parsedStartTime, err := time.Parse("15:04", startTime)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Неверный формат времени начала",
			})
			return
		}

		endTime := parsedStartTime.Add(time.Minute * time.Duration(duration))
		endTimeStr := endTime.Format("15:04")

		var conflicts int64
		initializers.DB.Model(&models.Meeting{}).
			Where("id != ?", id).
			Where("creator_id = ?", body.CreatorID).
			Where("date = ?", date).
			Where("NOT ((start_time::time + (duration * interval '1 minute')::interval)::text <= ? OR start_time >= ?)",
				startTime,
				endTimeStr).
			Count(&conflicts)

		if conflicts > 0 {
			c.JSON(http.StatusConflict, gin.H{
				"error": "Обновлённые данные конфликтуют с другими встречами пользователя",
			})
			return
		}
	}

	if body.Date != "" {
		meeting.Date = body.Date
	}

	if body.StartTime != "" {
		meeting.StartTime = body.StartTime
	}

	if body.Duration != 0 {
		meeting.Duration = body.Duration
	}

	if body.Description != "" {
		meeting.Description = body.Description
	}

	if body.Link != "" {
		meeting.Link = body.Link
	}

	if err := initializers.DB.Save(&meeting).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Не удалось обновить встречу",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Встреча успешно обновлена",
		"data":    meeting,
	})
}

// DeleteMeeting удаляет встречу и напоминания по ней.
// @Summary      Удалить встречу
// @Tags         meetings
// @Accept       json
// @Produce      json
// @Param        id    path      string               true  "ID встречи"
// @Param        body  body      DeleteMeetingBody    true  "creator_id"
// @Success      200   {object}  DeleteMeetingSuccess
// @Failure      400   {object}  ErrorJSON
// @Failure      404   {object}  ErrorJSON
// @Failure      500   {object}  ErrorJSON
// @Router       /meetings/{id} [delete]
func DeleteMeeting(c *gin.Context) {
	id := c.Param("id")

	var body struct {
		CreatorID string `json:"creator_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Необходимо указать creator_id",
		})
		return
	}

	var meeting models.Meeting
	if err := initializers.DB.Where("creator_id = ?", body.CreatorID).First(&meeting, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Встреча не найдена или у вас нет прав на её удаление",
		})
		return
	}

	sent, failed := NotifySubscribersMeetingCancelled(meeting)

	if err := initializers.DB.Where("meeting_id = ?", meeting.ID).Delete(&models.MeetingReminder{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Не удалось удалить подписки на напоминания",
		})
		return
	}

	if err := initializers.DB.Delete(&meeting).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Не удалось удалить встречу",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":                    "Встреча успешно удалена",
		"cancellation_emails_sent":   sent,
		"cancellation_emails_failed":   failed,
	})
}

// GetMeetingByCode ищет встречу по суффиксу кода в поле link.
// @Summary      Встреча по коду ссылки
// @Description  Параметр id — суффикс после /meeting/ или /room/ в link.
// @Tags         meetings
// @Produce      json
// @Param        id   path  string  true  "Код из ссылки"
// @Success      200  {object}  GetMeetingByCodeSuccess
// @Failure      404  {object}  ErrorJSON
// @Failure      500  {object}  ErrorJSON
// @Router       /meetings/{id} [get]
func GetMeetingByCode(c *gin.Context) {
	code := c.Param("id")

	var meeting models.Meeting
	if err := initializers.DB.Where("link LIKE ?", "%"+code).First(&meeting).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Встреча не найдена",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Ошибка при получении информации о встрече",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"meeting": meeting,
	})
}
