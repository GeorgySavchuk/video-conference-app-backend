package initializers

import (
	"log"

	"github.com/GeorgySavchuk/video-conference-app-backend/models"
)

func SyncDB() {
	DB.AutoMigrate(
		&models.User{},
		&models.Meeting{},
		&models.MeetingReminder{},
		&models.RoomChatMessage{},
		&models.RoomChatRead{},
	)
	// Таблица short_links больше не используется (короткие ссылки убраны).
	if err := DB.Exec("DROP TABLE IF EXISTS short_links").Error; err != nil {
		log.Printf("sync: drop legacy short_links: %v", err)
	}
}
