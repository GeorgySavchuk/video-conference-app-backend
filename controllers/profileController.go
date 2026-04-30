package controllers

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/GeorgySavchuk/video-conference-app-backend/initializers"
	"github.com/GeorgySavchuk/video-conference-app-backend/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

var allowedAvatarPresets = map[string]bool{
	"p0": true, "p1": true, "p2": true, "p3": true, "p4": true, "p5": true, "p6": true, "p7": true,
}

func deleteOldAvatarFile(avatar string) {
	if avatar == "" || !strings.HasPrefix(avatar, "avatars/") || strings.Contains(avatar, "..") {
		return
	}
	_ = os.Remove(filepath.Join("uploads", avatar))
}

// SetAvatarPreset PUT /profile/avatar/preset — только пресет, старый файл удаляется.
func SetAvatarPreset(c *gin.Context) {
	u, ok := c.Get("user")
	if !ok {
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}
	user := u.(models.User)

	var body struct {
		Preset string `json:"preset" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверное тело запроса"})
		return
	}
	pid := strings.TrimSpace(body.Preset)
	if !allowedAvatarPresets[pid] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неизвестный пресет"})
		return
	}

	deleteOldAvatarFile(user.Avatar)
	user.Avatar = "preset:" + pid
	if err := initializers.DB.Save(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Не удалось сохранить аватар"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"user": user.Public()})
}

// UploadAvatar POST /profile/avatar — multipart field "file".
func UploadAvatar(c *gin.Context) {
	u, ok := c.Get("user")
	if !ok {
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}
	user := u.(models.User)

	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Файл не получен"})
		return
	}
	if file.Size > 2*1024*1024 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Файл слишком большой (макс. 2 МБ)"})
		return
	}
	ct := strings.TrimSpace(strings.ToLower(file.Header.Get("Content-Type")))
	// Браузеры иногда шлют application/octet-stream или пустой MIME — добираем по расширению.
	if ct == "" || ct == "application/octet-stream" {
		switch strings.ToLower(filepath.Ext(file.Filename)) {
		case ".jpg", ".jpeg":
			ct = "image/jpeg"
		case ".png":
			ct = "image/png"
		case ".webp":
			ct = "image/webp"
		}
	}
	if !strings.HasPrefix(ct, "image/jpeg") && !strings.HasPrefix(ct, "image/jpg") &&
		!strings.HasPrefix(ct, "image/pjpeg") && !strings.HasPrefix(ct, "image/png") &&
		!strings.HasPrefix(ct, "image/webp") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Допустимы только JPEG, PNG или WebP"})
		return
	}
	ext := ".jpg"
	switch {
	case strings.Contains(ct, "png"):
		ext = ".png"
	case strings.Contains(ct, "webp"):
		ext = ".webp"
	}

	dir := filepath.Join("uploads", "avatars")
	if err := os.MkdirAll(dir, 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Не удалось создать каталог для файлов"})
		return
	}
	fn := fmt.Sprintf("%d_%s%s", user.ID, uuid.New().String(), ext)
	dst := filepath.Join(dir, fn)
	if err := c.SaveUploadedFile(file, dst); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Не удалось сохранить файл"})
		return
	}

	deleteOldAvatarFile(user.Avatar)
	rel := "avatars/" + fn
	user.Avatar = rel
	if err := initializers.DB.Save(&user).Error; err != nil {
		_ = os.Remove(dst)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Не удалось сохранить аватар"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"user": user.Public()})
}

// UpdateProfileName PUT /profile/name — смена отображаемого имени (уникальное в системе).
func UpdateProfileName(c *gin.Context) {
	u, ok := c.Get("user")
	if !ok {
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}
	user := u.(models.User)

	var body struct {
		Name string `json:"name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверное тело запроса"})
		return
	}
	name := strings.TrimSpace(body.Name)
	if utf8.RuneCountInString(name) < 2 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Имя слишком короткое (минимум 2 символа)"})
		return
	}
	if utf8.RuneCountInString(name) > 40 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Имя слишком длинное (максимум 40 символов)"})
		return
	}
	if user.Name == name {
		c.JSON(http.StatusOK, gin.H{"user": user.Public()})
		return
	}

	var taken int64
	if err := initializers.DB.Model(&models.User{}).
		Where("name = ? AND id <> ?", name, user.ID).
		Count(&taken).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Не удалось проверить имя"})
		return
	}
	if taken > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Это имя уже занято"})
		return
	}

	if err := initializers.DB.Model(&user).Update("name", name).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Не удалось обновить имя"})
		return
	}
	user.Name = name
	c.JSON(http.StatusOK, gin.H{"user": user.Public()})
}
