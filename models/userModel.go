package models

import (
	"fmt"
	"math/rand/v2"
	"strings"

	"gorm.io/gorm"
)

type User struct {
	gorm.Model
	Name     string `gorm:"unique" json:"name"`
	Password string `json:"-"`
	/** Нормализованный lowercase; пусто у старых записей до миграции. */
	Email string `gorm:"size:320;index" json:"email"`
	/** preset:p0…p7 или относительный путь avatars/… для загруженного файла */
	Avatar string `gorm:"size:768" json:"avatar"`
}

// RandomDefaultAvatarPreset — случайный пресет p0…p7 (как в profileController).
func RandomDefaultAvatarPreset() string {
	return fmt.Sprintf("preset:p%d", rand.IntN(8))
}

// BeforeCreate выставляет пресет, если Avatar не задан (регистрация и любые Create).
func (u *User) BeforeCreate(tx *gorm.DB) error {
	if strings.TrimSpace(u.Avatar) == "" {
		u.Avatar = RandomDefaultAvatarPreset()
	}
	return nil
}

// EnsureUserDefaultAvatar — для записей без пресета (старые пользователи до миграции логики).
func EnsureUserDefaultAvatar(db *gorm.DB, u *User) {
	if u == nil || u.ID == 0 || strings.TrimSpace(u.Avatar) != "" {
		return
	}
	u.Avatar = RandomDefaultAvatarPreset()
	_ = db.Model(u).Update("avatar", u.Avatar)
}
