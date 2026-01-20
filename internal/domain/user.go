package domain

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type User struct {
	ID                primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	ThreadsUserID     string             `bson:"threads_user_id" json:"threads_user_id"`
	Username          string             `bson:"username" json:"username"`
	DisplayName       string             `bson:"display_name" json:"display_name"`
	ProfilePictureURL string             `bson:"profile_picture_url" json:"profile_picture_url"`
	AccessToken       string             `bson:"access_token" json:"-"`
	RefreshToken      string             `bson:"refresh_token" json:"-"`
	TokenExpiresAt    time.Time          `bson:"token_expires_at" json:"-"`
	AutoReplyEnabled  bool               `bson:"auto_reply_enabled" json:"auto_reply_enabled"`
	DefaultTemplateID *primitive.ObjectID `bson:"default_template_id,omitempty" json:"default_template_id,omitempty"`
	Settings          UserSettings       `bson:"settings" json:"settings"`
	CreatedAt         time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt         time.Time          `bson:"updated_at" json:"updated_at"`
}

type UserSettings struct {
	ReplyDelaySeconds      int      `bson:"reply_delay_seconds" json:"reply_delay_seconds"`
	MaxRepliesPerHour      int      `bson:"max_replies_per_hour" json:"max_replies_per_hour"`
	IgnoreVerifiedAccounts bool     `bson:"ignore_verified_accounts" json:"ignore_verified_accounts"`
	IgnoreKeywords         []string `bson:"ignore_keywords" json:"ignore_keywords"`
}

func NewUser(threadsUserID, username, displayName, profilePictureURL string) *User {
	now := time.Now()
	return &User{
		ThreadsUserID:     threadsUserID,
		Username:          username,
		DisplayName:       displayName,
		ProfilePictureURL: profilePictureURL,
		AutoReplyEnabled:  true,
		Settings: UserSettings{
			ReplyDelaySeconds:      30,
			MaxRepliesPerHour:      50,
			IgnoreVerifiedAccounts: false,
			IgnoreKeywords:         []string{},
		},
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func (u *User) SetTokens(accessToken, refreshToken string, expiresAt time.Time) {
	u.AccessToken = accessToken
	u.RefreshToken = refreshToken
	u.TokenExpiresAt = expiresAt
	u.UpdatedAt = time.Now()
}

func (u *User) IsTokenExpired() bool {
	return time.Now().After(u.TokenExpiresAt)
}

func (u *User) TokenExpiresIn() time.Duration {
	return time.Until(u.TokenExpiresAt)
}
