# AutoCrudGo

一个简单而强大的 Go 语言 CRUD API 生成器，基于 Gin 和 Gom。

## 特性

- 基于结构体标签自动映射数据库字段
- 支持自定义路由前缀
- 支持字段过滤（查询、更新、创建）
- 支持字段排除
- 支持分页查询
- 支持条件过滤
- 支持排序
- 支持自定义主键

## 安装

```bash
go get github.com/kmlixh/crud
```

## 快速开始

1. 定义模型：

```go
type User struct {
    Id        int64     `json:"id" gom:"id,primary,auto_increment"`
    Username  string    `json:"username" gom:"username"`
    Password  string    `json:"-" gom:"password"`
    Email     string    `json:"email" gom:"email"`
    Status    int       `json:"status" gom:"status"`
    CreatedAt time.Time `json:"createdAt" gom:"created_at"`
    UpdatedAt time.Time `json:"updatedAt" gom:"updated_at"`
}
```

2. 注册路由：

```go
// 初始化数据库
db, _ := gom.Open("mysql", "user:pass@tcp(host:port)/dbname?charset=utf8mb4")

// 初始化 Gin
engine := gin.Default()

// 注册 CRUD 路由
crud.Register(engine, db, User{}, crud.Options{
    PathPrefix:    "/api/users",
    QueryFields:   []string{"id", "username", "email", "status"},
    UpdateFields:  []string{"username", "email", "status"},
    CreateFields:  []string{"username", "password", "email", "status"},
    ExcludeFields: []string{"password"},
})

// 启动服务器
engine.Run(":8080")
```

## API 说明

### 列表查询

```http
GET /api/users?pageNum=1&pageSize=10
```

支持的查询参数：
- `pageNum`: 页码（默认 1）
- `pageSize`: 每页大小（默认 10）
- `orderBy`: 排序字段，多个字段用逗号分隔，前缀 `-` 表示降序
- 字段查询：直接使用字段名作为参数
- 模糊查询：字段名加后缀 `_like`
- 范围查询：字段名加后缀 `_gt`、`_gte`、`_lt`、`_lte`

### 获取单条记录

```http
GET /api/users/:id
```

### 创建记录

```http
POST /api/users
Content-Type: application/json

{
    "username": "test",
    "password": "123456",
    "email": "test@example.com",
    "status": 1
}
```

### 更新记录

```http
PUT /api/users/:id
Content-Type: application/json

{
    "username": "new_name",
    "email": "new_email@example.com",
    "status": 1
}
```

### 删除记录

```http
DELETE /api/users/:id
```

## 配置选项

```go
type Options struct {
    // 路由前缀
    PathPrefix string
    // 主键字段（默认为 "id"）
    PrimaryKey string
    // 可查询字段（为空表示所有字段）
    QueryFields []string
    // 可更新字段（为空表示所有字段）
    UpdateFields []string
    // 可创建字段（为空表示所有字段）
    CreateFields []string
    // 排除字段
    ExcludeFields []string
}
```

## 响应格式

### 成功响应

```json
{
    "code": 0,
    "data": {
        // 响应数据
    }
}
```

### 分页响应

```json
{
    "code": 0,
    "data": {
        "pageNum": 1,
        "pageSize": 10,
        "total": 100,
        "totalPages": 10,
        "data": [
            // 数据列表
        ]
    }
}
```

### 错误响应

```json
{
    "code": 500,
    "message": "错误信息"
}
```

## 完整示例

查看 [example](./example) 目录获取完整的示例代码。

## 许可证

MIT License 