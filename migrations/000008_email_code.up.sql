CREATE TABLE `vx_email_code` (
  `code_id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `email` varchar(320) NOT NULL,
  `purpose` varchar(16) NOT NULL COMMENT 'register、reset',
  `code_hash` char(64) NOT NULL COMMENT '验证码哈希，不保存明文',
  `attempts` tinyint unsigned NOT NULL DEFAULT 0 COMMENT '校验失败次数',
  `expires_at` datetime(6) NOT NULL,
  `consumed_at` datetime(6) DEFAULT NULL,
  `created_at` datetime(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  PRIMARY KEY (`code_id`),
  KEY `idx_email_code_lookup` (`email`, `purpose`, `created_at`),
  KEY `idx_email_code_expiry` (`expires_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='邮箱验证码';
