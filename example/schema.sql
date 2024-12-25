-- 创建数据库
CREATE DATABASE IF NOT EXISTS test DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci;

-- 使用数据库
USE test;

-- 删除已存在的表
DROP TABLE IF EXISTS `users`;

-- 创建用户表
CREATE TABLE `users` (
  `id` bigint(20) NOT NULL AUTO_INCREMENT,
  `username` varchar(50) NOT NULL COMMENT '用户名',
  `password` varchar(100) NOT NULL COMMENT '密码',
  `age` int(11) DEFAULT NULL COMMENT '年龄',
  `email` varchar(100) DEFAULT NULL COMMENT '邮箱',
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_username` (`username`)
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8mb4 COMMENT='用户表';

-- 插入测试数据
INSERT INTO `users` (`username`, `password`, `age`, `email`) VALUES
('admin', '123456', 30, 'admin@example.com'),
('test1', '123456', 25, 'test1@example.com'),
('test2', '123456', 28, 'test2@example.com'); 