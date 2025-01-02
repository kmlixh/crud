package crud

import (
	"fmt"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"text/template"

	"github.com/gin-gonic/gin"
	"github.com/kmlixh/gom/v4"
	"github.com/kmlixh/gom/v4/define"
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

// ProcessStep 处理步骤
type ProcessStep int

const (
	PreProcess  ProcessStep = iota // 预处理：参数验证、权限检查等
	BuildQuery                     // 构建查询：构建数据库操作语句
	Execute                        // 执行：执行数据库操作
	PostProcess                    // 后处理：响应处理
)

// StepPhase 步骤阶段
type StepPhase int

const (
	BeforePhase StepPhase = iota // 执行前
	OnPhase                      // 执行中
	AfterPhase                   // 执行后
)

// ProcessContext 处理上下文
type ProcessContext struct {
	GinContext *gin.Context     // Gin上下文
	Chain      *gom.Chain       // 数据库链
	Data       map[string]any   // 数据
	Result     *gom.QueryResult // 查询结果
	Error      error            // 错误信息
}

// ProcessHandler 处理器函数
type ProcessHandler func(*ProcessContext) error

// QueryCondFunc 查询条件函数
type QueryCondFunc func(*gom.Chain) *gom.Chain

// ItemHandler 自定义处理器
type ItemHandler struct {
	Path          string                                       // 路由后缀
	Method        string                                       // HTTP 方法
	Handler       gin.HandlerFunc                              // 处理函数
	Conds         []QueryCondFunc                              // 查询条件函数
	AllowedFields []string                                     // 允许的字段列表
	Orders        []*define.OrderBy                            // 排序配置
	description   string                                       // 处理器描述
	Processors    map[ProcessStep]map[StepPhase]ProcessHandler // 处理器映射
}

// NewHandler 创建新的处理器
func NewHandler(path string, method string) *ItemHandler {
	h := &ItemHandler{
		Path:   path,
		Method: method,
	}
	h.Handler = h.HandleRequest
	return h
}

// PreProcess 添加预处理器
func (h *ItemHandler) PreProcess(handler ProcessHandler) *ItemHandler {
	h.AddProcessor(PreProcess, OnPhase, handler)
	return h
}

// PreProcessBefore 添加预处理前置处理器
func (h *ItemHandler) PreProcessBefore(handler ProcessHandler) *ItemHandler {
	h.AddProcessor(PreProcess, BeforePhase, handler)
	return h
}

// PreProcessAfter 添加预处理后置处理器
func (h *ItemHandler) PreProcessAfter(handler ProcessHandler) *ItemHandler {
	h.AddProcessor(PreProcess, AfterPhase, handler)
	return h
}

// BuildQuery 添加查询构建器
func (h *ItemHandler) BuildQuery(handler ProcessHandler) *ItemHandler {
	h.AddProcessor(BuildQuery, OnPhase, handler)
	return h
}

// BuildQueryBefore 添加查询构建前置处理器
func (h *ItemHandler) BuildQueryBefore(handler ProcessHandler) *ItemHandler {
	h.AddProcessor(BuildQuery, BeforePhase, handler)
	return h
}

// BuildQueryAfter 添加查询构建后置处理器
func (h *ItemHandler) BuildQueryAfter(handler ProcessHandler) *ItemHandler {
	h.AddProcessor(BuildQuery, AfterPhase, handler)
	return h
}

// ExecuteStep 添加执行器
func (h *ItemHandler) ExecuteStep(handler ProcessHandler) *ItemHandler {
	h.AddProcessor(Execute, OnPhase, handler)
	return h
}

// ExecuteStepBefore 添加执行前置处理器
func (h *ItemHandler) ExecuteStepBefore(handler ProcessHandler) *ItemHandler {
	h.AddProcessor(Execute, BeforePhase, handler)
	return h
}

// ExecuteStepAfter 添加执行后置处理器
func (h *ItemHandler) ExecuteStepAfter(handler ProcessHandler) *ItemHandler {
	h.AddProcessor(Execute, AfterPhase, handler)
	return h
}

// PostProcess 添加后处理器
func (h *ItemHandler) PostProcess(handler ProcessHandler) *ItemHandler {
	h.AddProcessor(PostProcess, OnPhase, handler)
	return h
}

// PostProcessBefore 添加后处理前置处理器
func (h *ItemHandler) PostProcessBefore(handler ProcessHandler) *ItemHandler {
	h.AddProcessor(PostProcess, BeforePhase, handler)
	return h
}

// PostProcessAfter 添加后处理后置处理器
func (h *ItemHandler) PostProcessAfter(handler ProcessHandler) *ItemHandler {
	h.AddProcessor(PostProcess, AfterPhase, handler)
	return h
}

// HandleRequest 处理请求
func (h *ItemHandler) HandleRequest(c *gin.Context) {
	ctx := &ProcessContext{
		GinContext: c,
		Data:       make(map[string]any),
	}

	// 执行所有步骤
	steps := []ProcessStep{
		PreProcess,  // 预处理：参数验证、权限检查等
		BuildQuery,  // 构建查询：构建数据库操作语句
		Execute,     // 执行：执行数据库操作
		PostProcess, // 后处理：响应处理
	}

	for _, step := range steps {
		if err := h.processStep(ctx, step); err != nil {
			// 如果是最后一步，让它自己处理错误
			if step == PostProcess {
				return
			}
			// 否则使用默认错误处理
			JsonErr(c, CodeError, err.Error())
			return
		}
	}
}

// processStep 执行单个步骤
func (h *ItemHandler) processStep(ctx *ProcessContext, step ProcessStep) error {
	// 执行前处理
	if handler := h.GetProcessor(step, BeforePhase); handler != nil {
		if err := handler(ctx); err != nil {
			return err
		}
	}

	// 执行处理
	if handler := h.GetProcessor(step, OnPhase); handler != nil {
		if err := handler(ctx); err != nil {
			return err
		}
	}

	// 如果是构建查询步骤，处理排序
	if step == BuildQuery && len(h.Orders) > 0 {
		applyOrderBy(ctx.Chain, h.Orders)
	}

	// 执行后处理
	if handler := h.GetProcessor(step, AfterPhase); handler != nil {
		if err := handler(ctx); err != nil {
			return err
		}
	}

	return nil
}

// AddProcessor 添加处理器
func (h *ItemHandler) AddProcessor(step ProcessStep, phase StepPhase, handler ProcessHandler) {
	if h.Processors == nil {
		h.Processors = make(map[ProcessStep]map[StepPhase]ProcessHandler)
	}
	if h.Processors[step] == nil {
		h.Processors[step] = make(map[StepPhase]ProcessHandler)
	}
	h.Processors[step][phase] = handler
}

// GetProcessor 获取处理器
func (h *ItemHandler) GetProcessor(step ProcessStep, phase StepPhase) ProcessHandler {
	if h.Processors == nil {
		return nil
	}
	if h.Processors[step] == nil {
		return nil
	}
	return h.Processors[step][phase]
}

// Crud 自动CRUD处理器
type Crud struct {
	db          *gom.DB                 // 数据库连接
	entity      interface{}             // 实体对象
	tableName   string                  // 表名
	handlers    map[string]*ItemHandler // 处理器映射
	description string                  // 实体描述
}

// DefaultProcessors 默认处理器集合
type DefaultProcessors struct {
	crud *Crud
}

// NewDefaultProcessors 创建默认处理器集合
func NewDefaultProcessors(crud *Crud) *DefaultProcessors {
	return &DefaultProcessors{crud: crud}
}

// List 处理器
func (dp *DefaultProcessors) List() *ItemHandler {
	h := &ItemHandler{
		Path:   "/list",
		Method: http.MethodGet,
	}

	// 构建对象
	h.AddProcessor(BuildQuery, BeforePhase, func(ctx *ProcessContext) error {
		if v, exists := ctx.GinContext.Get("chain"); exists {
			ctx.Chain = v.(*gom.Chain)
		} else {
			ctx.Chain = dp.crud.db.Chain().Table(dp.crud.tableName)
		}
		return nil
	})

	// 构建SQL
	h.AddProcessor(BuildQuery, OnPhase, func(ctx *ProcessContext) error {
		// 处理查询条件
		if err := parseQueryConditions(ctx.GinContext, ctx.Chain); err != nil {
			return err
		}

		// 处理字段选择
		if len(h.AllowedFields) > 0 {
			ctx.Chain = ctx.Chain.Fields(h.AllowedFields...)
		}

		// 处理排序
		if sort := ctx.GinContext.Query("sort"); sort != "" {
			if sort[0] == '-' {
				ctx.Chain = ctx.Chain.OrderByDesc(sort[1:])
			} else {
				ctx.Chain = ctx.Chain.OrderBy(sort)
			}
		}
		return nil
	})

	// 执行SQL
	h.AddProcessor(Execute, OnPhase, func(ctx *ProcessContext) error {
		// 执行查询
		ctx.Result = ctx.Chain.List()
		return ctx.Result.Error()
	})

	// 处理响应
	h.AddProcessor(PostProcess, OnPhase, func(ctx *ProcessContext) error {
		JsonOk(ctx.GinContext, gin.H{
			"data": ctx.Result.Data,
		})
		return nil
	})

	return h
}

// Get 处理器
func (dp *DefaultProcessors) Get() *ItemHandler {
	h := &ItemHandler{
		Path:   "/detail/:id",
		Method: http.MethodGet,
	}

	// 构建对象
	h.AddProcessor(BuildQuery, OnPhase, func(ctx *ProcessContext) error {
		if v, exists := ctx.GinContext.Get("chain"); exists {
			ctx.Chain = v.(*gom.Chain)
		} else {
			ctx.Chain = dp.crud.db.Chain().Table(dp.crud.tableName)
		}
		return nil
	})

	// 构建SQL
	h.AddProcessor(BuildQuery, OnPhase, func(ctx *ProcessContext) error {
		if len(h.AllowedFields) > 0 {
			ctx.Chain = ctx.Chain.Fields(h.AllowedFields...)
		}
		ctx.Chain = ctx.Chain.Eq("id", ctx.GinContext.Param("id"))
		return nil
	})

	// 执行SQL
	h.AddProcessor(Execute, OnPhase, func(ctx *ProcessContext) error {
		ctx.Result = ctx.Chain.One()
		return ctx.Result.Error()
	})

	// 处理响应
	h.AddProcessor(PostProcess, OnPhase, func(ctx *ProcessContext) error {
		if ctx.Result.Empty() {
			JsonErr(ctx.GinContext, CodeInvalid, "record not found")
			return nil
		}
		JsonOk(ctx.GinContext, ctx.Result.Data[0])
		return nil
	})

	return h
}

// Create 处理器
func (dp *DefaultProcessors) Create() *ItemHandler {
	h := &ItemHandler{
		Path:   "/save",
		Method: http.MethodPost,
	}

	// 预处理
	h.AddProcessor(PreProcess, OnPhase, func(ctx *ProcessContext) error {
		data := make(map[string]interface{})
		if err := ctx.GinContext.ShouldBindJSON(&data); err != nil {
			return err
		}
		ctx.Data["input"] = data
		return nil
	})

	// 构建对象
	h.AddProcessor(BuildQuery, OnPhase, func(ctx *ProcessContext) error {
		if v, exists := ctx.GinContext.Get("chain"); exists {
			ctx.Chain = v.(*gom.Chain)
		} else {
			ctx.Chain = dp.crud.db.Chain().Table(dp.crud.tableName)
		}
		return nil
	})

	// 构建SQL
	h.AddProcessor(BuildQuery, OnPhase, func(ctx *ProcessContext) error {
		data := ctx.Data["input"].(map[string]interface{})
		if len(h.AllowedFields) > 0 {
			filtered := make(map[string]interface{})
			for _, field := range h.AllowedFields {
				if value, ok := data[field]; ok {
					filtered[field] = value
				}
			}
			data = filtered
		}
		ctx.Chain = ctx.Chain.Values(data)
		return nil
	})

	// 执行SQL
	h.AddProcessor(Execute, OnPhase, func(ctx *ProcessContext) error {
		result, err := ctx.Chain.Save()
		if err != nil {
			return err
		}
		ctx.Data["result"] = result
		return nil
	})

	// 处理响应
	h.AddProcessor(PostProcess, OnPhase, func(ctx *ProcessContext) error {
		JsonOk(ctx.GinContext, ctx.Data["result"])
		return nil
	})

	return h
}

// Update 处理器
func (dp *DefaultProcessors) Update() *ItemHandler {
	h := &ItemHandler{
		Path:   "/update/:id",
		Method: http.MethodPut,
	}

	// 预处理
	h.AddProcessor(PreProcess, OnPhase, func(ctx *ProcessContext) error {
		data := make(map[string]interface{})
		if err := ctx.GinContext.ShouldBindJSON(&data); err != nil {
			return err
		}
		ctx.Data["input"] = data
		return nil
	})

	// 构建对象
	h.AddProcessor(BuildQuery, OnPhase, func(ctx *ProcessContext) error {
		if v, exists := ctx.GinContext.Get("chain"); exists {
			ctx.Chain = v.(*gom.Chain)
		} else {
			ctx.Chain = dp.crud.db.Chain().Table(dp.crud.tableName)
		}
		return nil
	})

	// 构建SQL
	h.AddProcessor(BuildQuery, OnPhase, func(ctx *ProcessContext) error {
		data := ctx.Data["input"].(map[string]interface{})
		if len(h.AllowedFields) > 0 {
			filtered := make(map[string]interface{})
			for _, field := range h.AllowedFields {
				if value, ok := data[field]; ok {
					filtered[field] = value
				}
			}
			data = filtered
		}
		ctx.Chain = ctx.Chain.Eq("id", ctx.GinContext.Param("id")).Values(data)
		return nil
	})

	// 执行SQL
	h.AddProcessor(Execute, OnPhase, func(ctx *ProcessContext) error {
		result, err := ctx.Chain.Save()
		if err != nil {
			return err
		}
		ctx.Data["result"] = result
		return nil
	})

	// 处理响应
	h.AddProcessor(PostProcess, OnPhase, func(ctx *ProcessContext) error {
		JsonOk(ctx.GinContext, ctx.Data["result"])
		return nil
	})

	return h
}

// Delete 处理器
func (dp *DefaultProcessors) Delete() *ItemHandler {
	h := &ItemHandler{
		Path:   "/delete/:id",
		Method: http.MethodDelete,
	}

	// 构建对象
	h.AddProcessor(BuildQuery, OnPhase, func(ctx *ProcessContext) error {
		if v, exists := ctx.GinContext.Get("chain"); exists {
			ctx.Chain = v.(*gom.Chain)
		} else {
			ctx.Chain = dp.crud.db.Chain().Table(dp.crud.tableName)
		}
		return nil
	})

	// 构建SQL
	h.AddProcessor(BuildQuery, OnPhase, func(ctx *ProcessContext) error {
		ctx.Chain = ctx.Chain.Eq("id", ctx.GinContext.Param("id"))
		return nil
	})

	// 执行SQL
	h.AddProcessor(Execute, OnPhase, func(ctx *ProcessContext) error {
		result, err := ctx.Chain.Delete()
		if err != nil {
			return err
		}
		ctx.Data["result"] = result
		return nil
	})

	// 处理响应
	h.AddProcessor(PostProcess, OnPhase, func(ctx *ProcessContext) error {
		JsonOk(ctx.GinContext, gin.H{
			"message": "deleted successfully",
			"result":  ctx.Data["result"],
		})
		return nil
	})

	return h
}

// Page 处理器
func (dp *DefaultProcessors) Page() *ItemHandler {
	h := &ItemHandler{
		Path:   "/page",
		Method: http.MethodGet,
	}
	h.Handler = h.HandleRequest

	// 预处理
	h.AddProcessor(PreProcess, OnPhase, func(ctx *ProcessContext) error {
		page, size := parsePagination(ctx.GinContext)
		ctx.Data["page"] = page
		ctx.Data["size"] = size
		return nil
	})

	// 构建对象
	h.AddProcessor(BuildQuery, BeforePhase, func(ctx *ProcessContext) error {
		if v, exists := ctx.GinContext.Get("chain"); exists {
			ctx.Chain = v.(*gom.Chain)
		} else {
			if dp.crud.tableName == "" {
				return fmt.Errorf("table name is not set")
			}
			ctx.Chain = dp.crud.db.Chain().Table(dp.crud.tableName)
		}
		return nil
	})

	// 构建SQL
	h.AddProcessor(BuildQuery, OnPhase, func(ctx *ProcessContext) error {
		// 处理查询条件
		if err := parseQueryConditions(ctx.GinContext, ctx.Chain); err != nil {
			return err
		}

		// 处理字段选择
		if len(h.AllowedFields) > 0 {
			ctx.Chain = ctx.Chain.Fields(h.AllowedFields...)
		}

		// 处理排序和分页
		page := ctx.Data["page"].(int)
		size := ctx.Data["size"].(int)
		ctx.Chain = ctx.Chain.Offset((page - 1) * size).Limit(size)

		if sort := ctx.GinContext.Query("sort"); sort != "" {
			if sort[0] == '-' {
				ctx.Chain = ctx.Chain.OrderByDesc(sort[1:])
			} else {
				ctx.Chain = ctx.Chain.OrderBy(sort)
			}
		}
		return nil
	})

	// 执行SQL
	h.AddProcessor(Execute, OnPhase, func(ctx *ProcessContext) error {
		// 获取总数
		total, err := ctx.Chain.Count()
		if err != nil {
			return err
		}
		ctx.Data["total"] = total

		// 执行查询
		ctx.Result = ctx.Chain.List()
		return ctx.Result.Error()
	})

	// 处理响应
	h.AddProcessor(PostProcess, OnPhase, func(ctx *ProcessContext) error {
		page := ctx.Data["page"].(int)
		size := ctx.Data["size"].(int)
		total := ctx.Data["total"].(int64)

		JsonOk(ctx.GinContext, gin.H{
			"total":    total,
			"page":     page,
			"size":     size,
			"data":     ctx.Result.Data,
			"lastPage": (page * size) >= int(total),
		})
		return nil
	})

	return h
}

// New2 创建新的CRUD处理器（自动获取表名）
func New2(db *gom.DB, entity interface{}) *Crud {
	tableName, err := db.GetTableName(entity)
	if err != nil {
		// 如果获取表名失败，使用结构体名称的小写形式作为表名
		t := reflect.TypeOf(entity)
		if t.Kind() == reflect.Ptr {
			t = t.Elem()
		}
		tableName = strings.ToLower(t.Name())
	}

	crud := &Crud{
		db:        db,
		entity:    entity,
		tableName: tableName,
		handlers:  make(map[string]*ItemHandler),
	}

	// 初始化默认处理器
	crud.initDefaultHandlers()
	cruds = append(cruds, crud)
	return crud
}

// New 创建新的CRUD处理器
func New(db *gom.DB, entity interface{}, tableName string) *Crud {
	crud := &Crud{
		db:        db,
		entity:    entity,
		tableName: tableName,
		handlers:  make(map[string]*ItemHandler),
	}

	// 初始化默认处理器
	crud.initDefaultHandlers()
	cruds = append(cruds, crud)
	return crud
}

// initDefaultHandlers 初始化默认处理器
func (c *Crud) initDefaultHandlers() {
	processors := NewDefaultProcessors(c)

	// 列表处理器
	c.handlers[LIST] = processors.List()

	// 分页处理器
	c.handlers[PAGE] = processors.Page()

	// 详情处理器
	c.handlers[SINGLE] = processors.Get()

	// 创建处理器
	c.handlers[SAVE] = processors.Create()

	// 更新处理器
	c.handlers[UPDATE] = processors.Update()

	// 删除处理器
	c.handlers[DELETE] = processors.Delete()
}

// GetHandler 获取指定名称的处理器
func (ac *Crud) GetHandler(name string) (*ItemHandler, bool) {
	handler, ok := ac.handlers[name]
	if !ok {
		return nil, false
	}
	return handler, true
}

// AddHandler 添加自定义处理器
func (ac *Crud) AddHandler(name string, method string, handler *ItemHandler) {
	ac.handlers[name] = handler
}

// filterFields 过滤字段
func (c *Crud) filterFields(h *ItemHandler, data interface{}) map[string]interface{} {
	if len(h.AllowedFields) == 0 {
		if m, ok := data.(map[string]interface{}); ok {
			return m
		}
		return nil
	}

	allowedMap := make(map[string]bool)
	for _, field := range h.AllowedFields {
		allowedMap[field] = true
	}

	switch v := data.(type) {
	case map[string]interface{}:
		result := make(map[string]interface{})
		for key, value := range v {
			if allowedMap[key] {
				result[key] = value
			}
		}
		return result
	case []map[string]interface{}:
		// 如果是数组，只返回第一个元素
		if len(v) > 0 {
			result := make(map[string]interface{})
			for key, value := range v[0] {
				if allowedMap[key] {
					result[key] = value
				}
			}
			return result
		}
		return make(map[string]interface{})
	default:
		return make(map[string]interface{})
	}
}

// List 列表查询
func (ac *Crud) List(c *gin.Context) {
	// 如果请求包含分页参数，重定向到分页处理器
	if c.Query("page") != "" || c.Query("size") != "" {
		if handler, ok := ac.handlers[PAGE]; ok {
			handler.Handler(c)
			return
		}
	}

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

	// 处理字段选择
	if len(h.AllowedFields) > 0 {
		chain = chain.Fields(h.AllowedFields...)
	}

	// 处理排序
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
		"data": result.Data,
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

// validateOrderFields 验证排序字段是否合法
func (h *ItemHandler) validateOrderFields(orders []*define.OrderBy) error {
	if len(h.AllowedFields) == 0 {
		return nil
	}

	allowedMap := make(map[string]bool)
	for _, field := range h.AllowedFields {
		allowedMap[field] = true
	}

	for _, order := range orders {
		if !allowedMap[order.Field] {
			return fmt.Errorf("invalid order field: %s", order.Field)
		}
	}

	return nil
}

// applyOrderBy 应用排序
func applyOrderBy(chain *gom.Chain, orders []*define.OrderBy) *gom.Chain {
	if len(orders) == 0 {
		return chain
	}

	for _, order := range orders {
		if order.Type == define.OrderDesc {
			chain = chain.OrderByDesc(order.Field)
		} else {
			chain = chain.OrderBy(order.Field)
		}
	}

	return chain
}

// SetOrders 设置排序
func (h *ItemHandler) SetOrders(orders ...*define.OrderBy) *ItemHandler {
	h.Orders = orders
	return h
}

// 全局路由信息
var (
	registeredHandlers = make(map[string][]HandlerInfo)
)

// HandlerInfo 处理器信息
type HandlerInfo struct {
	Path        string   `json:"path"`        // 完整路径
	Method      string   `json:"method"`      // HTTP方法
	Model       string   `json:"model"`       // 模型名称
	Fields      []string `json:"fields"`      // 允许的字段
	Operations  string   `json:"operations"`  // 操作类型
	Description string   `json:"description"` // 处理器描述
}

// ModelInfo 模型信息
type ModelInfo struct {
	Name        string        `json:"name"`        // 模型名称
	Description string        `json:"description"` // 模型描述
	Handlers    []HandlerInfo `json:"handlers"`    // 处理器列表
}

// RegisterHandler 注册处理器信息
func (c *Crud) RegisterHandler(modelName string, info HandlerInfo) {
	if _, ok := registeredHandlers[modelName]; !ok {
		registeredHandlers[modelName] = make([]HandlerInfo, 0)
	}
	registeredHandlers[modelName] = append(registeredHandlers[modelName], info)
}

// RegisterRoutes 注册路由并记录信息
func (c *Crud) RegisterRoutes(group *gin.RouterGroup, path string) {
	modelName := reflect.TypeOf(c.entity).Elem().Name()
	basePath := path

	// 注册默认处理器
	for name, handler := range c.handlers {
		fullPath := basePath + handler.Path
		group.Handle(handler.Method, fullPath, handler.Handler)

		// 记录路由信息
		info := HandlerInfo{
			Path:        fullPath,
			Method:      handler.Method,
			Model:       modelName,
			Fields:      handler.AllowedFields,
			Operations:  name,
			Description: handler.description,
		}
		c.RegisterHandler(modelName, info)
	}
}

// RegisterApi 注册API信息路由
func RegisterApi(router gin.IRouter, path string) {
	router.GET(path, func(c *gin.Context) {
		// 将 map 转换为 slice，以便添加模型描述
		models := make([]ModelInfo, 0)
		for modelName, handlers := range registeredHandlers {
			// 查找第一个处理器的 Crud 实例以获取描述
			var description string
			for _, crud := range cruds {
				if reflect.TypeOf(crud.entity).Elem().Name() == modelName {
					description = crud.description
					break
				}
			}
			models = append(models, ModelInfo{
				Name:        modelName,
				Description: description,
				Handlers:    handlers,
			})
		}
		JsonOk(c, models)
	})
}

// GetRegisteredHandlers 获取所有注册的处理器信息
func GetRegisteredHandlers() map[string][]HandlerInfo {
	return registeredHandlers
}

// SetDescription 设置实体描述
func (c *Crud) SetDescription(desc string) *Crud {
	c.description = desc
	return c
}

// SetDescription 设置处理器描述
func (h *ItemHandler) SetDescription(desc string) *ItemHandler {
	h.description = desc
	return h
}

// 存储所有的 Crud 实例
var cruds []*Crud

// apiDocTemplate HTML文档模板
const apiDocTemplate = `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>API Documentation</title>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif;
            line-height: 1.6;
            color: #333;
            max-width: 1200px;
            margin: 0 auto;
            padding: 20px;
        }
        .model {
            background: #f8f9fa;
            border-radius: 5px;
            padding: 20px;
            margin-bottom: 30px;
        }
        .model-name {
            font-size: 24px;
            color: #2c3e50;
            margin-bottom: 10px;
        }
        .model-description {
            color: #666;
            margin-bottom: 20px;
        }
        .handler {
            background: white;
            border: 1px solid #e9ecef;
            border-radius: 4px;
            padding: 15px;
            margin-bottom: 15px;
        }
        .method {
            display: inline-block;
            padding: 4px 8px;
            border-radius: 3px;
            font-size: 12px;
            font-weight: bold;
            margin-right: 10px;
        }
        .get { background: #61affe; color: white; }
        .post { background: #49cc90; color: white; }
        .put { background: #fca130; color: white; }
        .delete { background: #f93e3e; color: white; }
        .path {
            font-family: monospace;
            font-size: 14px;
            color: #3b4151;
        }
        .fields {
            margin-top: 10px;
            color: #666;
        }
        .fields span {
            display: inline-block;
            background: #e9ecef;
            padding: 2px 6px;
            border-radius: 3px;
            margin: 2px;
            font-size: 12px;
        }
        .description {
            margin-top: 10px;
            font-style: italic;
            color: #666;
        }
    </style>
</head>
<body>
    <h1>API Documentation</h1>
    {{range .}}
    <div class="model">
        <div class="model-name">{{.Name}}</div>
        <div class="model-description">{{.Description}}</div>
        {{range .Handlers}}
        <div class="handler">
            <div>
                <span class="method {{toLowerCase .Method}}">{{.Method}}</span>
                <span class="path">{{.Path}}</span>
            </div>
            {{if .Description}}
            <div class="description">{{.Description}}</div>
            {{end}}
            {{if .Fields}}
            <div class="fields">
                Fields: {{range .Fields}}<span>{{.}}</span>{{end}}
            </div>
            {{end}}
        </div>
        {{end}}
    </div>
    {{end}}
</body>
</html>
`

// RegisterApiDoc 注册API文档路由
func RegisterApiDoc(router gin.IRouter, path string) {
	router.GET(path, func(c *gin.Context) {
		// 将 map 转换为 slice，以便添加模型描述
		models := make([]ModelInfo, 0)
		for modelName, handlers := range registeredHandlers {
			// 查找对应的 Crud 实例以获取描述
			var description string
			for _, crud := range cruds {
				if reflect.TypeOf(crud.entity).Elem().Name() == modelName {
					description = crud.description
					break
				}
			}
			models = append(models, ModelInfo{
				Name:        modelName,
				Description: description,
				Handlers:    handlers,
			})
		}

		// 创建模板
		tmpl := template.New("api-doc")

		// 添加自定义函数
		tmpl = tmpl.Funcs(template.FuncMap{
			"toLowerCase": strings.ToLower,
		})

		// 解析模板
		tmpl, err := tmpl.Parse(apiDocTemplate)
		if err != nil {
			JsonErr(c, CodeError, "Failed to parse template: "+err.Error())
			return
		}

		// 设置响应头
		c.Header("Content-Type", "text/html; charset=utf-8")

		// 执行模板
		if err := tmpl.Execute(c.Writer, models); err != nil {
			JsonErr(c, CodeError, "Failed to execute template: "+err.Error())
			return
		}
	})
}
