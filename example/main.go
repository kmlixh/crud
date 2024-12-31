package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

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

	// 创建自动CRUD处理器
	userCrud := crud.New2(db, &User{})

	// 修改默认处理器的字段控制
	if listHandler, ok := userCrud.GetHandler(crud.LIST); ok {
		listHandler.AllowedFields = []string{"id", "username", "email", "status", "created_at"} // 列表不返回密码
		// 默认按创建时间降序，用户名升序排序
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

	userCrud.AddHandler("active_users", activeUsersHandler.Method, *activeUsersHandler)

	// 登录处理器
	loginHandler := crud.NewHandler("/login", http.MethodPost).
		PreProcess(func(ctx *crud.ProcessContext) error {
			var req struct {
				Username string `json:"username" binding:"required"`
				Password string `json:"password" binding:"required"`
			}
			if err := ctx.GinContext.ShouldBindJSON(&req); err != nil {
				return fmt.Errorf("invalid request: %v", err)
			}
			ctx.Data["request"] = req
			return nil
		}).
		BuildQuery(func(ctx *crud.ProcessContext) error {
			req := ctx.Data["request"].(struct {
				Username string
				Password string
			})

			ctx.Chain = db.Chain().Table("users").
				Eq("username", req.Username).
				Eq("password", req.Password)
			return nil
		}).
		ExecuteStep(func(ctx *crud.ProcessContext) error {
			result := ctx.Chain.One()
			if err := result.Error(); err != nil {
				return fmt.Errorf("failed to query user: %v", err)
			}
			if result.Empty() {
				return fmt.Errorf("invalid username or password")
			}
			ctx.Data["user"] = result.Data[0]
			return nil
		}).
		PostProcess(func(ctx *crud.ProcessContext) error {
			user := ctx.Data["user"].(map[string]interface{})
			// 生成 token
			token := crud.GenerateToken(
				fmt.Sprintf("%v", user["id"]),
				"user",
				24*time.Hour,
			)
			// 保存 token
			if err := crud.GetTokenStore().SaveToken(token); err != nil {
				return fmt.Errorf("failed to save token: %v", err)
			}
			crud.CodeMsgFunc(ctx.GinContext, crud.CodeSuccess, "login success", gin.H{
				"token": token.Token,
				"user":  user,
			})
			return nil
		})

	userCrud.AddHandler("login", loginHandler.Method, *loginHandler)

	// 注册路由
	api := r.Group("/api")
	{
		// 公开接口
		userCrud.Register(api.Group("/public/users"))

		// 需要认证的接口
		auth := api.Group("/users")
		auth.Use(crud.GlobalTokenAuthMiddleware())
		userCrud.Register(auth)

		// 获取当前用户信息
		auth.GET("/me", func(c *gin.Context) {
			userID, userType, ok := crud.GetCurrentUser(c)
			if !ok {
				crud.CodeMsgFunc(c, crud.CodeError, "unauthorized", nil)
				return
			}
			crud.CodeMsgFunc(c, crud.CodeSuccess, "success", gin.H{
				"user_id":   userID,
				"user_type": userType,
			})
		})

		// 退出登录
		auth.POST("/logout", func(c *gin.Context) {
			token, ok := crud.GetCurrentToken(c)
			if !ok {
				crud.CodeMsgFunc(c, crud.CodeError, "unauthorized", nil)
				return
			}
			if err := crud.GetTokenStore().DeleteToken(token); err != nil {
				crud.CodeMsgFunc(c, crud.CodeError, fmt.Sprintf("failed to logout: %v", err), nil)
				return
			}
			crud.CodeMsgFunc(c, crud.CodeSuccess, "logout success", nil)
		})
	}

	// 启动服务器
	if err := r.Run(":8080"); err != nil {
		log.Fatal(err)
	}
}
