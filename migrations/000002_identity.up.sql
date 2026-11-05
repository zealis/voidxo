CREATE TABLE `vx_role` (
  `role_id` smallint unsigned NOT NULL AUTO_INCREMENT,
  `code` varchar(32) NOT NULL COMMENT '稳定的角色代码',
  `name` varchar(50) NOT NULL,
  `description` varchar(255) NOT NULL DEFAULT '',
  `is_system` tinyint(1) NOT NULL DEFAULT 0,
  `created_at` datetime(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  PRIMARY KEY (`role_id`),
  UNIQUE KEY `uk_role_code` (`code`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='角色';

CREATE TABLE `vx_permission` (
  `permission_id` smallint unsigned NOT NULL AUTO_INCREMENT,
  `code` varchar(64) NOT NULL COMMENT '稳定的权限代码',
  `name` varchar(80) NOT NULL,
  `created_at` datetime(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  PRIMARY KEY (`permission_id`),
  UNIQUE KEY `uk_permission_code` (`code`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='权限';

CREATE TABLE `vx_role_permission` (
  `role_id` smallint unsigned NOT NULL,
  `permission_id` smallint unsigned NOT NULL,
  PRIMARY KEY (`role_id`, `permission_id`),
  KEY `idx_role_permission_permission` (`permission_id`),
  CONSTRAINT `fk_role_permission_role` FOREIGN KEY (`role_id`) REFERENCES `vx_role` (`role_id`) ON DELETE CASCADE,
  CONSTRAINT `fk_role_permission_permission` FOREIGN KEY (`permission_id`) REFERENCES `vx_permission` (`permission_id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='角色权限关系';

CREATE TABLE `vx_user` (
  `user_id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `username` varchar(20) NOT NULL,
  `email` varchar(320) NOT NULL,
  `password_hash` varchar(255) NOT NULL COMMENT 'Argon2id PHC 字符串',
  `status` tinyint unsigned NOT NULL DEFAULT 1 COMMENT '1 正常，0 禁用',
  `avatar_key` varchar(50) DEFAULT NULL,
  `cover_key` varchar(50) DEFAULT NULL,
  `headline` varchar(40) NOT NULL DEFAULT '',
  `bio` varchar(160) NOT NULL DEFAULT '',
  `blog` varchar(255) NOT NULL DEFAULT '',
  `company` varchar(255) NOT NULL DEFAULT '',
  `location` varchar(255) NOT NULL DEFAULT '',
  `create_ip` varchar(45) DEFAULT NULL,
  `create_location` varchar(100) NOT NULL DEFAULT '',
  `last_login_at` datetime(6) DEFAULT NULL,
  `last_login_ip` varchar(45) DEFAULT NULL,
  `last_login_location` varchar(100) NOT NULL DEFAULT '',
  `email_verified_at` datetime(6) DEFAULT NULL,
  `password_changed_at` datetime(6) DEFAULT NULL,
  `follower_count` int unsigned NOT NULL DEFAULT 0,
  `followee_count` int unsigned NOT NULL DEFAULT 0,
  `following_article_count` int unsigned NOT NULL DEFAULT 0,
  `following_question_count` int unsigned NOT NULL DEFAULT 0,
  `following_topic_count` int unsigned NOT NULL DEFAULT 0,
  `article_count` int unsigned NOT NULL DEFAULT 0,
  `question_count` int unsigned NOT NULL DEFAULT 0,
  `answer_count` int unsigned NOT NULL DEFAULT 0,
  `notification_unread` int unsigned NOT NULL DEFAULT 0,
  `inbox_unread` int unsigned NOT NULL DEFAULT 0,
  `created_at` datetime(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  `updated_at` datetime(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
  `disabled_at` datetime(6) DEFAULT NULL,
  PRIMARY KEY (`user_id`),
  UNIQUE KEY `uk_user_username` (`username`),
  UNIQUE KEY `uk_user_email` (`email`),
  KEY `idx_user_status_created` (`status`, `created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='用户';

CREATE TABLE `vx_user_role` (
  `user_id` bigint unsigned NOT NULL,
  `role_id` smallint unsigned NOT NULL,
  `assigned_at` datetime(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  PRIMARY KEY (`user_id`, `role_id`),
  KEY `idx_user_role_role` (`role_id`),
  CONSTRAINT `fk_user_role_user` FOREIGN KEY (`user_id`) REFERENCES `vx_user` (`user_id`) ON DELETE CASCADE,
  CONSTRAINT `fk_user_role_role` FOREIGN KEY (`role_id`) REFERENCES `vx_role` (`role_id`) ON DELETE RESTRICT
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='用户角色关系';

ALTER TABLE `vx_user_sessions`
  MODIFY `user_id` bigint unsigned NOT NULL,
  ADD CONSTRAINT `fk_user_sessions_user`
  FOREIGN KEY (`user_id`) REFERENCES `vx_user` (`user_id`) ON DELETE CASCADE;

INSERT INTO `vx_role` (`code`, `name`, `description`, `is_system`) VALUES
('admin', '管理员', '拥有全部后台管理权限', 1),
('moderator', '版主', '拥有内容审核与举报处理权限', 1),
('user', '用户', '普通注册用户', 1);

INSERT INTO `vx_permission` (`code`, `name`) VALUES
('admin.access', '访问管理后台'),
('user.manage', '管理用户'),
('content.manage', '管理内容'),
('content.moderate', '审核内容'),
('report.manage', '处理举报'),
('option.manage', '修改站点设置'),
('file.manage', '管理文件'),
('notification.manage', '管理通知');

INSERT INTO `vx_role_permission` (`role_id`, `permission_id`)
SELECT r.`role_id`, p.`permission_id`
FROM `vx_role` r
CROSS JOIN `vx_permission` p
WHERE r.`code` = 'admin';

INSERT INTO `vx_role_permission` (`role_id`, `permission_id`)
SELECT r.`role_id`, p.`permission_id`
FROM `vx_role` r
JOIN `vx_permission` p ON p.`code` IN ('content.moderate', 'report.manage')
WHERE r.`code` = 'moderator';
