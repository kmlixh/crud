-- 创建数据库
CREATE DATABASE IF NOT EXISTS test DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

USE test;

-- 创建用户表
CREATE TABLE IF NOT EXISTS users (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    username VARCHAR(50) NOT NULL UNIQUE COMMENT '用户名',
    password VARCHAR(100) NOT NULL COMMENT '密码',
    email VARCHAR(100) NOT NULL UNIQUE COMMENT '邮箱',
    status INT NOT NULL DEFAULT 1 COMMENT '状态：1-正常，0-禁用',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间'
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='用户表';

-- 创建文章表
CREATE TABLE IF NOT EXISTS articles (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    title VARCHAR(200) NOT NULL COMMENT '标题',
    content TEXT NOT NULL COMMENT '内容',
    user_id BIGINT NOT NULL COMMENT '作者ID',
    status INT NOT NULL DEFAULT 1 COMMENT '状态：1-正常，0-草稿，-1-删除',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    FOREIGN KEY (user_id) REFERENCES users(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='文章表';

-- 插入测试数据
INSERT INTO users (username, password, email, status) VALUES
    ('admin', '123456', 'admin@example.com', 1),
    ('test', '123456', 'test@example.com', 1),
    ('demo', '123456', 'demo@example.com', 0);

INSERT INTO articles (title, content, user_id, status) VALUES
    ('Hello World', 'This is my first article.', 1, 1),
    ('Getting Started', 'A guide to getting started with our platform.', 1, 1),
    ('Draft Article', 'This is a draft article.', 2, 0),
    ('Another Article', 'This is another article.', 2, 1),
    ('Deleted Article', 'This article has been deleted.', 3, -1);