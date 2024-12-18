CREATE DATABASE IF NOT EXISTS `DH_Blog` DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci;

USE `DH_Blog`;

-- 创建文章表 (`articles`)
DROP TABLE IF EXISTS `articles`;

CREATE TABLE `articles` (
    `id` INT NOT NULL AUTO_INCREMENT COMMENT '唯一标识每篇文章',
    `title` VARCHAR(255) NOT NULL COMMENT '文章标题',
    `content` TEXT NOT NULL COMMENT '文章内容',
    `category_id` INT DEFAULT NULL COMMENT '引用 categories 表的 id, 表示文章分类',
    `create_time` DATETIME NOT NULL COMMENT '文章创建时间',
    `update_time` DATETIME DEFAULT NULL COMMENT '文章的最后更新时间',
    `views` INT DEFAULT 0 COMMENT '记录文章浏览次数',
    `word_num` INT DEFAULT NULL COMMENT '文章字数',
    `thumbnail_url` VARCHAR(255) DEFAULT NULL COMMENT '文章缩略图的URL',
    `is_locked` TINYINT(1) DEFAULT 0 COMMENT '文章是否被锁定，0表示未锁定，1表示锁定',
    `lock_password` VARCHAR(16) DEFAULT NULL COMMENT '锁定密码，用于解锁文章',
    PRIMARY KEY (`id`)
) ENGINE = InnoDB;

-- 创建文章标签关联表 (`posttags`)
DROP TABLE IF EXISTS `posttags`;

CREATE TABLE `posttags` (
    `post_id` INT NOT NULL COMMENT '引用 articles 表的 id',
    `tag_id` INT NOT NULL COMMENT '引用 tags 表的 id',
    PRIMARY KEY (`post_id`, `tag_id`)
) ENGINE = InnoDB;

-- 创建标签表 (`tags`)
DROP TABLE IF EXISTS `tags`;

CREATE TABLE `tags` (
    `id` INT NOT NULL AUTO_INCREMENT COMMENT '唯一标识每个标签',
    `name` VARCHAR(255) NOT NULL COMMENT '标签名',
    `slug` VARCHAR(255) NOT NULL COMMENT 'URL友好的字符串，用于生成标签页面链接',
    `create_time` DATETIME DEFAULT NULL COMMENT '标签创建时间',
    `update_time` DATETIME DEFAULT NULL COMMENT '标签的最后更新时间',
    PRIMARY KEY (`id`),
    UNIQUE INDEX `name` (`name`) COMMENT '标签名的唯一索引',
    UNIQUE INDEX `slug` (`slug`) COMMENT '标签 slug 的唯一索引'
) ENGINE = InnoDB;

-- 创建分类表 (`categories`)
DROP TABLE IF EXISTS `categories`;

CREATE TABLE `categories` (
    `id` INT NOT NULL AUTO_INCREMENT COMMENT '唯一标识每个分类',
    `name` VARCHAR(255) NOT NULL COMMENT '分类名称',
    `slug` VARCHAR(255) NOT NULL COMMENT 'URL友好的字符串, 通常用于生成分类页面的链接',
    `create_time` DATETIME DEFAULT NULL COMMENT '分类创建时间',
    `update_time` DATETIME DEFAULT NULL COMMENT '分类的最后更新时间',
    PRIMARY KEY (`id`),
    UNIQUE INDEX `name` (`name`) COMMENT '分类名的唯一索引',
    UNIQUE INDEX `slug` (`slug`) COMMENT '分类 slug 的唯一索引'
) ENGINE = InnoDB;

-- 创建分类关联默认标签表 (`category_default_tags`)
DROP TABLE IF EXISTS `category_default_tags`;

CREATE TABLE `category_default_tags` (
    `category_id` INT NOT NULL COMMENT '引用 categories 表的 id',
    `tag_id` INT NOT NULL COMMENT '引用 tags 表的 id',
    PRIMARY KEY (`category_id`, `tag_id`)
) ENGINE = InnoDB;

-- 创建用户表 (`users`)
DROP TABLE IF EXISTS `users`;

CREATE TABLE `users` (
    `id` INT NOT NULL AUTO_INCREMENT COMMENT '唯一标识每个用户',
    `username` CHAR(16) NOT NULL COMMENT '用户名',
    `password` CHAR(60) NOT NULL COMMENT 'BCrypt 加密后的密码',
    `create_time` DATETIME NOT NULL COMMENT '用户创建时间',
    PRIMARY KEY (`id`),
    UNIQUE INDEX `username` (`username`) COMMENT '用户名的唯一索引'
) ENGINE = InnoDB;

-- 创建每日统计数据表 (`daily_stats`)
CREATE TABLE daily_stats (
    id INT AUTO_INCREMENT PRIMARY KEY COMMENT '唯一标识每一天的统计数据',
    date DATE NOT NULL UNIQUE COMMENT '记录日期',
    visit_count INT NOT NULL DEFAULT 0 COMMENT '记录当天的访问量',
    article_count INT NOT NULL DEFAULT 0 COMMENT '记录当天的文章发布数量',
    comment_count INT NOT NULL DEFAULT 0 COMMENT '记录当天的评论数量',
    tag_count INT NOT NULL DEFAULT 0 COMMENT '记录当天的标签发布数量'
);

-- 访问记录表 (`access_logs`)
CREATE TABLE access_logs (
    id BIGINT AUTO_INCREMENT PRIMARY KEY COMMENT '唯一标识每次访问',
    ip_address CHAR(15) NOT NULL COMMENT '记录访问者的IP地址',
    access_time DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '记录访问时间',
    user_agent VARCHAR(255) COMMENT '记录用户的浏览器信息',
    request_url VARCHAR(255) COMMENT '记录请求的URL'
) COMMENT='记录每次访问的详细信息';

-- ip 地址统计表 (`ip_stats`)
CREATE TABLE ip_stats (
    id INT AUTO_INCREMENT PRIMARY KEY COMMENT '唯一标识每个IP地址的统计信息',
    ip_address CHAR(15) NOT NULL UNIQUE COMMENT '记录IP地址',
    city CHAR(20) COMMENT '记录IP所在的城市',
    access_count INT NOT NULL DEFAULT 0 COMMENT '记录该IP地址的访问次数',
    banned_count INT NOT NULL DEFAULT 0 COMMENT '记录该IP地址被封禁的次数',
    ban_status TINYINT NOT NULL DEFAULT 0 COMMENT '记录该IP地址是否被封禁，0表示未封禁，1表示封禁',
) COMMENT='记录每个IP地址的访问次数和被封禁次数';

-- 评论表 (`comments`)
CREATE TABLE comments (
    id INT AUTO_INCREMENT PRIMARY KEY COMMENT '唯一标识每条评论',
    article_id INT NOT NULL COMMENT '记录评论所属的文章ID',
    author VARCHAR(32) NOT NULL COMMENT '记录评论的作者名',
    email VARCHAR(32) NOT NULL COMMENT '记录评论作者的电子邮件',
    content VARCHAR(255) NOT NULL COMMENT '记录评论内容',
    is_public TINYINT NOT NULL DEFAULT 1 COMMENT '表示评论是否公开',
    create_time DATETIME NOT NULL COMMENT '记录评论的创建时间',
    parent_id INT DEFAULT NULL COMMENT '记录父评论的ID',
    ua VARCHAR(32) NOT NULL COMMENT '记录评论作者的User Agent',
    is_admin TINYINT DEFAULT 0 COMMENT '表示评论作者是否为管理员'
) COMMENT='存储文章的评论信息' CHARACTER SET utf8mb4;
