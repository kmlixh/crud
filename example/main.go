package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/kmlixh/crud"
	"github.com/kmlixh/gom/v4"
	_ "github.com/kmlixh/gom/v4/factory/mysql"
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

	// 创建路由
	r := gin.Default()

	// 创建自动CRUD处理器 - 方式1：显式指定表名
	userCrud1 := crud.New(db, &User{}, "users")

	// 创建自动CRUD处理器 - 方式2：使用空表名，自动获取
	userCrud2 := crud.New(db, &User{}, "")

	// 创建自动CRUD处理器 - 方式3：直接使用 New2
	userCrud3 := crud.New2(db, &User{})

	// 修改默认处理器的字段控制
	if listHandler, ok := userCrud1.GetHandler(crud.LIST); ok {
		listHandler.AllowedFields = []string{"id", "username", "email", "status", "created_at"} // 列表不返回密码
		userCrud1.AddHandler(crud.LIST, listHandler.Method, listHandler)
	}

	if detailHandler, ok := userCrud1.GetHandler(crud.SINGLE); ok {
		detailHandler.AllowedFields = []string{"id", "username", "email", "age", "status", "created_at", "updated_at"} // 详情不返回密码
		userCrud1.AddHandler(crud.SINGLE, detailHandler.Method, detailHandler)
	}

	if updateHandler, ok := userCrud1.GetHandler(crud.UPDATE); ok {
		updateHandler.AllowedFields = []string{"username", "email", "age", "status"} // 更新时不允许修改密码和时间戳
		userCrud1.AddHandler(crud.UPDATE, updateHandler.Method, updateHandler)
	}

	// 添加自定义处理器 - 修改密码
	passwordHandler := crud.NewHandler("/change-password", http.MethodPost).
		PreProcess(func(ctx *crud.ProcessContext) error {
			var req struct {
				ID          int64  `json:"id" binding:"required"`
				OldPassword string `json:"old_password" binding:"required"`
				NewPassword string `json:"new_password" binding:"required,min=6"`
			}
			if err := ctx.GinContext.ShouldBindJSON(&req); err != nil {
				return fmt.Errorf("invalid request: %v", err)
			}
			ctx.Data["request"] = req
			return nil
		}).
		BuildQuery(func(ctx *crud.ProcessContext) error {
			req := ctx.Data["request"].(struct {
				ID          int64
				OldPassword string
				NewPassword string
			})

			// 验证旧密码
			result := db.Chain().Table("users").
				Eq("id", req.ID).
				Eq("password", req.OldPassword).
				One()

			if result.Empty() {
				return fmt.Errorf("invalid old password")
			}

			// 准备更新语句
			ctx.Chain = db.Chain().Table("users").
				Eq("id", req.ID).
				Values(map[string]interface{}{
					"password": req.NewPassword,
				})
			return nil
		}).
		ExecuteStep(func(ctx *crud.ProcessContext) error {
			result, err := ctx.Chain.Save()
			if err != nil {
				return fmt.Errorf("failed to update password: %v", err)
			}
			ctx.Data["result"] = result
			return nil
		}).
		PostProcess(func(ctx *crud.ProcessContext) error {
			crud.CodeMsgFunc(ctx.GinContext, crud.CodeSuccess, "密码修改成功", nil)
			return nil
		})

	userCrud1.AddHandler("change_password", passwordHandler.Method, *passwordHandler)

	// 添加自定义处理器 - 批量更新状态
	batchStatusHandler := crud.NewHandler("/batch/status", http.MethodPost).
		PreProcess(func(ctx *crud.ProcessContext) error {
			var req struct {
				IDs    []int64 `json:"ids" binding:"required,min=1"`
				Status string  `json:"status" binding:"required,oneof=active inactive deleted"`
			}
			if err := ctx.GinContext.ShouldBindJSON(&req); err != nil {
				return fmt.Errorf("invalid request: %v", err)
			}
			ctx.Data["request"] = req
			return nil
		}).
		BuildQuery(func(ctx *crud.ProcessContext) error {
			req := ctx.Data["request"].(struct {
				IDs    []int64
				Status string
			})

			ctx.Chain = db.Chain().Table("users").
				In("id", req.IDs).
				Values(map[string]interface{}{
					"status": req.Status,
				})
			return nil
		}).
		ExecuteStep(func(ctx *crud.ProcessContext) error {
			result, err := ctx.Chain.Save()
			if err != nil {
				return fmt.Errorf("failed to update status: %v", err)
			}
			ctx.Data["result"] = result
			return nil
		}).
		PostProcess(func(ctx *crud.ProcessContext) error {
			crud.CodeMsgFunc(ctx.GinContext, crud.CodeSuccess, "状态更新成功", ctx.Data["result"])
			return nil
		})

	userCrud1.AddHandler("batch_update_status", batchStatusHandler.Method, *batchStatusHandler)

	// 注册路由 - 展示三种不同方式创建的处理器
	api := r.Group("/api")
	{
		userCrud1.Register(api.Group("/users/v1")) // 方式1：显式指定表名
		userCrud2.Register(api.Group("/users/v2")) // 方式2：使用空表名，自动获取
		userCrud3.Register(api.Group("/users/v3")) // 方式3：直接使用 New2
	}

	// 启动服务器
	if err := r.Run(":8080"); err != nil {
		log.Fatal(err)
	}
}
