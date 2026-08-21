package database

import (
	"xorapi/internal/model"

	"gorm.io/gorm"
)

func Migrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&model.User{},
		&model.APIKey{},
		&model.InviteCode{},
		&model.EmailCode{},
		&model.Group{},
		&model.Channel{},
		&model.ModelInfo{},
		&model.PointGrant{},
		&model.Transaction{},
		&model.CallLog{},
		&model.CheckInRecord{},
		&model.SystemConfig{},
	)
}

var defaultConfigs = map[string]string{
	"registration_enabled":  "true",
	"invite_required":       "false",
	"email_verify":          "false",
	"checkin_enabled":       "true",
	"checkin_base":          "10",
	"checkin_streak_bonus":  "2",
	"checkin_max_reward":    "30",
	"checkin_expire_days":   "30",
	"exchange_note":         "1 元 ≈ 100 点（仅展示用，实际以管理员充值为准）",
	"smtp_host":             "",
	"smtp_port":             "465",
	"smtp_user":             "",
	"smtp_pass":             "",
	"smtp_from":             "",
	"relay_rpm":             "60",
	"active_template":       "default",
	"site_announcement":     "",
}

func SeedConfigs(db *gorm.DB) error {
	for k, v := range defaultConfigs {
		var count int64
		db.Model(&model.SystemConfig{}).Where("key = ?", k).Count(&count)
		if count == 0 {
			if err := db.Create(&model.SystemConfig{Key: k, Value: v}).Error; err != nil {
				return err
			}
		}
	}
	return nil
}
