package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/kmlixh/crud"
	"github.com/kmlixh/gom/v4"
	_ "github.com/kmlixh/gom/v4/factory/mysql"
	"github.com/redis/go-redis/v9"
)

// User 用户模型
type User struct {
	ID        int64  `json:"id" gorm:"primaryKey;autoIncrement"`
	Username  string `json:"username" validate:"required"`
	Password  string `json:"password" validate:"required"`
	Email     string `json:"email" validate:"required,email"`
	Age       int    `json:"age"`
	Status    string `json:"status"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

func main() {
	// 初始化数据库连接
	db, err := gom.Open("mysql", "root:password@tcp(localhost:3306)/test?charset=utf8mb4&parseTime=True&loc=Local", false)
	if err != nil {
		log.Fatal(err)
	}

	// 初始化 Redis 连接
	redisClient := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "", // no password set
		DB:       0,  // use default DB
	})

	// 创建 Redis token 存储并设置为全局存储
	tokenStore := crud.NewRedisTokenStore(redisClient, "app")
	crud.SetTokenStore(tokenStore)

	// 创建路由
	r := gin.Default()

	// 创建API组
	api := r.Group("/api")

	// 创建自动CRUD处理器
	userCrud := crud.New2(db, &User{})

	// 修改默认处理器的字段控制
	if listHandler, ok := userCrud.GetHandler(crud.LIST); ok {
		listHandler.AllowedFields = []string{"id", "username", "email", "status", "created_at"} // 列表不返回密码
		chain := db.Chain()
		chain.OrderByDesc("created_at").OrderBy("username")
		listHandler.BuildQuery(func(ctx *crud.ProcessContext) error {
			ctx.Chain = chain
			return nil
		})
		userCrud.AddHandler(crud.LIST, listHandler.Method, listHandler)
	}

	if detailHandler, ok := userCrud.GetHandler(crud.SINGLE); ok {
		detailHandler.AllowedFields = []string{"id", "username", "email", "age", "status", "created_at", "updated_at"} // 详情不返回密码
		userCrud.AddHandler(crud.SINGLE, detailHandler.Method, detailHandler)
	}

	if updateHandler, ok := userCrud.GetHandler(crud.UPDATE); ok {
		updateHandler.AllowedFields = []string{"username", "email", "age", "status"} // 更新时不允许修改密码和时间戳
		userCrud.AddHandler(crud.UPDATE, updateHandler.Method, updateHandler)
	}

	// 添加自定义处理器 - 获取活跃用户列表
	activeUsersHandler := crud.NewHandler("/active", http.MethodGet).
		PreProcess(func(ctx *crud.ProcessContext) error {
			return nil
		}).
		BuildQuery(func(ctx *crud.ProcessContext) error {
			ctx.Chain = db.Chain().Table("users").
				Eq("status", "active").
				OrderBy("username") // 按用户名升序排序
			return nil
		}).
		ExecuteStep(func(ctx *crud.ProcessContext) error {
			result := ctx.Chain.List()
			if err := result.Error(); err != nil {
				return fmt.Errorf("failed to query users: %v", err)
			}
			ctx.Data["result"] = result.Data
			return nil
		}).
		PostProcess(func(ctx *crud.ProcessContext) error {
			crud.CodeMsgFunc(ctx.GinContext, crud.CodeSuccess, "success", ctx.Data["result"])
			return nil
		})

	userCrud.AddHandler("active_users", activeUsersHandler.Method, activeUsersHandler)

	// 注册用户相关路由
	userCrud.RegisterRoutes(api, "/users")

	// 注册API信息路由
	crud.RegisterApi(r, "/api-info")

	// 启动服务器
	r.Run(":8080")
}
