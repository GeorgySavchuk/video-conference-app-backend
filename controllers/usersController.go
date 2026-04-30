package controllers

import (
	"net/http"
	"net/mail"
	"os"
	"strings"
	"time"

	"github.com/GeorgySavchuk/video-conference-app-backend/initializers"
	"github.com/GeorgySavchuk/video-conference-app-backend/models"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

func SignUp(c *gin.Context) {
	var body struct {
		Name     string `json:"name"`
		Password string `json:"password"`
		Email    string `json:"email"`
	}

	if c.Bind(&body) != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Не удалось спарсить тело запроса",
		})

		return
	}

	name := strings.TrimSpace(body.Name)
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Укажите логин"})
		return
	}

	email := strings.TrimSpace(strings.ToLower(body.Email))
	if email == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Укажите email"})
		return
	}
	if _, err := mail.ParseAddress(email); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Некорректный email"})
		return
	}

	var dupEmail models.User
	if err := initializers.DB.Where("email = ?", email).First(&dupEmail).Error; err == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Этот email уже зарегистрирован"})
		return
	}

	var dupName models.User
	if err := initializers.DB.Where("name = ?", name).First(&dupName).Error; err == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Это имя пользователя уже занято"})
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(body.Password), 10)

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Не удалось захешировать пароль",
		})

		return
	}

	// Avatar задаётся в models.User.BeforeCreate (случайный p0…p7).
	user := models.User{Name: name, Password: string(hash), Email: email}
	result := initializers.DB.Create(&user)

	if result.Error != nil {
		// На случай гонки или записи вне проверки выше — дубликат логина в БД.
		var again models.User
		if err := initializers.DB.Where("name = ?", name).First(&again).Error; err == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Это имя пользователя уже занято"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Не удалось создать пользователя",
		})

		return
	}

	c.JSON(http.StatusOK, gin.H{})
}

func Login(c *gin.Context) {
	var body struct {
		Name     string `json:"name"`
		Password string `json:"password"`
	}

	if c.Bind(&body) != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Не удалось спарсить тело запроса",
		})

		return
	}

	var user models.User
	initializers.DB.First(&user, "name = ?", body.Name)

	if user.ID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Неправильный логин или пароль",
		})

		return
	}

	err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(body.Password))

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Неправильный логин или пароль",
		})

		return
	}

	models.EnsureUserDefaultAvatar(initializers.DB, &user)

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": user.ID,
		"exp": time.Now().Add(time.Hour * 24 * 30).Unix(),
	})

	tokenString, err := token.SignedString([]byte(os.Getenv("JWT_SECRET")))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Не удалось создать токен",
		})
		return
	}

	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie("Authorization", tokenString, 3600*24*30, "", "", false, true)

	c.JSON(http.StatusOK, gin.H{
		"message": "Авторизован",
		"user":    user.Public(),
	})
}

func Logout(c *gin.Context) {
	c.SetCookie("Authorization", "", -1, "", "", false, true)

	c.JSON(http.StatusOK, gin.H{
		"message": "Успешный выход из системы",
	})
}

func Validate(c *gin.Context) {
	raw, ok := c.Get("user")
	if !ok {
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}
	u := raw.(models.User)

	c.JSON(http.StatusOK, gin.H{
		"message": "Авторизован",
		"user":    u.Public(),
	})
}
