package models

import (
	"time"

	"gorm.io/gorm"
)

// MeetingReminder — подписка на напоминание о встрече по почте (по ссылке с room id).
type MeetingReminder struct {
	gorm.Model

	MeetingID uint   `gorm:"not null;uniqueIndex:ux_meeting_email" json:"meeting_id"`
	Email     string `gorm:"size:320;not null;uniqueIndex:ux_meeting_email" json:"email"`

	/** Заполняется после успешной отправки письма «скоро начало». */
	ReminderSentAt *time.Time `json:"reminder_sent_at,omitempty"`
}

func (MeetingReminder) TableName() string {
	return "meeting_reminders"
}
