package crud

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/kmlixh/gom/v4"
)

// 默认路由常量
const (
	LIST   = "list"   // 列表
	SINGLE = "single" // 详情
	SAVE   = "save"   // 保存
	UPDATE = "update" // 更新
	DELETE = "delete" // 删除
	PAGE   = "page"   // 分页
)

// ItemHandler 自定义处理器
type ItemHandler struct {
	Path          string          // 路由后缀
	Method        string          // HTTP 方法
	Handler       gin.HandlerFunc // 处理函数
	Conds         []QueryCondFunc // 查询条件函数
	AllowedFields []string        // 允许的字段列表（查询和更新都使用此列表）
}

// QueryCondFunc 查询条件函数
type QueryCondFunc func(*gom.Chain) *gom.Chain

// Crud 自动CRUD处理器
type Crud struct {
	db        *gom.DB
	entity    interface{}
	tableName string
	handlers  map[string]ItemHandler
}

// New 创建新的Crud实例
func New(db *gom.DB, entity interface{}, tableName string) *Crud {
	crud := &Crud{
		db:        db,
		entity:    entity,
		tableName: tableName,
		handlers:  make(map[string]ItemHandler),
	}

	// 注册默认处理器
	crud.AddHandler(LIST, http.MethodGet, ItemHandler{
		Path:    "/list",
		Method:  http.MethodGet,
		Handler: crud.List,
	})

	crud.AddHandler(PAGE, http.MethodGet, ItemHandler{
		Path:    "/page",
		Method:  http.MethodGet,
		Handler: crud.List,
	})

	crud.AddHandler(SINGLE, http.MethodGet, ItemHandler{
		Path:    "/detail/:id",
		Method:  http.MethodGet,
		Handler: crud.Get,
	})

	crud.AddHandler(SAVE, http.MethodPost, ItemHandler{
		Path:    "/save",
		Method:  http.MethodPost,
		Handler: crud.Create,
	})

	crud.AddHandler(UPDATE, http.MethodPost, ItemHandler{
		Path:    "/update/:id",
		Method:  http.MethodPut,
		Handler: crud.Update,
	})

	crud.AddHandler(DELETE, http.MethodGet, ItemHandler{
		Path:    "/delete/:id",
		Method:  http.MethodDelete,
		Handler: crud.Delete,
	})

	return crud
}

// GetHandler 获取指定名称的处理器
func (ac *Crud) GetHandler(name string) (ItemHandler, bool) {
	h, ok := ac.handlers[name]
	return h, ok
}

// AddHandler 添加自定义处理器
func (ac *Crud) AddHandler(name string, method string, handler ItemHandler) {
	ac.handlers[name] = handler
}

// filterFields 过滤字段（用于查询和更新）
func (ac *Crud) filterFields(h ItemHandler, data map[string]interface{}) map[string]interface{} {
	if len(h.AllowedFields) == 0 {
		return data
	}

	filtered := make(map[string]interface{})
	allowedMap := make(map[string]bool)
	for _, f := range h.AllowedFields {
		allowedMap[f] = true
	}

	for k, v := range data {
		if allowedMap[k] {
			filtered[k] = v
		}
	}

	return filtered
}

// List 列表查询
func (ac *Crud) List(c *gin.Context) {
	page, size := parsePagination(c)
	var chain *gom.Chain
	if v, exists := c.Get("chain"); exists {
		chain = v.(*gom.Chain)
	} else {
		chain = ac.db.Chain().Table(ac.tableName)
	}

	// 处理查询条件
	if err := parseQueryConditions(c, chain); err != nil {
		JsonErr(c, CodeInvalid, err.Error())
		return
	}

	// 获取当前处理器
	h := ac.handlers[LIST]
	if c.Query("page") != "" {
		h = ac.handlers[PAGE]
	}

	// 处理字段选择
	if len(h.AllowedFields) > 0 {
		chain = chain.Fields(h.AllowedFields...)
	}

	// 获取总数
	total, err := chain.Count()
	if err != nil {
		JsonErr(c, CodeError, err.Error())
		return
	}

	// 处理排序和分页
	chain = chain.Offset((page - 1) * size).Limit(size)
	if sort := c.Query("sort"); sort != "" {
		if sort[0] == '-' {
			chain = chain.OrderByDesc(sort[1:])
		} else {
			chain = chain.OrderBy(sort)
		}
	}

	// 执行查询
	result := chain.List()
	if err := result.Error(); err != nil {
		JsonErr(c, CodeError, err.Error())
		return
	}

	JsonOk(c, gin.H{
		"total":    total,
		"page":     page,
		"size":     size,
		"data":     result.Data,
		"lastPage": (page * size) >= int(total),
	})
}

// Create 创建记录
func (ac *Crud) Create(c *gin.Context) {
	data := make(map[string]interface{})
	if err := c.ShouldBindJSON(&data); err != nil {
		JsonErr(c, CodeInvalid, err.Error())
		return
	}

	// 过滤字段
	h := ac.handlers[SAVE]
	data = ac.filterFields(h, data)

	var chain *gom.Chain
	if v, exists := c.Get("chain"); exists {
		chain = v.(*gom.Chain)
	} else {
		chain = ac.db.Chain().Table(ac.tableName)
	}

	chain = chain.Values(data)
	result, err := chain.Save()
	if err != nil {
		JsonErr(c, CodeError, err.Error())
		return
	}

	JsonOk(c, result)
}

// Get 获取单条记录
func (ac *Crud) Get(c *gin.Context) {
	var chain *gom.Chain
	if v, exists := c.Get("chain"); exists {
		chain = v.(*gom.Chain)
	} else {
		chain = ac.db.Chain().Table(ac.tableName)
	}

	// 处理字段选择
	h := ac.handlers[SINGLE]
	if len(h.AllowedFields) > 0 {
		chain = chain.Fields(h.AllowedFields...)
	}

	chain = chain.Eq("id", c.Param("id"))
	result := chain.One()

	if err := result.Error(); err != nil {
		JsonErr(c, CodeError, "record not found")
		return
	}

	if result.Empty() {
		JsonErr(c, CodeInvalid, "record not found")
		return
	}

	JsonOk(c, result.Data[0])
}

// Update 更新记录
func (ac *Crud) Update(c *gin.Context) {
	data := make(map[string]interface{})
	if err := c.ShouldBindJSON(&data); err != nil {
		JsonErr(c, CodeInvalid, err.Error())
		return
	}

	// 过滤字段
	h := ac.handlers[UPDATE]
	data = ac.filterFields(h, data)

	var chain *gom.Chain
	if v, exists := c.Get("chain"); exists {
		chain = v.(*gom.Chain)
	} else {
		chain = ac.db.Chain().Table(ac.tableName)
	}

	chain = chain.Eq("id", c.Param("id")).Values(data)
	result, err := chain.Save()
	if err != nil {
		JsonErr(c, CodeError, err.Error())
		return
	}

	JsonOk(c, result)
}

// Delete 删除记录
func (ac *Crud) Delete(c *gin.Context) {
	var chain *gom.Chain
	if v, exists := c.Get("chain"); exists {
		chain = v.(*gom.Chain)
	} else {
		chain = ac.db.Chain().Table(ac.tableName)
	}

	chain = chain.Eq("id", c.Param("id"))
	result, err := chain.Delete()

	if err != nil {
		JsonErr(c, CodeError, err.Error())
		return
	}

	JsonOk(c, gin.H{
		"message": "deleted successfully",
		"result":  result,
	})
}

// 辅助函数
func parsePagination(c *gin.Context) (page, size int) {
	page, _ = strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ = strconv.Atoi(c.DefaultQuery("size", "10"))
	if page < 1 {
		page = 1
	}
	if size < 1 {
		size = 10
	}
	return
}

func parseQueryConditions(c *gin.Context, chain *gom.Chain) error {
	for key, values := range c.Request.URL.Query() {
		if len(values) == 0 || key == "page" || key == "size" || key == "sort" {
			continue
		}

		value := values[0]
		if value == "" {
			continue
		}

		// 支持操作符: eq, ne, gt, ge, lt, le, like, in
		// 格式: field__op=value
		parts := strings.Split(key, "__")
		field := parts[0]
		op := "eq"
		if len(parts) > 1 {
			op = parts[1]
		}

		switch op {
		case "eq":
			chain = chain.Eq(field, value)
		case "ne":
			chain = chain.Ne(field, value)
		case "gt":
			chain = chain.Gt(field, value)
		case "ge":
			chain = chain.Ge(field, value)
		case "lt":
			chain = chain.Lt(field, value)
		case "le":
			chain = chain.Le(field, value)
		case "like":
			chain = chain.Like(field, value)
		case "in":
			chain = chain.In(field, strings.Split(value, ","))
		default:
			return fmt.Errorf("unsupported operator: %s", op)
		}
	}
	return nil
}

// Register 注册所有CRUD路由
func (ac *Crud) Register(r gin.IRouter) {
	for _, h := range ac.handlers {
		switch h.Method {
		case http.MethodGet:
			r.GET(h.Path, h.Handler)
		case http.MethodPost:
			r.POST(h.Path, h.Handler)
		case http.MethodPut:
			r.PUT(h.Path, h.Handler)
		case http.MethodDelete:
			r.DELETE(h.Path, h.Handler)
		case http.MethodPatch:
			r.PATCH(h.Path, h.Handler)
		case "ANY", "*":
			r.Any(h.Path, h.Handler)
		}
	}
}
