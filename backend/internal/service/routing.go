package service

import (
	"xorapi/internal/model"
	"xorapi/internal/store"
)

// ChannelPlan 一次调用候选：渠道 + 所属模型信息
type ChannelPlan struct {
	Channel model.Channel
	Model   model.ModelInfo
	GroupID uint
}

// ResolveChannels 解析模型候选渠道：模型所属分组优先，同名模型跨分组 fallback
func ResolveChannels(modelName string) ([]ChannelPlan, bool) {
	var models []model.ModelInfo
	if err := store.DB.Where("name = ? AND callable = ?", modelName, true).Find(&models).Error; err != nil || len(models) == 0 {
		return nil, false
	}
	groupIDs := make([]uint, 0, len(models))
	groupOf := map[uint]model.ModelInfo{}
	for _, m := range models {
		groupIDs = append(groupIDs, m.GroupID)
		groupOf[m.GroupID] = m
	}
	var groups []model.Group
	if err := store.DB.Where("id IN ? AND status = ?", groupIDs, "active").Order("priority ASC, id ASC").Find(&groups).Error; err != nil {
		return nil, false
	}
	plans := make([]ChannelPlan, 0, 8)
	for _, g := range groups {
		var channels []model.Channel
		if err := store.DB.Where("group_id = ? AND status = ?", g.ID, "active").Order("priority ASC, id ASC").Find(&channels).Error; err != nil {
			continue
		}
		for _, ch := range channels {
			plans = append(plans, ChannelPlan{Channel: ch, Model: groupOf[g.ID], GroupID: g.ID})
		}
	}
	return plans, len(plans) > 0
}
