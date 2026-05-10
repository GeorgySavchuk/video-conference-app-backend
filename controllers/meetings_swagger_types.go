package controllers

import "github.com/GeorgySavchuk/video-conference-app-backend/models"

// ErrorJSON стандартный JSON с полем error.
type ErrorJSON struct {
	Error string `json:"error" example:"Неверные данные запроса"`
}

// CreateMeetingBody тело запроса POST /meetings.
type CreateMeetingBody struct {
	CreatorID    string   `json:"creator_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	RoomID       string   `json:"room_id"`
	Date         string   `json:"date" example:"15.05.2006"`
	StartTime    string   `json:"start_time" example:"14:30"`
	Duration     int      `json:"duration" example:"60"`
	Description  string   `json:"description"`
	Link         string   `json:"link"`
	InviteEmails []string `json:"invite_emails"`
}

// CreateMeetingSuccess ответ при успешном создании встречи.
type CreateMeetingSuccess struct {
	Message              string         `json:"message"`
	Data                 models.Meeting `json:"data"`
	InviteEmailsSent     int            `json:"invite_emails_sent"`
	InviteEmailsFailed   int            `json:"invite_emails_failed"`
}

// MeetingsUpcomingResponse список предстоящих встреч.
type MeetingsUpcomingResponse struct {
	Meetings []models.Meeting `json:"meetings"`
}

// CurrentMeetingResponse текущая встреча или пустой ответ (meeting может быть null).
type CurrentMeetingResponse struct {
	Message string          `json:"message,omitempty"`
	Meeting *models.Meeting `json:"meeting"`
}

// UpdateMeetingBody тело PUT /meetings/:id.
type UpdateMeetingBody struct {
	CreatorID   string `json:"creator_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Date        string `json:"date"`
	StartTime   string `json:"start_time"`
	Duration    int    `json:"duration"`
	Description string `json:"description"`
	Link        string `json:"link"`
}

// UpdateMeetingSuccess ответ после обновления.
type UpdateMeetingSuccess struct {
	Message string         `json:"message"`
	Data    models.Meeting `json:"data"`
}

// DeleteMeetingBody тело DELETE /meetings/:id.
type DeleteMeetingBody struct {
	CreatorID string `json:"creator_id" example:"550e8400-e29b-41d4-a716-446655440000"`
}

// DeleteMeetingSuccess ответ после удаления.
type DeleteMeetingSuccess struct {
	Message                  string `json:"message"`
	CancellationEmailsSent   int    `json:"cancellation_emails_sent"`
	CancellationEmailsFailed int    `json:"cancellation_emails_failed"`
}

// GetMeetingByCodeSuccess встреча по коду из ссылки.
type GetMeetingByCodeSuccess struct {
	Meeting models.Meeting `json:"meeting"`
}

// ReminderStatusSuccess ответ GET .../reminders/status.
type ReminderStatusSuccess struct {
	Subscribed bool `json:"subscribed"`
}

// SubscribeReminderBody тело POST .../reminders/subscribe.
type SubscribeReminderBody struct {
	RoomID string `json:"room_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Email  string `json:"email" example:"user@example.com"`
}

// SubscribeReminderCreated ответ 201 при новой подписке.
type SubscribeReminderCreated struct {
	Message string `json:"message"`
}

// SimpleMessageSuccess ответ только с message.
type SimpleMessageSuccess struct {
	Message string `json:"message"`
}
