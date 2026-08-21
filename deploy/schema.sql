-- XorAPI MySQL 初始化脚本（参考用）
-- 说明：正常情况下无需手动执行，安装向导会通过 GORM AutoMigrate 自动建表。
-- 此脚本供高级用户预建表或审计表结构时使用，兼容 MySQL 5.6.4+ / 5.7+ / 8.0+（及 MariaDB 10.x），utf8mb4 编码。
-- 兼容性说明：所有 utf8mb4 索引列长度均 ≤191 字符（764 字节），适配 MySQL 5.6 默认 767 字节的索引前缀限制；
-- 未使用 JSON 类型、CHECK 约束、函数索引、生成列等 5.7+/8.0+ 专属特性；DATETIME(3) 自 5.6.4 起支持。

CREATE DATABASE IF NOT EXISTS `xorapi` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
USE `xorapi`;

-- 用户表
CREATE TABLE IF NOT EXISTS `users` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `email` VARCHAR(191) NOT NULL,
  `username` VARCHAR(64) DEFAULT NULL,
  `password_hash` VARCHAR(255) DEFAULT NULL,
  `role` VARCHAR(16) DEFAULT 'user',
  `status` VARCHAR(16) DEFAULT 'active',
  `email_verified` TINYINT(1) DEFAULT 0,
  `points` DECIMAL(18,4) DEFAULT 0,
  `created_at` DATETIME(3) DEFAULT NULL,
  `updated_at` DATETIME(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_users_email` (`email`),
  KEY `idx_users_role` (`role`),
  KEY `idx_users_status` (`status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- API Key 表（仅存 SHA-256 哈希）
CREATE TABLE IF NOT EXISTS `api_keys` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `user_id` BIGINT UNSIGNED NOT NULL,
  `name` VARCHAR(64) DEFAULT NULL,
  `key_hash` VARCHAR(64) NOT NULL,
  `key_preview` VARCHAR(32) DEFAULT NULL,
  `ip_whitelist` TEXT,
  `daily_limit` BIGINT DEFAULT 0,
  `status` VARCHAR(16) DEFAULT 'active',
  `times` BIGINT DEFAULT 0,
  `last_used_at` DATETIME(3) DEFAULT NULL,
  `created_at` DATETIME(3) DEFAULT NULL,
  `updated_at` DATETIME(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_api_keys_key_hash` (`key_hash`),
  KEY `idx_api_keys_user_id` (`user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 邀请码表
CREATE TABLE IF NOT EXISTS `invite_codes` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `code` VARCHAR(32) NOT NULL,
  `created_by` BIGINT UNSIGNED NOT NULL,
  `used_by` BIGINT UNSIGNED DEFAULT NULL,
  `status` VARCHAR(16) DEFAULT 'unused',
  `created_at` DATETIME(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_invite_codes_code` (`code`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 邮箱验证码表
CREATE TABLE IF NOT EXISTS `email_codes` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `email` VARCHAR(191) NOT NULL,
  `code` VARCHAR(8) DEFAULT NULL,
  `expires_at` DATETIME(3) DEFAULT NULL,
  `used_at` DATETIME(3) DEFAULT NULL,
  `created_at` DATETIME(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_email_codes_email` (`email`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 模型分组表
CREATE TABLE IF NOT EXISTS `groups` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `name` VARCHAR(64) NOT NULL,
  `description` VARCHAR(255) DEFAULT NULL,
  `priority` BIGINT DEFAULT 0,
  `status` VARCHAR(16) DEFAULT 'active',
  `created_at` DATETIME(3) DEFAULT NULL,
  `updated_at` DATETIME(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_groups_name` (`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 上游渠道表（API Key 为 AES-256-GCM 加密存储）
CREATE TABLE IF NOT EXISTS `channels` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `group_id` BIGINT UNSIGNED NOT NULL,
  `name` VARCHAR(64) DEFAULT NULL,
  `base_url` VARCHAR(255) DEFAULT NULL,
  `api_key_enc` VARCHAR(512) DEFAULT NULL,
  `protocol` VARCHAR(16) DEFAULT 'openai',
  `priority` BIGINT DEFAULT 0,
  `status` VARCHAR(16) DEFAULT 'active',
  `test_status` VARCHAR(16) DEFAULT 'unknown',
  `last_test_at` DATETIME(3) DEFAULT NULL,
  `created_at` DATETIME(3) DEFAULT NULL,
  `updated_at` DATETIME(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_channels_group_id` (`group_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 模型表（同名模型可存在于多个分组，实现跨组 fallback）
CREATE TABLE IF NOT EXISTS `model_infos` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `name` VARCHAR(128) NOT NULL,
  `group_id` BIGINT UNSIGNED NOT NULL,
  `display_name` VARCHAR(128) DEFAULT NULL,
  `description` TEXT,
  `tags` VARCHAR(255) DEFAULT NULL,
  `billing_type` VARCHAR(16) DEFAULT 'token',
  `input_price` DECIMAL(18,6) DEFAULT 0,
  `output_price` DECIMAL(18,6) DEFAULT 0,
  `per_call_price` DECIMAL(18,4) DEFAULT 0,
  `visible` TINYINT(1) DEFAULT 1,
  `callable` TINYINT(1) DEFAULT 1,
  PRIMARY KEY (`id`),
  KEY `idx_model_name` (`name`),
  KEY `idx_model_infos_group_id` (`group_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 点数批次表（free=签到等免费点数可过期，charged=充值点数）
CREATE TABLE IF NOT EXISTS `point_grants` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `user_id` BIGINT UNSIGNED NOT NULL,
  `type` VARCHAR(16) NOT NULL,
  `amount` DECIMAL(18,4) NOT NULL,
  `remaining` DECIMAL(18,4) NOT NULL,
  `expires_at` DATETIME(3) DEFAULT NULL,
  `note` VARCHAR(255) DEFAULT NULL,
  `created_at` DATETIME(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_point_grants_user_id` (`user_id`),
  KEY `idx_point_grants_type` (`type`),
  KEY `idx_point_grants_expires_at` (`expires_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 交易流水表
CREATE TABLE IF NOT EXISTS `transactions` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `user_id` BIGINT UNSIGNED NOT NULL,
  `type` VARCHAR(32) NOT NULL,
  `amount` DECIMAL(18,4) NOT NULL,
  `balance_after` DECIMAL(18,4) NOT NULL,
  `model` VARCHAR(128) DEFAULT NULL,
  `detail` TEXT,
  `created_at` DATETIME(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_transactions_user_id` (`user_id`),
  KEY `idx_transactions_type` (`type`),
  KEY `idx_transactions_created_at` (`created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 调用日志表
CREATE TABLE IF NOT EXISTS `call_logs` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `user_id` BIGINT UNSIGNED NOT NULL,
  `api_key_id` BIGINT UNSIGNED NOT NULL,
  `model` VARCHAR(128) DEFAULT NULL,
  `group_id` BIGINT UNSIGNED NOT NULL,
  `channel_id` BIGINT UNSIGNED NOT NULL,
  `protocol` VARCHAR(16) DEFAULT NULL,
  `prompt_tokens` BIGINT DEFAULT 0,
  `completion_tokens` BIGINT DEFAULT 0,
  `points` DECIMAL(18,4) DEFAULT 0,
  `latency_ms` BIGINT DEFAULT 0,
  `status` VARCHAR(16) DEFAULT 'success',
  `ip` VARCHAR(64) DEFAULT NULL,
  `error` VARCHAR(255) DEFAULT NULL,
  `created_at` DATETIME(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_call_logs_user_id` (`user_id`),
  KEY `idx_call_logs_api_key_id` (`api_key_id`),
  KEY `idx_call_logs_model` (`model`),
  KEY `idx_call_logs_status` (`status`),
  KEY `idx_call_logs_created_at` (`created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 签到记录表
CREATE TABLE IF NOT EXISTS `check_in_records` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `user_id` BIGINT UNSIGNED NOT NULL,
  `date` VARCHAR(10) NOT NULL,
  `streak` BIGINT DEFAULT 1,
  `reward` DECIMAL(18,4) DEFAULT 0,
  `created_at` DATETIME(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_user_date` (`user_id`, `date`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 系统配置表
CREATE TABLE IF NOT EXISTS `system_configs` (
  `key` VARCHAR(64) NOT NULL,
  `value` TEXT,
  PRIMARY KEY (`key`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 默认系统配置
INSERT INTO `system_configs` (`key`, `value`) VALUES
  ('registration_enabled', 'true'),
  ('invite_required', 'false'),
  ('email_verify', 'false'),
  ('checkin_enabled', 'true'),
  ('checkin_base', '10'),
  ('checkin_streak_bonus', '2'),
  ('checkin_max_reward', '30'),
  ('checkin_expire_days', '30'),
  ('exchange_note', '1 元 ≈ 100 点（仅展示用，实际以管理员充值为准）'),
  ('smtp_host', ''),
  ('smtp_port', '465'),
  ('smtp_user', ''),
  ('smtp_pass', ''),
  ('smtp_from', ''),
  ('relay_rpm', '60'),
  ('active_template', 'default'),
  ('site_announcement', '')
ON DUPLICATE KEY UPDATE `value` = VALUES(`value`);
