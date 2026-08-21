package service

import (
	"encoding/json"
	"fmt"
	"math"
	"time"

	"xorapi/internal/model"
	"xorapi/internal/store"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func Round4(f float64) float64 {
	return math.Round(f*10000) / 10000
}

// ComputeCost 计算一次调用的点数消耗
func ComputeCost(m *model.ModelInfo, promptTokens, completionTokens int) float64 {
	if m.BillingType == "per_call" {
		return Round4(m.PerCallPrice)
	}
	cost := float64(promptTokens)/1_000_000*m.InputPrice + float64(completionTokens)/1_000_000*m.OutputPrice
	return Round4(cost)
}

// GrantPoints 增加点数（type: free/charged）
func GrantPoints(db *gorm.DB, userID uint, typ string, amount float64, expiresAt *time.Time, note string) error {
	if amount <= 0 {
		return fmt.Errorf("点数必须大于 0")
	}
	return db.Transaction(func(tx *gorm.DB) error {
		grant := model.PointGrant{UserID: userID, Type: typ, Amount: amount, Remaining: amount, ExpiresAt: expiresAt, Note: note}
		if err := tx.Create(&grant).Error; err != nil {
			return err
		}
		var user model.User
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&user, userID).Error; err != nil {
			return err
		}
		balance := Round4(user.Points + amount)
		if err := tx.Model(&user).Update("points", balance).Error; err != nil {
			return err
		}
		return tx.Create(&model.Transaction{
			UserID: userID, Type: txnType(typ, note), Amount: amount, BalanceAfter: balance, Detail: note,
		}).Error
	})
}

func txnType(typ, note string) string {
	switch {
	case typ == "free" && note == "每日签到":
		return "signin"
	case typ == "charged":
		return "recharge"
	default:
		return "adjust"
	}
}

// ConsumePoints 扣除点数：先扣即将过期的免费点数，再扣充值点数
func ConsumePoints(db *gorm.DB, userID uint, amount float64, m string, detail map[string]any) error {
	if amount < 0 {
		return fmt.Errorf("非法扣费")
	}
	return db.Transaction(func(tx *gorm.DB) error {
		var user model.User
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&user, userID).Error; err != nil {
			return err
		}
		if user.Points+1e-9 < amount {
			return fmt.Errorf("点数不足（当前余额 %.4f，需要 %.4f）", user.Points, amount)
		}
		var grants []model.PointGrant
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("user_id = ? AND remaining > 0 AND (expires_at IS NULL OR expires_at > NOW())", userID).
			Order("CASE WHEN type = 'free' THEN 0 ELSE 1 END, expires_at IS NULL ASC, expires_at ASC").
			Find(&grants).Error; err != nil {
			return err
		}
		need := amount
		for i := range grants {
			if need <= 1e-9 {
				break
			}
			take := grants[i].Remaining
			if take > need {
				take = need
			}
			if err := tx.Model(&model.PointGrant{}).Where("id = ?", grants[i].ID).
				Update("remaining", Round4(grants[i].Remaining-take)).Error; err != nil {
				return err
			}
			grants[i].Remaining -= take
			need = Round4(need - take)
		}
		if need > 1e-9 {
			return fmt.Errorf("点数扣除异常，剩余未扣 %.4f", need)
		}
		balance := Round4(user.Points - amount)
		if err := tx.Model(&user).Update("points", balance).Error; err != nil {
			return err
		}
		detailJSON, _ := json.Marshal(detail)
		return tx.Create(&model.Transaction{
			UserID: userID, Type: "consume", Amount: -amount, BalanceAfter: balance, Model: m, Detail: string(detailJSON),
		}).Error
	})
}

// AdjustPoints 管理员调整（正充负扣）
func AdjustPoints(db *gorm.DB, userID uint, delta float64, note string) error {
	if delta > 0 {
		return GrantPoints(db, userID, "charged", delta, nil, note)
	}
	return db.Transaction(func(tx *gorm.DB) error {
		var user model.User
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&user, userID).Error; err != nil {
			return err
		}
		if user.Points+1e-9 < -delta {
			return fmt.Errorf("用户余额不足（%.4f），无法扣减 %.4f", user.Points, -delta)
		}
		var grants []model.PointGrant
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("user_id = ? AND remaining > 0", userID).
			Order("expires_at IS NULL ASC, expires_at ASC").
			Find(&grants).Error; err != nil {
			return err
		}
		need := -delta
		for i := range grants {
			if need <= 1e-9 {
				break
			}
			take := grants[i].Remaining
			if take > need {
				take = need
			}
			if err := tx.Model(&model.PointGrant{}).Where("id = ?", grants[i].ID).
				Update("remaining", Round4(grants[i].Remaining-take)).Error; err != nil {
				return err
			}
			grants[i].Remaining -= take
			need = Round4(need - take)
		}
		balance := Round4(user.Points + delta)
		if err := tx.Model(&user).Update("points", balance).Error; err != nil {
			return err
		}
		return tx.Create(&model.Transaction{
			UserID: userID, Type: "adjust", Amount: delta, BalanceAfter: balance, Detail: note,
		}).Error
	})
}

// CheckIn 每日签到
func CheckIn(db *gorm.DB, userID uint) (*model.CheckInRecord, error) {
	now := time.Now()
	today := now.Format("2006-01-02")
	var rec model.CheckInRecord
	err := db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("user_id = ? AND date = ?", userID, today).First(&rec).Error; err == nil {
			return fmt.Errorf("今天已签到")
		}
		var last model.CheckInRecord
		streak := 1
		if err := tx.Where("user_id = ?", userID).Order("date DESC").First(&last).Error; err == nil {
			if last.Date == now.AddDate(0, 0, -1).Format("2006-01-02") {
				streak = last.Streak + 1
			}
		}
		base := float64(GetConfigInt("checkin_base", 10))
		bonus := float64(GetConfigInt("checkin_streak_bonus", 2))
		maxReward := float64(GetConfigInt("checkin_max_reward", 30))
		reward := Round4(base + float64(streak-1)*bonus)
		if reward > maxReward {
			reward = maxReward
		}
		expireDays := GetConfigInt("checkin_expire_days", 30)
		expiresAt := now.AddDate(0, 0, expireDays)
		rec = model.CheckInRecord{UserID: userID, Date: today, Streak: streak, Reward: reward}
		if err := tx.Create(&rec).Error; err != nil {
			return err
		}
		grant := model.PointGrant{
			UserID: userID, Type: "free", Amount: reward, Remaining: reward,
			ExpiresAt: &expiresAt, Note: "每日签到",
		}
		if err := tx.Create(&grant).Error; err != nil {
			return err
		}
		var user model.User
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&user, userID).Error; err != nil {
			return err
		}
		balance := Round4(user.Points + reward)
		if err := tx.Model(&user).Update("points", balance).Error; err != nil {
			return err
		}
		return tx.Create(&model.Transaction{
			UserID: userID, Type: "signin", Amount: reward, BalanceAfter: balance, Detail: "连续签到 " + fmt.Sprint(streak) + " 天",
		}).Error
	})
	if err != nil {
		return nil, err
	}
	return &rec, nil
}

// LogCall 记录调用日志并扣费
func LogCall(userID, apiKeyID, groupID, channelID uint, m, protocol, ip string,
	promptTokens, completionTokens int, cost float64, latencyMs int64, status, errMsg string, mi *model.ModelInfo) {

	db := store.DB
	log := model.CallLog{
		UserID: userID, APIKeyID: apiKeyID, Model: m, GroupID: groupID, ChannelID: channelID,
		Protocol: protocol, PromptTokens: promptTokens, CompletionTokens: completionTokens,
		Points: cost, LatencyMs: latencyMs, Status: status, IP: ip, Error: errMsg,
	}
	db.Create(&log)
	if status == "success" && cost > 0 {
		_ = ConsumePoints(db, userID, cost, m, map[string]any{
			"prompt_tokens": promptTokens, "completion_tokens": completionTokens,
			"billing_type": mi.BillingType,
		})
	}
}
