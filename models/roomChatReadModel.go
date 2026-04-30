package models

import "gorm.io/gorm"

// RoomChatRead — до какого сообщения пользователь дочитал чат в комнате (по ID RoomChatMessage).
type RoomChatRead struct {
	gorm.Model
	UserID            uint   `gorm:"uniqueIndex:ux_room_chat_read;not null" json:"user_id"`
	RoomID            string `gorm:"uniqueIndex:ux_room_chat_read;not null;size:128" json:"room_id"`
	LastReadMessageID uint   `gorm:"not null;default:0" json:"last_read_message_id"`
}
