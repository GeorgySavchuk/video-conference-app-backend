package middleware

import (
	"fmt"
	"os"
	"time"

	"github.com/GeorgySavchuk/video-conference-app-backend/initializers"
	"github.com/GeorgySavchuk/video-conference-app-backend/models"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// OptionalAuth — при валидной cookie выставляет user; иначе пропускает без 401.
func OptionalAuth(c *gin.Context) {
	tokenString, err := c.Cookie("Authorization")
	if err != nil {
		c.Next()
		return
	}

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(os.Getenv("JWT_SECRET")), nil
	})
	if err != nil || token == nil || !token.Valid {
		c.Next()
		return
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		c.Next()
		return
	}
	if exp, ok := claims["exp"].(float64); ok && float64(time.Now().Unix()) > exp {
		c.Next()
		return
	}

	var user models.User
	initializers.DB.First(&user, claims["sub"])
	if user.ID == 0 {
		c.Next()
		return
	}

	models.EnsureUserDefaultAvatar(initializers.DB, &user)
	c.Set("user", user)
	c.Next()
}
