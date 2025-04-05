package models

import (
	"gorm.io/gorm"
)

type Meeting struct {
	gorm.Model
	Date        string
	StartTime   string
	Duration    int
	Description string
	Link        string
}
