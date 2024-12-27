package main

import (
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

// CodeMessage 响应码和消息
type CodeMessage struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// CustomResponse 自定义响应结构
type CustomResponse struct {
	CodeMessage
	Data interface{} `json:"data"`
}

// NewResponse 创建新的响应
func NewResponse(code int, message string, data interface{}) CustomResponse {
	return CustomResponse{
		CodeMessage: CodeMessage{
			Code:    code,
			Message: message,
		},
		Data: data,
	}
}

func main() {
	// 初始化数据库连接
	db, err := gom.Open("mysql", "root:password@tcp(localhost:3306)/test?charset=utf8mb4&parseTime=True&loc=Local", false)
	if err != nil {
		log.Fatal(err)
	}

	// 创建路由
	r := gin.Default()

	// 创建自动CRUD处理器
	userCrud := crud.New(db, &User{}, "users")

	// 修改默认处理器的字段控制
	if listHandler, ok := userCrud.GetHandler(crud.LIST); ok {
		listHandler.AllowedFields = []string{"id", "username", "email", "status", "created_at"} // 列表不返回密码
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

	// 添加自定义处理器 - 获取活跃用户
	userCrud.AddHandler("active_users", http.MethodGet, crud.ItemHandler{
		Path:          "/active",
		Method:        http.MethodGet,
		AllowedFields: []string{"id", "username", "status"}, // 只返回必要字段
		Handler: func(c *gin.Context) {
			chain := db.Chain().Table("users").Eq("status", "active")
			result := chain.List()
			if err := result.Error(); err != nil {
				crud.JsonErr(c, crud.CodeError, err.Error())
				return
			}
			crud.JsonOk(c, result.Data)
		},
	})

	// 添加自定义处理器 - 批量更新状态
	userCrud.AddHandler("batch_update_status", http.MethodPost, crud.ItemHandler{
		Path:          "/batch/status",
		Method:        http.MethodPost,
		AllowedFields: []string{"status"}, // 只允许更新状态字段
		Handler: func(c *gin.Context) {
			var req struct {
				IDs    []int64 `json:"ids"`
				Status string  `json:"status"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				crud.JsonErr(c, crud.CodeInvalid, err.Error())
				return
			}

			chain := db.Chain().Table("users").In("id", req.IDs).Values(map[string]interface{}{
				"status": req.Status,
			})
			result, err := chain.Save()
			if err != nil {
				crud.JsonErr(c, crud.CodeError, err.Error())
				return
			}

			crud.JsonOk(c, result)
		},
	})

	// 添加自定义处理器 - 修改密码
	userCrud.AddHandler("change_password", http.MethodPost, crud.ItemHandler{
		Path:          "/change-password",
		Method:        http.MethodPost,
		AllowedFields: []string{"password"}, // 只允许更新密码字段
		Handler: func(c *gin.Context) {
			var req struct {
				ID          int64  `json:"id"`
				OldPassword string `json:"old_password"`
				NewPassword string `json:"new_password"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				crud.JsonErr(c, crud.CodeInvalid, err.Error())
				return
			}

			// 验证旧密码
			result := db.Chain().Table("users").
				Eq("id", req.ID).
				Eq("password", req.OldPassword).
				One()

			if result.Empty() {
				crud.JsonErr(c, crud.CodeInvalid, "invalid old password")
				return
			}

			// 更新新密码
			chain := db.Chain().Table("users").
				Eq("id", req.ID).
				Values(map[string]interface{}{
					"password": req.NewPassword,
				})
			result2, err := chain.Save()
			if err != nil {
				crud.JsonErr(c, crud.CodeError, err.Error())
				return
			}

			crud.JsonOk(c, result2)
		},
	})

	// 注册路由
	api := r.Group("/api")
	{
		userCrud.Register(api.Group("/users"))
	}

	// 启动服务器
	if err := r.Run(":8080"); err != nil {
		log.Fatal(err)
	}
}
