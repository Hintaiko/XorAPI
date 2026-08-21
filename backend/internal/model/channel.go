package model

import "time"

type Group struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	Name        string    `gorm:"size:64;uniqueIndex" json:"name"`
	Description string    `gorm:"size:255" json:"description"`
	Priority    int       `gorm:"default:0" json:"priority"`
	Status      string    `gorm:"size:16;default:active" json:"status"`
	Channels    []Channel `gorm:"foreignKey:GroupID" json:"channels,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type Channel struct {
	ID         uint       `gorm:"primaryKey" json:"id"`
	GroupID    uint       `gorm:"index" json:"group_id"`
	Name       string     `gorm:"size:64" json:"name"`
	BaseURL    string     `gorm:"size:255" json:"base_url"`
	APIKeyEnc  string     `gorm:"size:512" json:"-"`
	Protocol   string     `gorm:"size:16;default:openai" json:"protocol"`
	Priority   int        `gorm:"default:0" json:"priority"`
	Status     string     `gorm:"size:16;default:active" json:"status"`
	TestStatus string     `gorm:"size:16;default:unknown" json:"test_status"`
	LastTestAt *time.Time `json:"last_test_at"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}

type ModelInfo struct {
	ID           uint    `gorm:"primaryKey" json:"id"`
	Name         string  `gorm:"size:128;index:idx_model_name" json:"name"`
	GroupID      uint    `gorm:"index" json:"group_id"`
	DisplayName  string  `gorm:"size:128" json:"display_name"`
	Description  string  `gorm:"type:text" json:"description"`
	Tags         string  `gorm:"size:255" json:"tags"`
	BillingType  string  `gorm:"size:16;default:token" json:"billing_type"`
	InputPrice   float64 `gorm:"type:decimal(18,6);default:0" json:"input_price"`
	OutputPrice  float64 `gorm:"type:decimal(18,6);default:0" json:"output_price"`
	PerCallPrice float64 `gorm:"type:decimal(18,4);default:0" json:"per_call_price"`
	Visible      bool    `gorm:"default:true" json:"visible"`
	Callable    bool    `gorm:"default:true" json:"callable"`
}
