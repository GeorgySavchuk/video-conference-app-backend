package main

import (
	"fmt"
	"os"

	"github.com/GeorgySavchuk/video-conference-app-backend/controllers"
	"github.com/GeorgySavchuk/video-conference-app-backend/initializers"
	"github.com/GeorgySavchuk/video-conference-app-backend/middleware"
	"github.com/gin-gonic/gin"
)

func init() {
	initializers.ConnectToDB()
	initializers.SyncDB()
}

func main() {
	r := gin.Default()

	r.Use(middleware.CORSMiddleware())

	v1 := r.Group("/api/v1")
	{
		v1.POST("/auth/signup", controllers.SignUp)
		v1.POST("/auth/signin", controllers.Login)
		v1.POST("/auth/logout", controllers.Logout)
		v1.GET("/auth/validate", middleware.RequireAuth, controllers.Validate)

		v1.POST("/meetings", controllers.CreateMeeting)
		v1.GET("/meetings/upcoming", controllers.GetAllUpcomingMeetings)
		v1.GET("/meetings/current", controllers.GetCurrentMeeting)
		v1.PUT("/meetings/:id", controllers.UpdateMeeting)
		v1.DELETE("/meetings/:id", controllers.DeleteMeeting)
		v1.GET("/meetings/:id", controllers.GetMeetingByCode)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	r.Run(fmt.Sprintf(":%s", port))
}
