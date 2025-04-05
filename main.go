package main

import (
	"github.com/GeorgySavchuk/video-conference-app-backend/controllers"
	"github.com/GeorgySavchuk/video-conference-app-backend/initializers"
	"github.com/GeorgySavchuk/video-conference-app-backend/middleware"
	"github.com/gin-gonic/gin"
)

func init() {
	initializers.LoadEnv()
	initializers.ConnectToDB()
	initializers.SyncDB()
}

func main() {
	r := gin.Default()

	// POST запросы
	r.POST("/signup", controllers.SignUp)
	r.POST("/login", controllers.Login)
	r.POST("/logout", controllers.Logout)
	r.POST("/meetings", controllers.CreateMeeting)
	r.POST("/meetings/:id", controllers.UpdateMeeting)
	// GET запросы
	r.GET("/validate", middleware.RequireAuth, controllers.Validate)
	r.GET("/upcoming_meetings", controllers.GetAllUpcomingMeetings)
	r.GET("/current_meeting", controllers.GetCurrentMeeting)
	// DELETE запросы
	r.DELETE("/meetings/:id")

	r.Run()
}
