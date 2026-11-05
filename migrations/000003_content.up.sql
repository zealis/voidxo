CREATE TABLE `vx_topic` (
  `topic_id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(20) NOT NULL,
  `description` varchar(1000) NOT NULL DEFAULT '',
  `cover_key` varchar(50) DEFAULT NULL,
  `article_count` int unsigned NOT NULL DEFAULT 0,
  `question_count` int unsigned NOT NULL DEFAULT 0,
  `follower_count` int unsigned NOT NULL DEFAULT 0,
  `created_at` datetime(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  `updated_at` datetime(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
  `deleted_at` datetime(6) DEFAULT NULL,
  PRIMARY KEY (`topic_id`),
  UNIQUE KEY `uk_topic_name` (`name`),
  KEY `idx_topic_deleted_follower` (`deleted_at`, `follower_count`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='话题';

CREATE TABLE `vx_topicable` (
  `topic_id` bigint unsigned NOT NULL,
  `topicable_id` bigint unsigned NOT NULL,
  `topicable_type` varchar(16) NOT NULL,
  `created_at` datetime(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  PRIMARY KEY (`topic_id`, `topicable_id`, `topicable_type`),
  KEY `idx_topicable_target` (`topicable_type`, `topicable_id`),
  CONSTRAINT `fk_topicable_topic` FOREIGN KEY (`topic_id`) REFERENCES `vx_topic` (`topic_id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='话题关系';

CREATE TABLE `vx_question` (
  `question_id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `user_id` bigint unsigned NOT NULL,
  `title` varchar(80) NOT NULL,
  `content_markdown` mediumtext NOT NULL,
  `content_rendered` mediumtext NOT NULL,
  `comment_count` int unsigned NOT NULL DEFAULT 0,
  `answer_count` int unsigned NOT NULL DEFAULT 0,
  `follower_count` int unsigned NOT NULL DEFAULT 0,
  `vote_count` int NOT NULL DEFAULT 0,
  `vote_up_count` int unsigned NOT NULL DEFAULT 0,
  `vote_down_count` int unsigned NOT NULL DEFAULT 0,
  `last_answer_at` datetime(6) DEFAULT NULL,
  `created_at` datetime(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  `updated_at` datetime(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
  `deleted_at` datetime(6) DEFAULT NULL,
  PRIMARY KEY (`question_id`),
  KEY `idx_question_list` (`deleted_at`, `updated_at`, `question_id`),
  KEY `idx_question_user` (`user_id`, `deleted_at`),
  KEY `idx_question_last_answer` (`deleted_at`, `last_answer_at`),
  CONSTRAINT `fk_question_user` FOREIGN KEY (`user_id`) REFERENCES `vx_user` (`user_id`) ON DELETE RESTRICT
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='提问';

CREATE TABLE `vx_answer` (
  `answer_id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `question_id` bigint unsigned NOT NULL,
  `user_id` bigint unsigned NOT NULL,
  `content_markdown` mediumtext NOT NULL,
  `content_rendered` mediumtext NOT NULL,
  `comment_count` int unsigned NOT NULL DEFAULT 0,
  `vote_count` int NOT NULL DEFAULT 0,
  `vote_up_count` int unsigned NOT NULL DEFAULT 0,
  `vote_down_count` int NOT NULL DEFAULT 0,
  `created_at` datetime(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  `updated_at` datetime(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
  `deleted_at` datetime(6) DEFAULT NULL,
  PRIMARY KEY (`answer_id`),
  KEY `idx_answer_question` (`question_id`, `deleted_at`, `created_at`, `answer_id`),
  KEY `idx_answer_user` (`user_id`, `deleted_at`),
  CONSTRAINT `fk_answer_question` FOREIGN KEY (`question_id`) REFERENCES `vx_question` (`question_id`) ON DELETE CASCADE,
  CONSTRAINT `fk_answer_user` FOREIGN KEY (`user_id`) REFERENCES `vx_user` (`user_id`) ON DELETE RESTRICT
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='回答';
