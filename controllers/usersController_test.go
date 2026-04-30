package controllers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/GeorgySavchuk/video-conference-app-backend/initializers"
	"github.com/GeorgySavchuk/video-conference-app-backend/models"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/bcrypt"
)

func setupTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.Default()
	return router
}

func setupTestDB(t *testing.T) {
	// Настройка тестовой базы данных
	os.Setenv("DB_HOST", "localhost")
	os.Setenv("DB_USER", "postgres")
	os.Setenv("DB_PASSWORD", "postgres")
	os.Setenv("DB_NAME", "video_conference_test")
	os.Setenv("DB_PORT", "5432")
	os.Setenv("JWT_SECRET", "test_secret")

	initializers.ConnectToDB()
	initializers.DB.AutoMigrate(&models.User{})
}

func cleanupTestDB(t *testing.T) {
	initializers.DB.Migrator().DropTable(&models.User{})
}

func TestSignUp(t *testing.T) {
	setupTestDB(t)
	defer cleanupTestDB(t)

	router := setupTestRouter()
	router.POST("/signup", SignUp)

	tests := []struct {
		name           string
		payload        map[string]string
		expectedStatus int
		expectedError  bool
	}{
		{
			name: "Успешная регистрация",
			payload: map[string]string{
				"name":     "testuser",
				"password": "testpass",
				"email":    "testuser@example.com",
			},
			expectedStatus: http.StatusOK,
			expectedError:  false,
		},
		{
			name: "Регистрация с занятым email",
			payload: map[string]string{
				"name":     "otheruser",
				"password": "testpass",
				"email":    "testuser@example.com",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  true,
		},
		{
			name: "Регистрация с существующим именем",
			payload: map[string]string{
				"name":     "testuser",
				"password": "testpass",
				"email":    "other@example.com",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  true,
		},
		{
			name: "Регистрация без имени",
			payload: map[string]string{
				"password": "testpass",
				"email":    "x@example.com",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  true,
		},
		{
			name: "Регистрация без email",
			payload: map[string]string{
				"name":     "another",
				"password": "testpass",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jsonData, _ := json.Marshal(tt.payload)
			req, _ := http.NewRequest("POST", "/signup", bytes.NewBuffer(jsonData))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedError {
				var response map[string]string
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Contains(t, response, "error")
			}
		})
	}
}

func TestLogin(t *testing.T) {
	setupTestDB(t)
	defer cleanupTestDB(t)

	router := setupTestRouter()
	router.POST("/login", Login)

	// Создаем тестового пользователя
	hash, _ := bcrypt.GenerateFromPassword([]byte("testpass"), 10)
	user := models.User{Name: "testuser", Password: string(hash)}
	initializers.DB.Create(&user)

	tests := []struct {
		name           string
		payload        map[string]string
		expectedStatus int
		expectedError  bool
	}{
		{
			name: "Успешная авторизация",
			payload: map[string]string{
				"name":     "testuser",
				"password": "testpass",
			},
			expectedStatus: http.StatusOK,
			expectedError:  false,
		},
		{
			name: "Неверный пароль",
			payload: map[string]string{
				"name":     "testuser",
				"password": "wrongpass",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  true,
		},
		{
			name: "Несуществующий пользователь",
			payload: map[string]string{
				"name":     "nonexistent",
				"password": "testpass",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jsonData, _ := json.Marshal(tt.payload)
			req, _ := http.NewRequest("POST", "/login", bytes.NewBuffer(jsonData))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedError {
				var response map[string]string
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Contains(t, response, "error")
			} else {
				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Contains(t, response, "message")
				assert.Contains(t, response, "user")
			}
		})
	}
}
