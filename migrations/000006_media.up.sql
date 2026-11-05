CREATE TABLE `vx_image` (
  `image_id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `storage_key` char(32) NOT NULL COMMENT '随机存储键，同时作为目录名',
  `filename` varchar(255) NOT NULL COMMENT '原始文件名，仅用于展示',
  `mime` varchar(32) NOT NULL,
  `size` int unsigned NOT NULL COMMENT '原图字节数',
  `width` int unsigned NOT NULL,
  `height` int unsigned NOT NULL,
  `kind` varchar(16) NOT NULL COMMENT 'content、avatar、cover、topic_cover',
  `user_id` bigint unsigned NOT NULL,
  `created_at` datetime(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  `deleted_at` datetime(6) DEFAULT NULL,
  PRIMARY KEY (`image_id`),
  UNIQUE KEY `uk_image_storage_key` (`storage_key`),
  KEY `idx_image_user` (`user_id`, `deleted_at`),
  CONSTRAINT `fk_image_user` FOREIGN KEY (`user_id`) REFERENCES `vx_user` (`user_id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='图片';
