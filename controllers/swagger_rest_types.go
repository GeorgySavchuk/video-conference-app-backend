package controllers

import "github.com/GeorgySavchuk/video-conference-app-backend/models"

// AuthSignupBody POST /auth/signup.
type AuthSignupBody struct {
	Name     string `json:"name" example:"alice"`
	Password string `json:"password"`
	Email    string `json:"email" example:"alice@example.com"`
}

// AuthLoginBody POST /auth/signin.
type AuthLoginBody struct {
	Name     string `json:"name"`
	Password string `json:"password"`
}

// AuthSessionSuccess ответ Login и Validate.
type AuthSessionSuccess struct {
	Message string            `json:"message"`
	User    models.UserPublic `json:"user"`
}

// EmptySuccess успешная регистрация — пустой JSON {}.
type EmptySuccess struct{}

// ProfileUserJSON ответ профиля с полем user.
type ProfileUserJSON struct {
	User models.UserPublic `json:"user"`
}

// AvatarPresetBody PUT /profile/avatar/preset.
type AvatarPresetBody struct {
	Preset string `json:"preset" example:"p3"`
}

// ProfileNameBody PUT /profile/name.
type ProfileNameBody struct {
	Name string `json:"name" binding:"required"`
}

// CreateRoomResponse POST /rooms.
type CreateRoomResponse struct {
	RoomID string `json:"roomId"`
}

// RoomChatMessageWire элемент сообщения в ответах чата.
type RoomChatMessageWire struct {
	ID          string `json:"id"`
	PeerID      string `json:"peerId"`
	DisplayName string `json:"displayName"`
	Text        string `json:"text"`
	Ts          int64  `json:"ts"`
	UserID      uint   `json:"userId"`
}

// RoomChatListResponse GET .../chat.
type RoomChatListResponse struct {
	Messages          []RoomChatMessageWire `json:"messages"`
	LastReadMessageID string                `json:"lastReadMessageId"`
}

// PostRoomChatBody POST .../chat (пользователь или гость).
type PostRoomChatBody struct {
	Text        string `json:"text"`
	DisplayName string `json:"displayName"`
	GuestPeerID string `json:"guestPeerId"`
}

// PostRoomChatCreated 201 — одно сообщение в поле message.
type PostRoomChatCreated struct {
	Message RoomChatMessageWire `json:"message"`
}

// PutRoomChatReadBody PUT .../chat/read.
type PutRoomChatReadBody struct {
	LastReadMessageID uint `json:"lastReadMessageId"`
}

// PutRoomChatReadSuccess ответ после обновления курсора.
type PutRoomChatReadSuccess struct {
	LastReadMessageID string `json:"lastReadMessageId"`
}
