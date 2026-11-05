CREATE TABLE IF NOT EXISTS `vx_option` (
  `name` varchar(40) NOT NULL COMMENT '配置键',
  `value` text NOT NULL COMMENT '配置值',
  PRIMARY KEY (`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='站点配置';

INSERT INTO `vx_option` (`name`, `value`) VALUES
('site_name', 'VOIDXO'),
('site_description', ''),
('site_keywords', ''),
('site_icp_beian', ''),
('site_gongan_beian', ''),
('language', 'zh-CN'),
('question_can_edit', 'false'),
('question_can_edit_before', '0'),
('question_can_edit_only_no_answer', 'true'),
('question_can_edit_only_no_comment', 'true'),
('question_can_delete', 'false'),
('question_can_delete_before', '0'),
('question_can_delete_only_no_answer', 'true'),
('question_can_delete_only_no_comment', 'true'),
('answer_can_edit', 'false'),
('answer_can_edit_before', '0'),
('answer_can_edit_only_no_comment', 'false'),
('answer_can_delete', 'false'),
('answer_can_delete_before', '0'),
('answer_can_delete_only_no_comment', 'true'),
('article_can_edit', 'false'),
('article_can_edit_before', '0'),
('article_can_edit_only_no_comment', 'true'),
('article_can_delete', 'false'),
('article_can_delete_before', '0'),
('article_can_delete_only_no_comment', 'true'),
('comment_can_edit', 'false'),
('comment_can_edit_before', '0'),
('comment_can_delete', 'false'),
('comment_can_delete_before', '0')
ON DUPLICATE KEY UPDATE `name` = VALUES(`name`);

CREATE TABLE IF NOT EXISTS `vx_user_sessions` (
  `session_id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `user_id` bigint unsigned NOT NULL,
  `token_hash` char(64) NOT NULL COMMENT '会话令牌 SHA-256，不保存原始令牌',
  `device_id` varchar(128) NOT NULL DEFAULT '',
  `ip_address` varchar(45) DEFAULT NULL,
  `user_agent` varchar(600) NOT NULL DEFAULT '',
  `created_at` datetime(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  `last_active_at` datetime(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  `expires_at` datetime(6) NOT NULL,
  `revoked_at` datetime(6) DEFAULT NULL,
  `revoked_reason` varchar(32) DEFAULT NULL,
  PRIMARY KEY (`session_id`),
  UNIQUE KEY `uk_user_sessions_token_hash` (`token_hash`),
  KEY `idx_user_sessions_user_active` (`user_id`, `revoked_at`, `expires_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='长期会话与安全审计记录';
