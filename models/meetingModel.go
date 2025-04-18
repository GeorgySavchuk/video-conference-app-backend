package models

import (
	"gorm.io/gorm"
)

type Meeting struct {
	gorm.Model
	CreatorID   string `json:"creator_id" gorm:"index"`
	Date        string `json:"date"`
	StartTime   string `json:"start_time"`
	Duration    int    `json:"duration"`
	Description string `json:"description"`
	Link        string `json:"link"`
}
