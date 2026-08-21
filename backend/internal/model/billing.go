package model

import "time"

// PointGrant 点数批次：免费点数（签到等，可过期）/ 充值点数（管理员充值）
type PointGrant struct {
	ID        uint       `gorm:"primaryKey" json:"id"`
	UserID    uint       `gorm:"index" json:"user_id"`
	Type      string     `gorm:"size:16;index" json:"type"` // free | charged
	Amount    float64    `gorm:"type:decimal(18,4)" json:"amount"`
	Remaining float64    `gorm:"type:decimal(18,4)" json:"remaining"`
	ExpiresAt *time.Time `gorm:"index" json:"expires_at"`
	Note      string     `gorm:"size:255" json:"note"`
	CreatedAt time.Time  `json:"created_at"`
}

type Transaction struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	UserID        uint      `gorm:"index" json:"user_id"`
	Type          string    `gorm:"size:32;index" json:"type"` // recharge | consume | signin | adjust | refund
	Amount        float64   `gorm:"type:decimal(18,4)" json:"amount"` // 正为增加，负为扣除
	BalanceAfter  float64   `gorm:"type:decimal(18,4)" json:"balance_after"`
	Model         string    `gorm:"size:128" json:"model"`
	Detail        string    `gorm:"type:text" json:"detail"`
	CreatedAt     time.Time `gorm:"index" json:"created_at"`
}

type CallLog struct {
	ID               uint      `gorm:"primaryKey" json:"id"`
	UserID           uint      `gorm:"index" json:"user_id"`
	APIKeyID         uint      `gorm:"index" json:"api_key_id"`
	Model            string    `gorm:"size:128;index" json:"model"`
	GroupID          uint      `json:"group_id"`
	ChannelID        uint      `json:"channel_id"`
	Protocol         string    `gorm:"size:16" json:"protocol"`
	PromptTokens     int       `gorm:"default:0" json:"prompt_tokens"`
	CompletionTokens int       `gorm:"default:0" json:"completion_tokens"`
	Points           float64   `gorm:"type:decimal(18,4)" json:"points"`
	LatencyMs        int64     `json:"latency_ms"`
	Status           string    `gorm:"size:16;default:success;index" json:"status"`
	IP               string    `gorm:"size:64" json:"ip"`
	Error            string    `gorm:"size:255" json:"error"`
	CreatedAt        time.Time `gorm:"index" json:"created_at"`
}

type CheckInRecord struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"uniqueIndex:idx_user_date" json:"user_id"`
	Date      string    `gorm:"size:10;uniqueIndex:idx_user_date" json:"date"`
	Streak    int       `json:"streak"`
	Reward    float64   `gorm:"type:decimal(18,4)" json:"reward"`
	CreatedAt time.Time `json:"created_at"`
}

type SystemConfig struct {
	Key   string `gorm:"primaryKey;size:64" json:"key"`
	Value string `gorm:"type:text" json:"value"`
}
