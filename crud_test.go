package crud

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/kmlixh/gom/v4"
	_ "github.com/kmlixh/gom/v4/factory/mysql"
	"github.com/stretchr/testify/assert"
)

// TestUser 测试用户模型
type TestUser struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Age      int    `json:"age"`
	Status   string `json:"status"`
}

func TestApiInfo(t *testing.T) {
	// 设置测试模式
	gin.SetMode(gin.TestMode)

	// 创建路由
	r := gin.New()

	// 初始化数据库连接
	db, err := gom.Open("mysql", "root:password@tcp(localhost:3306)/test?charset=utf8mb4&parseTime=True&loc=Local", false)
	if err != nil {
		t.Fatal(err)
	}

	// 创建自动CRUD处理器
	userCrud := New2(db, &TestUser{}).
		SetDescription("用户管理模块，提供用户的增删改查功能")

	// 修改默认处理器的字段控制
	if listHandler, ok := userCrud.GetHandler(LIST); ok {
		listHandler.AllowedFields = []string{"id", "username", "email", "status"}
		listHandler.SetDescription("获取用户列表，支持分页和排序")
		userCrud.AddHandler(LIST, listHandler.Method, listHandler)
	}

	if detailHandler, ok := userCrud.GetHandler(SINGLE); ok {
		detailHandler.AllowedFields = []string{"id", "username", "email", "age", "status"}
		detailHandler.SetDescription("获取用户详细信息")
		userCrud.AddHandler(SINGLE, detailHandler.Method, detailHandler)
	}

	// 添加自定义处理器
	activeUsersHandler := NewHandler("/active", http.MethodGet).
		SetDescription("获取所有活跃状态的用户列表").
		PreProcess(func(ctx *ProcessContext) error {
			return nil
		}).
		BuildQuery(func(ctx *ProcessContext) error {
			ctx.Chain = db.Chain().Table("users").
				Eq("status", "active")
			return nil
		}).
		ExecuteStep(func(ctx *ProcessContext) error {
			result := ctx.Chain.List()
			if err := result.Error(); err != nil {
				return err
			}
			ctx.Data["result"] = result.Data
			return nil
		}).
		PostProcess(func(ctx *ProcessContext) error {
			CodeMsgFunc(ctx.GinContext, CodeSuccess, "success", ctx.Data["result"])
			return nil
		})

	userCrud.AddHandler("active_users", activeUsersHandler.Method, activeUsersHandler)

	// 注册用户相关路由
	userCrud.RegisterRoutes(r.Group("/api"), "/users")

	// 注册API信息路由
	RegisterApi(r, "/api-info")

	// 创建测试请求
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api-info", nil)
	r.ServeHTTP(w, req)

	// 检查响应状态码
	assert.Equal(t, http.StatusOK, w.Code)

	// 解析响应内容
	var response struct {
		Code    int         `json:"code"`
		Message string      `json:"message"`
		Data    []ModelInfo `json:"data"`
	}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	// 检查响应内容
	assert.Equal(t, CodeSuccess, response.Code)
	assert.Equal(t, MsgSuccess, response.Message)
	assert.NotEmpty(t, response.Data)

	// 检查模型信息
	found := false
	for _, model := range response.Data {
		if model.Name == "TestUser" {
			found = true
			assert.Equal(t, "用户管理模块，提供用户的增删改查功能", model.Description)
			assert.NotEmpty(t, model.Handlers)

			// 检查处理器信息
			for _, handler := range model.Handlers {
				switch handler.Operations {
				case "LIST":
					assert.Equal(t, "获取用户列表，支持分页和排序", handler.Description)
					assert.Equal(t, []string{"id", "username", "email", "status"}, handler.Fields)
				case "SINGLE":
					assert.Equal(t, "获取用户详细信息", handler.Description)
					assert.Equal(t, []string{"id", "username", "email", "age", "status"}, handler.Fields)
				case "active_users":
					assert.Equal(t, "获取所有活跃状态的用户列表", handler.Description)
				}
			}
			break
		}
	}
	assert.True(t, found, "TestUser model not found in API info")
}

func TestSetDescription(t *testing.T) {
	// 创建 Crud 实例
	crud := &Crud{}
	desc := "Test Description"

	// 测试设置描述
	result := crud.SetDescription(desc)
	assert.Equal(t, desc, crud.description)
	assert.Equal(t, crud, result)

	// 创建 ItemHandler 实例
	handler := &ItemHandler{}
	handlerDesc := "Handler Description"

	// 测试设置处理器描述
	result2 := handler.SetDescription(handlerDesc)
	assert.Equal(t, handlerDesc, handler.description)
	assert.Equal(t, handler, result2)
}
