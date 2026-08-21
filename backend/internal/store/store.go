package store

import (
	"context"
	"fmt"
	"log"
	"time"

	"xorapi/internal/config"

	"github.com/redis/go-redis/v9"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var (
	Cfg   *config.Config
	DB    *gorm.DB
	Redis *redis.Client
)

func InitDB(cfg *config.Config) error {
	gormCfg := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	}
	db, err := gorm.Open(mysql.Open(cfg.DSN()), gormCfg)
	if err != nil {
		return fmt.Errorf("连接数据库失败: %w", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	sqlDB.SetMaxIdleConns(5)
	sqlDB.SetMaxOpenConns(50)
	sqlDB.SetConnMaxLifetime(time.Hour)
	DB = db
	return nil
}

func InitRedis(cfg *config.Config) *redis.Client {
	if cfg.Redis.Host == "" {
		return nil
	}
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr(),
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Printf("[warn] Redis 连接失败（%v），限流与计数将使用内存模式", err)
		_ = rdb.Close()
		return nil
	}
	Redis = rdb
	return rdb
}

// IncrDaily 每日计数（Redis 优先，失败回退内存），key 形如 kd:3:20260821
func IncrDaily(prefix string, id uint, now time.Time) int64 {
	day := now.Format("20060102")
	key := fmt.Sprintf("xorapi:%s:%d:%s", prefix, id, day)
	if Redis != nil {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		n, err := Redis.Incr(ctx, key).Result()
		if err == nil {
			if n == 1 {
				Redis.Expire(ctx, key, 48*time.Hour)
			}
			return n
		}
	}
	return memIncr(key)
}

// AllowMinute 简单固定窗口限流：每 key 每分钟 limit 次
func AllowMinute(key string, limit int64) bool {
	if limit <= 0 {
		return true
	}
	if Redis != nil {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		window := time.Now().Unix() / 60
		rkey := fmt.Sprintf("xorapi:rl:%s:%d", key, window)
		n, err := Redis.Incr(ctx, rkey).Result()
		if err == nil {
			if n == 1 {
				Redis.Expire(ctx, rkey, 90*time.Second)
			}
			return n <= limit
		}
	}
	return memAllow(key, limit)
}
