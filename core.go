package crud

import (
	"github.com/gin-gonic/gin"
	"github.com/kmlixh/gom/v4"
)

// Handler 处理器接口
type Handler interface {
	// List 列表查询
	List(c *gin.Context)
	// Get 获取单条记录
	Get(c *gin.Context)
	// Save 创建或更新记录
	Save(c *gin.Context)
	// Update 更新记录
	Update(c *gin.Context)
	// Delete 删除记录
	Delete(c *gin.Context)
}

// Options CRUD 选项
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

// Response API 响应
type Response struct {
	Code    int         `json:"code"`    // 状态码
	Message string      `json:"message"` // 消息
	Data    interface{} `json:"data"`    // 数据
}

// PageResult 分页结果
type PageResult struct {
	PageNum    int64       `json:"pageNum"`    // 当前页码
	PageSize   int64       `json:"pageSize"`   // 每页大小
	Total      int64       `json:"total"`      // 总记录数
	TotalPages int64       `json:"totalPages"` // 总页数
	Data       interface{} `json:"data"`       // 数据列表
}

// success 返回成功响应
func success(data interface{}) Response {
	return Response{
		Code: 0,
		Data: data,
	}
}

// fail 返回失败响应
func fail(err error) Response {
	return Response{
		Code:    500,
		Message: err.Error(),
	}
}

// Register 注册 CRUD 路由
func Register(router gin.IRouter, db *gom.DB, model interface{}, opts Options) {
	// 创建处理器
	h := newHandler(db, model, opts)

	// 注册路由
	group := router.Group(opts.PathPrefix)
	{
		// 查询和删除操作使用 GET
		group.GET("", h.List)
		group.GET("/:id", h.Get)
		group.GET("/delete/:id", h.Delete)

		// 其他操作使用 POST
		group.POST("/save", h.Save)         // 创建/更新
		group.POST("/update/:id", h.Update) // 更新
	}
}
