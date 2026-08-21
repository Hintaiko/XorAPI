package model

import "time"

type User struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	Email         string    `gorm:"size:191;uniqueIndex" json:"email"`
	Username      string    `gorm:"size:64" json:"username"`
	PasswordHash  string    `gorm:"size:255" json:"-"`
	Role          string    `gorm:"size:16;default:user;index" json:"role"`
	Status        string    `gorm:"size:16;default:active;index" json:"status"`
	EmailVerified bool      `gorm:"default:false" json:"email_verified"`
	Points        float64   `gorm:"type:decimal(18,4);default:0" json:"points"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type APIKey struct {
	ID         uint       `gorm:"primaryKey" json:"id"`
	UserID     uint       `gorm:"index" json:"user_id"`
	Name       string     `gorm:"size:64" json:"name"`
	KeyHash    string     `gorm:"size:64;uniqueIndex" json:"-"`
	KeyPreview string     `gorm:"size:32" json:"key_preview"`
	IPWhitelist string    `gorm:"type:text" json:"ip_whitelist"`
	DailyLimit int64      `gorm:"default:0" json:"daily_limit"`
	Status     string     `gorm:"size:16;default:active" json:"status"`
	Times      int64      `gorm:"default:0" json:"times"`
	LastUsedAt *time.Time `json:"last_used_at"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}

type InviteCode struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Code      string    `gorm:"size:32;uniqueIndex" json:"code"`
	CreatedBy uint      `json:"created_by"`
	UsedBy    *uint     `json:"used_by"`
	Status    string    `gorm:"size:16;default:unused" json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

type EmailCode struct {
	ID        uint       `gorm:"primaryKey" json:"id"`
	Email     string     `gorm:"size:191;index" json:"email"`
	Code      string     `gorm:"size:8" json:"-"`
	ExpiresAt time.Time  `json:"expires_at"`
	UsedAt    *time.Time `json:"used_at"`
	CreatedAt time.Time  `json:"created_at"`
}
