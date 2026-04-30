package models

import "time"

// UserPublic — пользователь в ответах API (без пароля и служебных полей GORM).
type UserPublic struct {
	ID        uint      `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Avatar    string    `json:"avatar"`
	CreatedAt time.Time `json:"created_at"`
}

// Public сериализует модель для Login, Validate и /profile/* .
func (u User) Public() UserPublic {
	return UserPublic{
		ID:        u.ID,
		Name:      u.Name,
		Email:     u.Email,
		Avatar:    u.Avatar,
		CreatedAt: u.CreatedAt,
	}
}
