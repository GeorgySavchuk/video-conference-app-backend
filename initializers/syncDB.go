package initializers

import "github.com/GeorgySavchuk/video-conference-app-backend/models"

func SyncDB() {
	DB.AutoMigrate(
		&models.User{},
		&models.Meeting{},
	)
}
