-- SQLite doesn't use CREATE DATABASE, it works with individual files
-- Just open or create the SQLite database file

-- 创建文章表 (`articles`)
DROP TABLE IF EXISTS `articles`;

CREATE TABLE `articles` (
  `id` INTEGER PRIMARY KEY AUTOINCREMENT COMMENT '唯一标识每篇文章',
  `title` TEXT NOT NULL COMMENT '文章标题',
  `content` TEXT NOT NULL COMMENT '文章内容',
  `category_id` INTEGER DEFAULT NULL COMMENT '引用 categories 表的 id, 表示文章分类',
  `create_time` TEXT NOT NULL COMMENT '文章创建时间',
  `update_time` TEXT DEFAULT NULL COMMENT '文章的最后更新时间',
  `views` INTEGER DEFAULT 0 COMMENT '记录文章浏览次数',
  `word_num` INTEGER DEFAULT NULL COMMENT '文章字数',
  `thumbnail_url` TEXT DEFAULT NULL COMMENT '文章缩略图的URL',
  `is_locked` INTEGER DEFAULT 0 COMMENT '文章是否被锁定，0表示未锁定，1表示锁定',
  `lock_password` TEXT DEFAULT NULL COMMENT '锁定密码，用于解锁文章'
);

-- 创建文章标签关联表 (`posttags`)
DROP TABLE IF EXISTS `posttags`;

CREATE TABLE `posttags` (
  `post_id` INTEGER NOT NULL COMMENT '引用 articles 表的 id',
  `tag_id` INTEGER NOT NULL COMMENT '引用 tags 表的 id',
  PRIMARY KEY (`post_id`, `tag_id`)
);

-- 创建标签表 (`tags`)
DROP TABLE IF EXISTS `tags`;

CREATE TABLE `tags` (
                        `id` INTEGER PRIMARY KEY AUTOINCREMENT COMMENT '唯一标识每个标签',
                        `name` TEXT NOT NULL COMMENT '标签名',
                        `slug` TEXT NOT NULL COMMENT 'URL友好的字符串，用于生成标签页面链接',
                        `create_time` TEXT DEFAULT NULL COMMENT '标签创建时间',
                        `update_time` TEXT DEFAULT NULL COMMENT '标签的最后更新时间'
);

-- 虽然SQLite不支持在CREATE TABLE语句中直接创建索引时添加注释，但我们可以单独创建索引
CREATE UNIQUE INDEX `idx_tags_name` ON `tags` (`name`); -- 标签名的唯一索引
CREATE UNIQUE INDEX `idx_tags_slug` ON `tags` (`slug`); -- 标签 slug 的唯一索引

-- 创建分类表 (`categories`)
DROP TABLE IF EXISTS `categories`;

CREATE TABLE `categories` (
    `id` INTEGER PRIMARY KEY AUTOINCREMENT COMMENT '唯一标识每个分类',
    `name` TEXT NOT NULL COMMENT '分类名称',
    `slug` TEXT NOT NULL COMMENT 'URL友好的字符串, 通常用于生成分类页面的链接',
    `create_time` TEXT DEFAULT NULL COMMENT '分类创建时间',
    `update_time` TEXT DEFAULT NULL COMMENT '分类的最后更新时间'
);

CREATE UNIQUE INDEX `idx_categories_name` ON `categories` (`name`); -- 分类名的唯一索引
CREATE UNIQUE INDEX `idx_categories_slug` ON `categories` (`slug`); -- 分类 slug 的唯一索引

-- 创建分类关联默认标签表 (`category_default_tags`)
DROP TABLE IF EXISTS `category_default_tags`;

CREATE TABLE `category_default_tags` (
               `category_id` INTEGER NOT NULL COMMENT '引用 categories 表的 id',
               `tag_id` INTEGER NOT NULL COMMENT '引用 tags 表的 id',
               PRIMARY KEY (`category_id`, `tag_id`)
);

-- 创建用户表 (`users`)
DROP TABLE IF EXISTS `users`;

CREATE TABLE `users` (
                         `id` INTEGER PRIMARY KEY AUTOINCREMENT COMMENT '唯一标识每个用户',
                         `username` TEXT NOT NULL COMMENT '用户名',
                         `password` TEXT NOT NULL COMMENT 'BCrypt 加密后的密码',
                         `create_time` TEXT NOT NULL COMMENT '用户创建时间'
);

CREATE UNIQUE INDEX `idx_users_username` ON `users` (`username`); -- 用户名的唯一索引

-- 创建每日统计数据表 (`daily_stats`)
DROP TABLE IF EXISTS `daily_stats`;

CREATE TABLE `daily_stats` (
     `id` INTEGER PRIMARY KEY AUTOINCREMENT COMMENT '唯一标识每一天的统计数据',
     `date` TEXT NOT NULL UNIQUE COMMENT '记录日期',
     `visit_count` INTEGER NOT NULL DEFAULT 0 COMMENT '记录当天的访问量',
     `article_count` INTEGER NOT NULL DEFAULT 0 COMMENT '记录当天的文章发布数量',
     `comment_count` INTEGER NOT NULL DEFAULT 0 COMMENT '记录当天的评论数量',
     `tag_count` INTEGER NOT NULL DEFAULT 0 COMMENT '记录当天的标签发布数量'
);

-- 访问记录表 (`access_logs`)
DROP TABLE IF EXISTS `access_logs`;

CREATE TABLE `access_logs` (
     `id` INTEGER PRIMARY KEY AUTOINCREMENT COMMENT '唯一标识每次访问',
     `ip_address` TEXT NOT NULL COMMENT '记录访问者的IP地址',
     `access_time` TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '记录访问时间',
     `user_agent` TEXT COMMENT '记录用户的浏览器信息',
     `request_url` TEXT COMMENT '记录请求的URL'
);

-- ip 地址统计表 (`ip_stats`)
DROP TABLE IF EXISTS `ip_stats`;

CREATE TABLE `ip_stats` (
  `id` INTEGER PRIMARY KEY AUTOINCREMENT COMMENT '唯一标识每个IP地址的统计信息',
  `ip_address` TEXT NOT NULL UNIQUE COMMENT '记录IP地址',
  `city` TEXT COMMENT '记录IP所在的城市',
  `access_count` INTEGER NOT NULL DEFAULT 0 COMMENT '记录该IP地址的访问次数',
  `banned_count` INTEGER NOT NULL DEFAULT 0 COMMENT '记录该IP地址被封禁的次数',
  `ban_status` INTEGER NOT NULL DEFAULT 0 COMMENT '记录该IP地址是否被封禁，0表示未封禁，1表示封禁'
);

-- 评论表 (`comments`)
DROP TABLE IF EXISTS `comments`;

CREATE TABLE `comments` (
  `id` INTEGER PRIMARY KEY AUTOINCREMENT COMMENT '唯一标识每条评论',
  `article_id` INTEGER NOT NULL COMMENT '记录评论所属的文章ID',
  `author` TEXT NOT NULL COMMENT '记录评论的作者名',
  `email` TEXT NOT NULL COMMENT '记录评论作者的电子邮件',
  `content` TEXT NOT NULL COMMENT '记录评论内容',
  `is_public` INTEGER NOT NULL DEFAULT 1 COMMENT '表示评论是否公开',
  `create_time` TEXT NOT NULL COMMENT '记录评论的创建时间',
  `parent_id` INTEGER DEFAULT NULL COMMENT '记录父评论的ID',
  `ua` TEXT NOT NULL COMMENT '记录评论作者的User Agent',
  `is_admin` INTEGER DEFAULT 0 COMMENT '表示评论作者是否为管理员'
);