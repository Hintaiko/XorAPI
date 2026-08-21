package service

import (
	"strconv"
	"sync"
	"time"

	"xorapi/internal/model"
	"xorapi/internal/store"

	"gorm.io/gorm"
)

var (
	cfgMu    sync.RWMutex
	cfgCache = map[string]string{}
	cfgAt    time.Time
)

func refreshConfigs() {
	if store.DB == nil {
		return
	}
	cfgMu.Lock()
	defer cfgMu.Unlock()
	if time.Since(cfgAt) < 10*time.Second {
		return
	}
	var rows []model.SystemConfig
	if err := store.DB.Find(&rows).Error; err != nil {
		return
	}
	m := map[string]string{}
	for _, r := range rows {
		m[r.Key] = r.Value
	}
	cfgCache = m
	cfgAt = time.Now()
}

func GetConfig(key string) string {
	refreshConfigs()
	cfgMu.RLock()
	defer cfgMu.RUnlock()
	return cfgCache[key]
}

func GetConfigBool(key string, def bool) bool {
	v := GetConfig(key)
	if v == "" {
		return def
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return def
	}
	return b
}

func GetConfigInt(key string, def int) int {
	v := GetConfig(key)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}

func SetConfigs(db *gorm.DB, kv map[string]string) error {
	for k, v := range kv {
		if err := db.Save(&model.SystemConfig{Key: k, Value: v}).Error; err != nil {
			return err
		}
	}
	cfgMu.Lock()
	cfgAt = time.Time{}
	cfgMu.Unlock()
	return nil
}
