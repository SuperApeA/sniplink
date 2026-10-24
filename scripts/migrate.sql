CREATE TABLE IF NOT EXISTS `links` (
    `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '主键ID',
    `code` varchar(16) COLLATE utf8mb4_bin NOT NULL COMMENT '短码',
    `url` varchar(2048) NOT NULL COMMENT '原始长链接',
    `click_count` bigint unsigned NOT NULL DEFAULT '0' COMMENT '点击次数',
    `expires_at` datetime(3) DEFAULT NULL COMMENT '过期时间',
    `created_at` datetime(3) NOT NULL COMMENT '创建时间',
    `updated_at` datetime(3) NOT NULL COMMENT '更新时间',
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_code` (`code`),
    KEY `idx_expires_at` (`expires_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='短链信息表';
