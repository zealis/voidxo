CREATE TABLE `vx_inbox` (
  `inbox_id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `sender_id` bigint unsigned NOT NULL,
  `receiver_id` bigint unsigned NOT NULL,
  `content_markdown` text NOT NULL COMMENT '原始私信内容',
  `content_rendered` mediumtext NOT NULL COMMENT '过滤渲染后的私信内容',
  `read_at` datetime(6) DEFAULT NULL COMMENT '阅读时间，NULL 表示未读',
  `created_at` datetime(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  PRIMARY KEY (`inbox_id`),
  KEY `idx_inbox_receiver` (`receiver_id`, `read_at`, `inbox_id`),
  KEY `idx_inbox_sender` (`sender_id`, `inbox_id`),
  KEY `idx_inbox_pair` (`sender_id`, `receiver_id`, `inbox_id`),
  CONSTRAINT `fk_inbox_sender` FOREIGN KEY (`sender_id`) REFERENCES `vx_user` (`user_id`) ON DELETE RESTRICT,
  CONSTRAINT `fk_inbox_receiver` FOREIGN KEY (`receiver_id`) REFERENCES `vx_user` (`user_id`) ON DELETE RESTRICT
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='私信';
