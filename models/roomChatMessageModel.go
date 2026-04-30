package models

import "gorm.io/gorm"

// RoomChatMessage — сообщение чата комнаты (roomId = UUID из /rooms, как в URL встречи).
// UserID=0 — гость; тогда GuestPeerID = стабильный id клиента (localStorage) для peerId в API.
type RoomChatMessage struct {
	gorm.Model
	RoomID      string `gorm:"index;not null;size:128" json:"room_id"`
	UserID      uint   `gorm:"index;not null" json:"user_id"`
	GuestPeerID string `gorm:"size:64;index" json:"guest_peer_id"`
	DisplayName string `gorm:"size:128" json:"display_name"`
	Body        string `gorm:"type:text;not null" json:"body"`
}
