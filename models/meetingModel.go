package models

import (
	"time"

	"gorm.io/gorm"
)

type Meeting struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	CreatorID   string `json:"creator_id" gorm:"index"`
	RoomID      string `json:"room_id" gorm:"size:64;index"`
	Date        string `json:"date"`
	StartTime   string `json:"start_time"`
	Duration    int    `json:"duration"`
	Description string `json:"description"`
	Link        string `json:"link"`
}
