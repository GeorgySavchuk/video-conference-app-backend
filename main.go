package main

import (
	"fmt"
	"os"

	"github.com/GeorgySavchuk/video-conference-app-backend/controllers"
	"github.com/GeorgySavchuk/video-conference-app-backend/initializers"
	"github.com/GeorgySavchuk/video-conference-app-backend/middleware"
	"github.com/GeorgySavchuk/video-conference-app-backend/workers"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func init() {
	// .env в корне бэкенда (не коммитить; см. .env.example)
	_ = godotenv.Load()
	initializers.ConnectToDB()
	initializers.SyncDB()
}

func main() {
	workers.StartMeetingReminderWorker()

	r := gin.Default()

	r.Use(middleware.CORSMiddleware())

	// Публичная раздача загруженных аватаров (без cookie — <img> с того же origin всё равно ок)
	r.Static("/api/v1/uploads", "./uploads")

	v1 := r.Group("/api/v1")
	{
		v1.POST("/auth/signup", controllers.SignUp)
		v1.POST("/auth/signin", controllers.Login)
		v1.POST("/auth/logout", controllers.Logout)
		v1.GET("/auth/validate", middleware.RequireAuth, controllers.Validate)
		v1.PUT("/profile/avatar/preset", middleware.RequireAuth, controllers.SetAvatarPreset)
		v1.POST("/profile/avatar", middleware.RequireAuth, controllers.UploadAvatar)
		v1.PUT("/profile/name", middleware.RequireAuth, controllers.UpdateProfileName)

		v1.POST("/meetings", controllers.CreateMeeting)
		v1.GET("/meetings/reminders/status", controllers.MeetingReminderStatus)
		v1.POST("/meetings/reminders/subscribe", controllers.SubscribeMeetingReminder)
		v1.GET("/meetings/upcoming", controllers.GetAllUpcomingMeetings)
		v1.GET("/meetings/current", controllers.GetCurrentMeeting)
		v1.PUT("/meetings/:id", controllers.UpdateMeeting)
		v1.DELETE("/meetings/:id", controllers.DeleteMeeting)
		v1.GET("/meetings/:id", controllers.GetMeetingByCode)
		v1.POST("/rooms", controllers.CreateRoom)
		v1.GET("/rooms/:roomId/chat", middleware.OptionalAuth, controllers.ListRoomChat)
		v1.POST("/rooms/:roomId/chat", middleware.OptionalAuth, controllers.PostRoomChat)
		v1.PUT("/rooms/:roomId/chat/read", middleware.RequireAuth, controllers.PutRoomChatRead)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	r.Run(fmt.Sprintf(":%s", port))
}
