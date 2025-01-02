-- 创建测试数据库
CREATE DATABASE IF NOT EXISTS test;
USE test;

-- 创建用户表
DROP TABLE IF EXISTS users;
CREATE TABLE users (
    id INT AUTO_INCREMENT PRIMARY KEY,
    username VARCHAR(50) NOT NULL,
    email VARCHAR(100) NOT NULL,
    age INT,
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE KEY idx_username (username),
    UNIQUE KEY idx_email (email),
    KEY idx_status (status),
    KEY idx_age (age)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;