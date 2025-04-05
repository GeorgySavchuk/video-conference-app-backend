package controllers

import (
	"errors"
	"net/http"
	"time"

	"github.com/GeorgySavchuk/video-conference-app-backend/initializers"
	"github.com/GeorgySavchuk/video-conference-app-backend/models"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func CreateMeeting(c *gin.Context) {
	var body struct {
		Date        string `json:"date"`
		StartTime   string `json:"start_time"`
		Duration    int    `json:"duration"`
		Description string `json:"description"`
		Link        string `json:"link"`
	}

	if err := c.Bind(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Не удалось спарсить тело запроса",
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
		Where("date = ?", body.Date).
		Where("NOT (ADDTIME(start_time, CONCAT(Duration, ':00')) <= ? OR start_time >= ?)", body.StartTime, endTimeStr).
		Count(&conflictingMeetings)

	if conflictingMeetings > 0 {
		c.JSON(http.StatusConflict, gin.H{
			"error": "Встреча пересекается с существующими",
		})

		return
	}

	meeting := models.Meeting{
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

	c.JSON(http.StatusOK, gin.H{
		"message": "Встреча успешно создана",
		"data":    meeting,
	})
}

func GetAllUpcomingMeetings(c *gin.Context) {
	now := time.Now()
	currentDate := now.Format("02.01.2006")
	currentTime := now.Format("15:04")

	var meetings []models.Meeting

	result := initializers.DB.Where(
		"(date = ? AND start_time > ?) OR date > ?",
		currentDate, currentTime, currentDate,
	).Order("date, start_time").Find(&meetings)

	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Не удалось получить список встреч",
		})

		return
	}

	c.JSON(http.StatusOK, gin.H{
		"meetings": meetings,
	})
}

func GetCurrentMeeting(c *gin.Context) {
	now := time.Now()
	currentDate := now.Format("02.01.2006")
	currentTime := now.Format("15:04")

	var meeting models.Meeting

	err := initializers.DB.
		Where("date = ?", currentDate).
		Where("start_time <= ?", currentTime).
		Where("ADDTIME(start_time, CONCAT(Duration, ':00')) > ?", currentTime).
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

func UpdateMeeting(c *gin.Context) {
	id := c.Param("id")

	var meeting models.Meeting
	if err := initializers.DB.First(&meeting, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Встреча не найдена",
		})
		return
	}

	var body struct {
		Date        string `json:"date"`
		StartTime   string `json:"start_time"`
		Duration    int    `json:"duration"`
		Description string `json:"description"`
		Link        string `json:"link"`
	}

	if err := c.Bind(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Неверный формат данных",
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
			Where("date = ?", date).
			Where("NOT (ADDTIME(start_time, CONCAT(Duration, ':00')) <= ? OR start_time >= ?)",
				startTime,
				endTimeStr).
			Count(&conflicts)

		if conflicts > 0 {
			c.JSON(http.StatusConflict, gin.H{
				"error": "Обновлённые данные конфликтуют с другими встречами",
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

func DeleteMeeting(c *gin.Context) {
	id := c.Param("id")

	var meeting models.Meeting
	if err := initializers.DB.First(&meeting, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Встреча не найдена",
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
		"message": "Встреча успешно удалена",
	})
}
