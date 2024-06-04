-- 用户表
CREATE TABLE Users (
    id INT AUTO_INCREMENT PRIMARY KEY,
    username VARCHAR(255) NOT NULL UNIQUE,
    password VARCHAR(255) NOT NULL,
    register_date DATETIME NOT NULL
);

-- 登录信息表
CREATE TABLE LoginInfo (
    id INT AUTO_INCREMENT PRIMARY KEY,
    login_time DATETIME NOT NULL,
    ip VARCHAR(15) NOT NULL,
    city VARCHAR(100) NOT NULL
);

-- 博客文章表
CREATE TABLE Articles (
    id INT AUTO_INCREMENT PRIMARY KEY,
    title VARCHAR(255) NOT NULL,
    content TEXT NOT NULL,
    category_id INT, -- 外键
    publish_date DATETIME NOT NULL,
    update_date DATETIME,
    views INT DEFAULT 0,
    word_num TINYINT DEFAULT 0
);

-- 分类表
CREATE TABLE Categories (
    id INT AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(255) NOT NULL UNIQUE,
    slug VARCHAR(255) NOT NULL UNIQUE,
    created_at DATETIME,
    updated_at DATETIME
);

-- 标签表
CREATE TABLE Tags (
    id INT AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(255) NOT NULL UNIQUE,
    slug VARCHAR(255) NOT NULL UNIQUE,
    created_at DATETIME,
    updated_at DATETIME
);

-- 文章与标签的关联表
CREATE TABLE PostTags (
    post_id INT,
    tag_id INT,
    PRIMARY KEY (post_id, tag_id), -- 组合主键确保每篇文章和标签的配对唯一
    FOREIGN KEY (post_id) REFERENCES BlogPosts(id) ON DELETE CASCADE,
    FOREIGN KEY (tag_id) REFERENCES Tags(id) ON DELETE CASCADE
);