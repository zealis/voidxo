CREATE TABLE `vx_outbox` (
  `outbox_id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `event_type` varchar(64) NOT NULL COMMENT '事件类型，如 content.question.created',
  `event_key` varchar(120) NOT NULL COMMENT '生产者幂等键，重复投递时被唯一键拦截',
  `payload` json NOT NULL COMMENT '事件负载，供消费者解析',
  `status` varchar(16) NOT NULL DEFAULT 'pending' COMMENT 'pending、processing、done、failed',
  `attempts` int unsigned NOT NULL DEFAULT 0,
  `available_at` datetime(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  `last_error` varchar(500) NOT NULL DEFAULT '',
  `created_at` datetime(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  `processed_at` datetime(6) DEFAULT NULL,
  PRIMARY KEY (`outbox_id`),
  UNIQUE KEY `uk_outbox_event_key` (`event_key`),
  KEY `idx_outbox_pending` (`status`, `available_at`, `outbox_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='事务性发件箱';
