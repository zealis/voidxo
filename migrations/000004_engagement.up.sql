CREATE TABLE `vx_article` (
  `article_id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `user_id` bigint unsigned NOT NULL,
  `title` varchar(80) NOT NULL,
  `content_markdown` mediumtext NOT NULL,
  `content_rendered` mediumtext NOT NULL,
  `comment_count` int unsigned NOT NULL DEFAULT 0,
  `follower_count` int unsigned NOT NULL DEFAULT 0,
  `vote_count` int NOT NULL DEFAULT 0,
  `vote_up_count` int unsigned NOT NULL DEFAULT 0,
  `vote_down_count` int unsigned NOT NULL DEFAULT 0,
  `created_at` datetime(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  `updated_at` datetime(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
  `deleted_at` datetime(6) DEFAULT NULL,
  PRIMARY KEY (`article_id`),
  KEY `idx_article_list` (`deleted_at`, `created_at`, `article_id`),
  KEY `idx_article_user` (`user_id`, `deleted_at`),
  CONSTRAINT `fk_article_user` FOREIGN KEY (`user_id`) REFERENCES `vx_user` (`user_id`) ON DELETE RESTRICT
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='文章';

CREATE TABLE `vx_comment` (
  `comment_id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `commentable_id` bigint unsigned NOT NULL,
  `commentable_type` varchar(16) NOT NULL COMMENT 'article、question、answer、comment',
  `root_id` bigint unsigned DEFAULT NULL COMMENT '顶层评论编号，用于折叠回复',
  `user_id` bigint unsigned NOT NULL,
  `content` varchar(1000) NOT NULL COMMENT '纯文本评论，渲染时转义',
  `reply_count` int unsigned NOT NULL DEFAULT 0,
  `vote_count` int NOT NULL DEFAULT 0,
  `vote_up_count` int unsigned NOT NULL DEFAULT 0,
  `vote_down_count` int unsigned NOT NULL DEFAULT 0,
  `created_at` datetime(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  `updated_at` datetime(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
  `deleted_at` datetime(6) DEFAULT NULL,
  PRIMARY KEY (`comment_id`),
  KEY `idx_comment_target` (`commentable_type`, `commentable_id`, `deleted_at`, `created_at`, `comment_id`),
  KEY `idx_comment_root` (`root_id`, `deleted_at`, `comment_id`),
  KEY `idx_comment_user` (`user_id`, `deleted_at`),
  CONSTRAINT `fk_comment_user` FOREIGN KEY (`user_id`) REFERENCES `vx_user` (`user_id`) ON DELETE RESTRICT
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='评论';

CREATE TABLE `vx_vote` (
  `user_id` bigint unsigned NOT NULL,
  `votable_id` bigint unsigned NOT NULL,
  `votable_type` varchar(16) NOT NULL COMMENT 'article、question、answer、comment',
  `type` varchar(8) NOT NULL COMMENT 'up、down',
  `created_at` datetime(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  PRIMARY KEY (`user_id`, `votable_type`, `votable_id`),
  KEY `idx_vote_target` (`votable_type`, `votable_id`, `type`),
  CONSTRAINT `fk_vote_user` FOREIGN KEY (`user_id`) REFERENCES `vx_user` (`user_id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='投票';

CREATE TABLE `vx_follow` (
  `user_id` bigint unsigned NOT NULL,
  `followable_id` bigint unsigned NOT NULL,
  `followable_type` varchar(16) NOT NULL COMMENT 'user、question、article、topic',
  `created_at` datetime(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  PRIMARY KEY (`user_id`, `followable_type`, `followable_id`),
  KEY `idx_follow_target` (`followable_type`, `followable_id`),
  CONSTRAINT `fk_follow_user` FOREIGN KEY (`user_id`) REFERENCES `vx_user` (`user_id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='关注';

CREATE TABLE `vx_report` (
  `report_id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `reportable_id` bigint unsigned NOT NULL,
  `reportable_type` varchar(16) NOT NULL COMMENT 'question、article、answer、comment、user',
  `user_id` bigint unsigned NOT NULL,
  `reason` varchar(200) NOT NULL,
  `status` varchar(16) NOT NULL DEFAULT 'pending' COMMENT 'pending、resolved、dismissed',
  `handled_by` bigint unsigned DEFAULT NULL,
  `handled_at` datetime(6) DEFAULT NULL,
  `created_at` datetime(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  PRIMARY KEY (`report_id`),
  UNIQUE KEY `uk_report_once` (`reportable_type`, `reportable_id`, `user_id`),
  KEY `idx_report_status` (`status`, `created_at`, `report_id`),
  CONSTRAINT `fk_report_user` FOREIGN KEY (`user_id`) REFERENCES `vx_user` (`user_id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='举报';

CREATE TABLE `vx_notification` (
  `notification_id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `receiver_id` bigint unsigned NOT NULL,
  `sender_id` bigint unsigned NOT NULL DEFAULT 0,
  `type` varchar(40) NOT NULL COMMENT 'question_answered、question_commented、question_deleted、article_commented、article_deleted、answer_commented、answer_deleted、comment_replied、comment_deleted',
  `article_id` bigint unsigned NOT NULL DEFAULT 0,
  `question_id` bigint unsigned NOT NULL DEFAULT 0,
  `answer_id` bigint unsigned NOT NULL DEFAULT 0,
  `comment_id` bigint unsigned NOT NULL DEFAULT 0,
  `reply_id` bigint unsigned NOT NULL DEFAULT 0,
  `content_deleted` mediumtext,
  `created_at` datetime(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  `read_at` datetime(6) DEFAULT NULL,
  PRIMARY KEY (`notification_id`),
  KEY `idx_notification_receiver` (`receiver_id`, `read_at`, `notification_id`),
  CONSTRAINT `fk_notification_receiver` FOREIGN KEY (`receiver_id`) REFERENCES `vx_user` (`user_id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='通知';

ALTER TABLE `vx_user`
  ADD COLUMN `comment_count` int unsigned NOT NULL DEFAULT 0 AFTER `answer_count`;
