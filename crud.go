package crud

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

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
	db        *gom.DB
	entity    interface{}
	tableName string
	handlers  map[string]ItemHandler
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

	// 预处理
	h.AddProcessor(PreProcess, OnPhase, func(ctx *ProcessContext) error {
		page, size := parsePagination(ctx.GinContext)
		ctx.Data["page"] = page
		ctx.Data["size"] = size
		return nil
	})

	// 构建对象
	h.AddProcessor(BuildQuery, BeforePhase, func(ctx *ProcessContext) error {
		// 处理查询条件
		if err := parseQueryConditions(ctx.GinContext, ctx.Chain); err != nil {
			return err
		}
		return nil
	})

	h.AddProcessor(BuildQuery, OnPhase, func(ctx *ProcessContext) error {
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

// New2 创建新的Crud实例（使用 gom.DB 的 GetTableName 方法获取表名）
func New2(db *gom.DB, entity interface{}, defaultHandlers ...string) *Crud {
	tableName, err := db.GetTableName(entity)
	if err != nil {
		// 如果获取表名失败，返回空表名，让 New 方法处理
		return New(db, entity, "", defaultHandlers...)
	}
	return New(db, entity, tableName, defaultHandlers...)
}

// New 创建新的Crud实例
func New(db *gom.DB, entity interface{}, tableName string, defaultHandlers ...string) *Crud {
	// 如果表名为空，则使用 New2 方法
	if tableName == "" {
		return New2(db, entity, defaultHandlers...)
	}

	crud := &Crud{
		db:        db,
		entity:    entity,
		tableName: tableName,
		handlers:  make(map[string]ItemHandler),
	}

	// 如果没有指定默认处理器，则创建所有默认处理器
	if len(defaultHandlers) == 0 {
		defaultHandlers = []string{LIST, PAGE, SINGLE, SAVE, UPDATE, DELETE}
	}

	// 创建默认处理器集合
	processors := NewDefaultProcessors(crud)

	// 创建指定的默认处理器
	handlerMap := map[string]*ItemHandler{
		LIST:   processors.List(),
		PAGE:   processors.List(), // 使用相同的列表处理器
		SINGLE: processors.Get(),
		SAVE:   processors.Create(),
		UPDATE: processors.Update(),
		DELETE: processors.Delete(),
	}

	// 注册指定的默认处理器
	for _, name := range defaultHandlers {
		if handler, ok := handlerMap[name]; ok {
			handler.Handler = handler.HandleRequest // 设置处理函数
			crud.handlers[name] = *handler
		}
	}

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
