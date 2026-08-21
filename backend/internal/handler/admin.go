package handler

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"xorapi/internal/model"
	"xorapi/internal/service"
	"xorapi/internal/store"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

// ---------- 系统配置 ----------

func AdminGetConfigs(c *gin.Context) {
	var rows []model.SystemConfig
	store.DB.Find(&rows)
	m := map[string]string{}
	for _, r := range rows {
		m[r.Key] = r.Value
	}
	OK(c, m)
}

func AdminUpdateConfigs(c *gin.Context) {
	var req map[string]string
	if !BindJSON(c, &req) {
		return
	}
	if err := service.SetConfigs(store.DB, req); err != nil {
		Fail(c, http.StatusInternalServerError, "保存失败: "+err.Error())
		return
	}
	OK(c, gin.H{"updated": true})
}

// ---------- 用户管理 ----------

func AdminListUsers(c *gin.Context) {
	page, size := pageNo(c)
	q := store.DB.Model(&model.User{})
	if kw := c.Query("search"); kw != "" {
		q = q.Where("email LIKE ? OR username LIKE ?", "%"+kw+"%", "%"+kw+"%")
	}
	var total int64
	q.Count(&total)
	var users []model.User
	q.Order("id DESC").Offset((page - 1) * size).Limit(size).Find(&users)
	OK(c, gin.H{"total": total, "list": users})
}

func AdminUpdateUser(c *gin.Context) {
	id := c.Param("id")
	var user model.User
	if err := store.DB.First(&user, id).Error; err != nil {
		Fail(c, http.StatusNotFound, "用户不存在")
		return
	}
	var req struct {
		Status   *string `json:"status"`
		Role     *string `json:"role"`
		Password string  `json:"password"`
	}
	if !BindJSON(c, &req) {
		return
	}
	updates := map[string]any{}
	if req.Status != nil && (*req.Status == "active" || *req.Status == "disabled") {
		updates["status"] = *req.Status
	}
	if req.Role != nil && (*req.Role == "admin" || *req.Role == "user") {
		updates["role"] = *req.Role
	}
	if len(req.Password) >= 6 {
		hash, _ := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		updates["password_hash"] = string(hash)
	}
	if len(updates) > 0 {
		store.DB.Model(&user).Updates(updates)
	}
	OK(c, gin.H{"updated": true})
}

func AdminAdjustPoints(c *gin.Context) {
	id := c.Param("id")
	var user model.User
	if err := store.DB.First(&user, id).Error; err != nil {
		Fail(c, http.StatusNotFound, "用户不存在")
		return
	}
	var req struct {
		Delta float64 `json:"delta"`
		Note  string  `json:"note"`
	}
	if !BindJSON(c, &req) {
		return
	}
	if req.Delta == 0 {
		Fail(c, http.StatusBadRequest, "调整数额不能为 0")
		return
	}
	if req.Note == "" {
		req.Note = "管理员调整"
	}
	if err := service.AdjustPoints(store.DB, user.ID, req.Delta, req.Note); err != nil {
		Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	var fresh model.User
	store.DB.First(&fresh, user.ID)
	OK(c, gin.H{"points": fresh.Points})
}

// ---------- 分组与渠道 ----------

func AdminListGroups(c *gin.Context) {
	var groups []model.Group
	store.DB.Preload("Channels").Order("priority ASC, id ASC").Find(&groups)
	OK(c, groups)
}

type groupPayload struct {
	ID          uint             `json:"id"`
	Name        string           `json:"name" binding:"required"`
	Description string           `json:"description"`
	Priority    int              `json:"priority"`
	Status      string           `json:"status"`
	Channels    []channelPayload `json:"channels"`
}

type channelPayload struct {
	ID       uint   `json:"id"`
	Name     string `json:"name" binding:"required"`
	BaseURL  string `json:"base_url" binding:"required"`
	APIKey   string `json:"api_key"`
	Protocol string `json:"protocol"`
	Priority int    `json:"priority"`
	Status   string `json:"status"`
}

func AdminSaveGroup(c *gin.Context) {
	var req groupPayload
	if !BindJSON(c, &req) {
		return
	}
	group := model.Group{
		Name: req.Name, Description: req.Description, Priority: req.Priority,
	}
	if req.Status == "" {
		req.Status = "active"
	}
	group.Status = req.Status
	if req.ID > 0 {
		group.ID = req.ID
		if err := store.DB.Model(&model.Group{}).Where("id = ?", req.ID).Updates(map[string]any{
			"name": group.Name, "description": group.Description, "priority": group.Priority, "status": group.Status,
		}).Error; err != nil {
			Fail(c, http.StatusInternalServerError, "保存失败: "+err.Error())
			return
		}
	} else {
		if err := store.DB.Create(&group).Error; err != nil {
			Fail(c, http.StatusBadRequest, "创建失败（名称可能重复）: "+err.Error())
			return
		}
	}
	var existIDs []uint
	store.DB.Model(&model.Channel{}).Where("group_id = ?", group.ID).Pluck("id", &existIDs)
	keep := map[uint]bool{}
	for _, ch := range req.Channels {
		if ch.Protocol == "" {
			ch.Protocol = "openai"
		}
		if ch.Status == "" {
			ch.Status = "active"
		}
		if ch.ID > 0 {
			keep[ch.ID] = true
			updates := map[string]any{
				"name": ch.Name, "base_url": ch.BaseURL, "protocol": ch.Protocol,
				"priority": ch.Priority, "status": ch.Status,
			}
			if ch.APIKey != "" {
				enc, err := service.Encrypt(store.Cfg.Server.EncryptKey, ch.APIKey)
				if err != nil {
					Fail(c, http.StatusInternalServerError, "渠道 Key 加密失败")
					return
				}
				updates["api_key_enc"] = enc
			}
			store.DB.Model(&model.Channel{}).Where("id = ?", ch.ID).Updates(updates)
		} else {
			if ch.APIKey == "" {
				Fail(c, http.StatusBadRequest, "新渠道必须填写 API Key")
				return
			}
			enc, err := service.Encrypt(store.Cfg.Server.EncryptKey, ch.APIKey)
			if err != nil {
				Fail(c, http.StatusInternalServerError, "渠道 Key 加密失败")
				return
			}
			store.DB.Create(&model.Channel{
				GroupID: group.ID, Name: ch.Name, BaseURL: ch.BaseURL,
				APIKeyEnc: enc, Protocol: ch.Protocol, Priority: ch.Priority, Status: ch.Status,
			})
		}
	}
	for _, id := range existIDs {
		if !keep[id] {
			store.DB.Delete(&model.Channel{}, id)
		}
	}
	OK(c, gin.H{"id": group.ID})
}

func AdminDeleteGroup(c *gin.Context) {
	id := c.Param("id")
	var models []model.ModelInfo
	store.DB.Where("group_id = ?", id).Find(&models)
	if len(models) > 0 {
		Fail(c, http.StatusBadRequest, "该分组下仍有模型，请先删除或迁移模型")
		return
	}
	store.DB.Delete(&model.Channel{}, "group_id = ?", id)
	store.DB.Delete(&model.Group{}, id)
	OK(c, gin.H{"deleted": true})
}

// AdminTestChannel 测试渠道连通性（列模型接口）
func AdminTestChannel(c *gin.Context) {
	id := c.Param("id")
	var ch model.Channel
	if err := store.DB.First(&ch, id).Error; err != nil {
		Fail(c, http.StatusNotFound, "渠道不存在")
		return
	}
	key, _ := service.Decrypt(store.Cfg.Server.EncryptKey, ch.APIKeyEnc)
	url := strings.TrimRight(ch.BaseURL, "/")
	if ch.Protocol == "anthropic" {
		url += "/v1/models"
	} else {
		url += "/models"
	}
	req, _ := http.NewRequest(http.MethodGet, url, nil)
	if ch.Protocol == "anthropic" {
		req.Header.Set("x-api-key", key)
		req.Header.Set("anthropic-version", "2023-06-01")
	} else {
		req.Header.Set("Authorization", "Bearer "+key)
	}
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	status, msg := "fail", ""
	if err != nil {
		msg = err.Error()
	} else {
		defer resp.Body.Close()
		if resp.StatusCode == 200 {
			status = "ok"
			msg = "连通正常"
		} else {
			msg = fmt.Sprintf("HTTP %d", resp.StatusCode)
		}
	}
	now := time.Now()
	store.DB.Model(&ch).Updates(map[string]any{"test_status": status, "last_test_at": &now})
	OK(c, gin.H{"test_status": status, "msg": msg})
}

// ---------- 模型管理 ----------

func AdminListModels(c *gin.Context) {
	var models []model.ModelInfo
	q := store.DB.Order("id ASC")
	if kw := c.Query("search"); kw != "" {
		q = q.Where("name LIKE ? OR display_name LIKE ?", "%"+kw+"%", "%"+kw+"%")
	}
	q.Find(&models)
	var groups []model.Group
	store.DB.Find(&groups)
	groupName := map[uint]string{}
	for _, g := range groups {
		groupName[g.ID] = g.Name
	}
	type item struct {
		model.ModelInfo
		GroupName string `json:"group_name"`
	}
	list := make([]item, 0, len(models))
	for _, m := range models {
		list = append(list, item{ModelInfo: m, GroupName: groupName[m.GroupID]})
	}
	OK(c, gin.H{"list": list, "groups": groups})
}

func AdminSaveModel(c *gin.Context) {
	var m model.ModelInfo
	if !BindJSON(c, &m) {
		return
	}
	if m.Name == "" || m.GroupID == 0 {
		Fail(c, http.StatusBadRequest, "模型名称与所属分组必填")
		return
	}
	if m.BillingType == "" {
		m.BillingType = "token"
	}
	if m.ID > 0 {
		if err := store.DB.Model(&model.ModelInfo{}).Where("id = ?", m.ID).Updates(map[string]any{
			"name": m.Name, "group_id": m.GroupID, "display_name": m.DisplayName, "description": m.Description,
			"tags": m.Tags, "billing_type": m.BillingType, "input_price": m.InputPrice,
			"output_price": m.OutputPrice, "per_call_price": m.PerCallPrice,
			"visible": m.Visible, "callable": m.Callable,
		}).Error; err != nil {
			Fail(c, http.StatusInternalServerError, "保存失败: "+err.Error())
			return
		}
	} else {
		if err := store.DB.Create(&m).Error; err != nil {
			Fail(c, http.StatusInternalServerError, "创建失败: "+err.Error())
			return
		}
	}
	OK(c, gin.H{"id": m.ID})
}

func AdminDeleteModel(c *gin.Context) {
	id := c.Param("id")
	res := store.DB.Delete(&model.ModelInfo{}, id)
	if res.RowsAffected == 0 {
		Fail(c, http.StatusNotFound, "模型不存在")
		return
	}
	OK(c, gin.H{"deleted": true})
}

// ---------- 邀请码 ----------

func AdminListInvites(c *gin.Context) {
	var list []model.InviteCode
	store.DB.Order("id DESC").Limit(200).Find(&list)
	OK(c, list)
}

func AdminCreateInvites(c *gin.Context) {
	var req struct {
		Count int `json:"count"`
	}
	if !BindJSON(c, &req) {
		return
	}
	if req.Count < 1 || req.Count > 100 {
		req.Count = 10
	}
	codes := make([]string, 0, req.Count)
	for i := 0; i < req.Count; i++ {
		code, _ := service.RandomToken(8)
		ic := model.InviteCode{Code: code, Status: "unused"}
		if err := store.DB.Create(&ic).Error; err == nil {
			codes = append(codes, code)
		}
	}
	OK(c, gin.H{"codes": codes})
}

func AdminDeleteInvite(c *gin.Context) {
	id := c.Param("id")
	store.DB.Delete(&model.InviteCode{}, id)
	OK(c, gin.H{"deleted": true})
}

// ---------- 数据看板 ----------

func AdminDashboard(c *gin.Context) {
	var totalUsers, totalCalls int64
	var totalPoints float64
	store.DB.Model(&model.User{}).Count(&totalUsers)
	store.DB.Model(&model.CallLog{}).Count(&totalCalls)
	store.DB.Model(&model.CallLog{}).Select("COALESCE(SUM(points),0)").Scan(&totalPoints)

	type modelRank struct {
		Model  string  `json:"model"`
		Calls  int64   `json:"calls"`
		Points float64 `json:"points"`
	}
	var ranks []modelRank
	store.DB.Model(&model.CallLog{}).
		Select("model, COUNT(*) as calls, COALESCE(SUM(points),0) as points").
		Group("model").Order("calls DESC").Limit(10).Scan(&ranks)

	type dayStat struct {
		Date  string `json:"date"`
		Calls int64  `json:"calls"`
	}
	var days []dayStat
	store.DB.Model(&model.CallLog{}).
		Select("DATE(created_at) as date, COUNT(*) as calls").
		Where("created_at > ?", time.Now().AddDate(0, 0, -7)).
		Group("DATE(created_at)").Order("date ASC").Scan(&days)

	OK(c, gin.H{
		"total_users": totalUsers, "total_calls": totalCalls, "total_points": totalPoints,
		"model_ranks": ranks, "recent_days": days,
	})
}
