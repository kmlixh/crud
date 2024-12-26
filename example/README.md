# CRUD 示例

这是一个使用 AutoCrudGo 框架的完整示例，展示了如何快速构建用户和文章的 CRUD API。

## 功能特点

- 用户管理：包含基本的用户信息管理功能
- 文章管理：包含文章的增删改查功能
- 支持分页查询
- 支持条件过滤
- 支持字段排序
- 支持字段过滤

## 数据库设计

### 用户表 (users)

| 字段 | 类型 | 说明 |
|------|------|------|
| id | BIGINT | 主键 |
| username | VARCHAR(50) | 用户名 |
| password | VARCHAR(100) | 密码 |
| email | VARCHAR(100) | 邮箱 |
| status | INT | 状态：1-正常，0-禁用 |
| created_at | TIMESTAMP | 创建时间 |
| updated_at | TIMESTAMP | 更新时间 |

### 文章表 (articles)

| 字段 | 类型 | 说明 |
|------|------|------|
| id | BIGINT | 主键 |
| title | VARCHAR(200) | 标题 |
| content | TEXT | 内容 |
| user_id | BIGINT | 作者ID |
| status | INT | 状态：1-正常，0-草稿，-1-删除 |
| created_at | TIMESTAMP | 创建时间 |
| updated_at | TIMESTAMP | 更新时间 |

## 快速开始

1. 初始化数据库：

```bash
mysql -u root -p < init.sql
```

2. 修改数据库连接配置：

打开 `main.go`，修改数据库连接字符串：

```go
db, err := gom.Open("mysql", "root:123456@tcp(localhost:3306)/test?charset=utf8mb4&parseTime=True&loc=Local")
```

3. 运行示例：

```bash
go run main.go
```

4. 测试 API：

使用 VS Code 的 REST Client 插件或其他工具（如 Postman）运行 `test.http` 中的测试用例。

## API 文档

### 用户 API

#### 列出用户
```http
GET /api/users?pageNum=1&pageSize=10
```

支持的查询参数：
- `pageNum`: 页码（默认 1）
- `pageSize`: 每页大小（默认 10）
- `orderBy`: 排序字段，多个字段用逗号分隔，前缀 `-` 表示降序
- `username_like`: 用户名模糊搜索
- `email_like`: 邮箱模糊搜索
- `status`: 状态过滤
- `created_at_gte`: 创建时间大于等于
- `created_at_lt`: 创建时间小于

#### 获取单个用户
```http
GET /api/users/:id
```

#### 创建用户
```http
POST /api/users
Content-Type: application/json

{
    "username": "newuser",
    "password": "123456",
    "email": "newuser@example.com",
    "status": 1
}
```

#### 更新用户
```http
PUT /api/users/:id
Content-Type: application/json

{
    "username": "updateduser",
    "email": "updated@example.com",
    "status": 1
}
```

#### 删除用户
```http
DELETE /api/users/:id
```

### 文章 API

#### 列出文章
```http
GET /api/articles?pageNum=1&pageSize=10
```

支持的查询参数：
- `pageNum`: 页码（默认 1）
- `pageSize`: 每页大小（默认 10）
- `orderBy`: 排序字段，多个字段用逗号分隔，前缀 `-` 表示降序
- `title_like`: 标题模糊搜索
- `content_like`: 内容模糊搜索
- `user_id`: 作者ID过滤
- `status`: 状态过滤
- `created_at_gte`: 创建时间大于等于
- `created_at_lt`: 创建时间小于

#### 获取单个文章
```http
GET /api/articles/:id
```

#### 创建文章
```http
POST /api/articles
Content-Type: application/json

{
    "title": "New Article",
    "content": "Article content",
    "user_id": 1,
    "status": 1
}
```

#### 更新文章
```http
PUT /api/articles/:id
Content-Type: application/json

{
    "title": "Updated Article",
    "content": "Updated content",
    "status": 1
}
```

#### 删除文章
```http
DELETE /api/articles/:id
```

## 响应格式

所有 API 响应都使用统一的 JSON 格式：

```json
{
    "code": 0,       // 状态码：0-成功，非0-失败
    "message": "",   // 错误消息（仅在失败时有值）
    "data": {}       // 响应数据
}
```

分页查询的响应格式：

```json
{
    "code": 0,
    "data": {
        "pageNum": 1,        // 当前页码
        "pageSize": 10,      // 每页大小
        "total": 100,        // 总记录数
        "totalPages": 10,    // 总页数
        "data": []           // 数据列表
    }
}
``` 