-- 创建数据库
CREATE DATABASE IF NOT EXISTS test DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci;

USE test;

-- 创建用户表
CREATE TABLE IF NOT EXISTS users (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    username VARCHAR(50) NOT NULL COMMENT '用户名',
    password VARCHAR(100) NOT NULL COMMENT '密码',
    email VARCHAR(100) NOT NULL COMMENT '邮箱',
    status INT NOT NULL DEFAULT 1 COMMENT '状态：1-正常，0-禁用',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    UNIQUE KEY uk_username (username),
    UNIQUE KEY uk_email (email)
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
    KEY idx_user_id (user_id),
    KEY idx_status (status),
    CONSTRAINT fk_articles_user_id FOREIGN KEY (user_id) REFERENCES users (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='文章表';

-- 插入测试数据
INSERT INTO users (username, password, email, status) VALUES
('admin', '123456', 'admin@example.com', 1),
('user1', '123456', 'user1@example.com', 1),
('user2', '123456', 'user2@example.com', 1);

INSERT INTO articles (title, content, user_id, status) VALUES
('First Article', 'This is the first article content.', 1, 1),
('Second Article', 'This is the second article content.', 1, 1),
('Draft Article', 'This is a draft article.', 2, 0); 