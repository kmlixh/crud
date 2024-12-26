package main

import (
	"log"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/kmlixh/crud"
	"github.com/kmlixh/gom/v4"
	_ "github.com/kmlixh/gom/v4/factory/mysql"
)

// User 用户模型
type User struct {
	Id        int64     `json:"id" gom:"id,primary,auto_increment"`
	Username  string    `json:"username" gom:"username"`
	Password  string    `json:"-" gom:"password"` // 密码不返回给前端
	Email     string    `json:"email" gom:"email"`
	Status    int       `json:"status" gom:"status"`
	CreatedAt time.Time `json:"createdAt" gom:"created_at"`
	UpdatedAt time.Time `json:"updatedAt" gom:"updated_at"`
}

// Article 文章模型
type Article struct {
	Id        int64     `json:"id" gom:"id,primary,auto_increment"`
	Title     string    `json:"title" gom:"title"`
	Content   string    `json:"content" gom:"content"`
	UserId    int64     `json:"userId" gom:"user_id"`
	Status    int       `json:"status" gom:"status"`
	CreatedAt time.Time `json:"createdAt" gom:"created_at"`
	UpdatedAt time.Time `json:"updatedAt" gom:"updated_at"`
}

func main() {
	// 初始化数据库连接
	db, err := gom.Open("mysql", "root:123456@tcp(localhost:3306)/test?charset=utf8mb4&parseTime=True&loc=Local", false)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// 初始化 Gin 引擎
	engine := gin.Default()

	// 注册用户 CRUD 路由
	crud.Register(engine, db, User{}, crud.Options{
		PathPrefix:    "/api/users",
		QueryFields:   []string{"id", "username", "email", "status", "created_at"},
		UpdateFields:  []string{"username", "email", "status"},
		CreateFields:  []string{"username", "password", "email", "status"},
		ExcludeFields: []string{"password"}, // 查询时排除密码字段
	})

	// 注册文章 CRUD 路由（使用默认配置）
	crud.Register(engine, db, Article{}, crud.Options{
		PathPrefix:   "/api/articles",
		UpdateFields: []string{"title", "content", "status"},
		CreateFields: []string{"title", "content", "user_id", "status"},
	})

	// 启动服务器
	if err := engine.Run(":8080"); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
