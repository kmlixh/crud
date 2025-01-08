package crud

import (
	"fmt"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/kmlixh/gom/v4"
	"github.com/kmlixh/gom/v4/define"
)

// Debug 标记
var Debug bool

// 调试输出函数
func debugf(format string, args ...interface{}) {
	if Debug {
		fmt.Printf(format+"\n", args...)
	}
}

// 处理器类型常量
const (
	LIST   = "list"   // 列表
	PAGE   = "page"   // 分页
	SINGLE = "single" // 单条
	SAVE   = "save"   // 保存
	UPDATE = "update" // 更新
	DELETE = "delete" // 删除
)

// 处理阶段常量
const (
	PreProcess  = "pre_process"
	BuildQuery  = "build_query"
	Execute     = "execute"
	PostProcess = "post_process"
)

// 处理时机常量
const (
	BeforePhase = "before"
	OnPhase     = "on"
	AfterPhase  = "after"
)

// ProcessContext 处理上下文
type ProcessContext struct {
	GinContext *gin.Context
	Chain      *gom.Chain
	Data       map[string]interface{}
}

// ItemHandler 处理器
type ItemHandler struct {
	Path          string
	Method        string
	AllowedFields []string
	processors    map[string]map[string][]ProcessorFunc
	Description   string
}

// ProcessorFunc 处理器函数类型
type ProcessorFunc func(*ProcessContext) error

// DefaultProcessors 默认处理器
type DefaultProcessors struct {
	crud *Crud
}

// Crud 结构体
type Crud struct {
	db             *gom.DB
	tableName      string
	model          interface{}
	handlers       map[string]*ItemHandler
	Description    string
	hasCreatedAt   bool
	hasUpdatedAt   bool
	createdAtField reflect.StructField
	updatedAtField reflect.StructField
}

// New2 创建新的 CRUD 实例
func New2(db *gom.DB, model interface{}) *Crud {
	debugf("Creating new CRUD instance")
	if db == nil {
		debugf("Error: database connection is nil")
		return nil
	}

	crud := &Crud{
		db:       db,
		model:    model,
		handlers: make(map[string]*ItemHandler),
	}

	// 获取表名
	if m, ok := model.(interface{ GetTableName() string }); ok {
		crud.tableName = m.GetTableName()
	} else if m, ok := model.(interface{ TableName() string }); ok {
		crud.tableName = m.TableName()
	} else {
		t := reflect.TypeOf(model)
		if t.Kind() == reflect.Ptr {
			t = t.Elem()
		}
		crud.tableName = strings.ToLower(t.Name())
	}
	debugf("Table name: %s", crud.tableName)

	// 检查并记录时间字段信息
	modelType := reflect.TypeOf(model)
	if modelType.Kind() == reflect.Ptr {
		modelType = modelType.Elem()
	}

	for i := 0; i < modelType.NumField(); i++ {
		field := modelType.Field(i)
		if field.Type == reflect.TypeOf(time.Time{}) {
			gomTag := field.Tag.Get("gom")
			if gomTag != "" {
				parts := strings.Split(gomTag, ",")
				if len(parts) > 0 {
					dbColumn := parts[0]
					if dbColumn == "created_at" {
						crud.hasCreatedAt = true
						crud.createdAtField = field
					} else if dbColumn == "updated_at" {
						crud.hasUpdatedAt = true
						crud.updatedAtField = field
					}
				}
			}
		}
	}

	// 初始化默认处理器
	crud.initDefaultHandlers()

	return crud
}

// deserializeJSON attempts to deserialize JSON data into a new instance of the target struct
func (c *Crud) deserializeJSON(ctx *gin.Context) (interface{}, error) {
	// Create a new instance of the model type
	modelType := reflect.TypeOf(c.model)
	if modelType.Kind() == reflect.Ptr {
		modelType = modelType.Elem()
	}
	newInstance := reflect.New(modelType).Interface()

	// Use gin's ShouldBind to handle the deserialization
	if err := ctx.ShouldBind(newInstance); err != nil {
		return nil, fmt.Errorf("failed to bind JSON data: %v", err)
	}

	return newInstance, nil
}

// initDefaultHandlers 初始化默认处理器
func (c *Crud) initDefaultHandlers() {
	// 列表处理器
	c.AddHandler(LIST, http.MethodGet, c.initListHandler())

	// 分页处理器
	c.AddHandler(PAGE, http.MethodGet, c.initPageHandler())

	// 单条处理器
	c.AddHandler(SINGLE, http.MethodGet, c.initSingleHandler())

	// 保存处理器
	c.AddHandler(SAVE, http.MethodPost, c.initSaveHandler())

	// 更新处理器
	c.AddHandler(UPDATE, http.MethodPut, c.initUpdateHandler())

	// 删除处理器
	c.AddHandler(DELETE, http.MethodDelete, c.initDeleteHandler())
}

// initSaveHandler 初始化保存处理器
func (c *Crud) initSaveHandler() *ItemHandler {
	saveHandler := NewHandler("/save", http.MethodPost)
	saveHandler.SetDescription("创建新记录")
	saveHandler.AddProcessor(PreProcess, OnPhase, func(ctx *ProcessContext) error {
		instance := reflect.New(reflect.TypeOf(c.model).Elem()).Interface()
		if err := ctx.GinContext.ShouldBind(instance); err != nil {
			JsonErr(ctx.GinContext, CodeInvalid, err.Error())
			return err
		}
		ctx.Data = map[string]interface{}{
			"instance": instance,
		}
		return nil
	})
	saveHandler.AddProcessor(Execute, OnPhase, func(ctx *ProcessContext) error {
		instance := ctx.Data["instance"]
		result := c.db.Chain().Table(c.tableName).Values(instance).Save()
		if result.Error != nil {
			JsonErr(ctx.GinContext, CodeError, fmt.Sprintf("failed to save record: %v", result.Error))
			return result.Error
		}
		ctx.Data["result"] = instance
		return nil
	})
	saveHandler.AddProcessor(PostProcess, OnPhase, func(ctx *ProcessContext) error {
		JsonOk(ctx.GinContext, ctx.Data["result"])
		return nil
	})
	return saveHandler
}

// initUpdateHandler 初始化更新处理器
func (c *Crud) initUpdateHandler() *ItemHandler {
	updateHandler := NewHandler("/update", http.MethodPut)
	updateHandler.SetDescription("更新记录")
	updateHandler.AddProcessor(PreProcess, OnPhase, func(ctx *ProcessContext) error {
		instance := reflect.New(reflect.TypeOf(c.model).Elem()).Interface()
		if err := ctx.GinContext.ShouldBind(instance); err != nil {
			JsonErr(ctx.GinContext, CodeInvalid, err.Error())
			return err
		}
		ctx.Data = map[string]interface{}{
			"instance": instance,
		}
		return nil
	})
	updateHandler.AddProcessor(Execute, OnPhase, func(ctx *ProcessContext) error {
		instance := ctx.Data["instance"]
		result := c.db.Chain().From(instance).Update()
		if result.Error != nil {
			JsonErr(ctx.GinContext, CodeError, fmt.Sprintf("failed to update record: %v", result.Error))
			return result.Error
		}
		ctx.Data["result"] = instance
		return nil
	})
	updateHandler.AddProcessor(PostProcess, OnPhase, func(ctx *ProcessContext) error {
		JsonOk(ctx.GinContext, ctx.Data["result"])
		return nil
	})
	return updateHandler
}

// initListHandler 初始化列表处理器
func (c *Crud) initListHandler() *ItemHandler {
	listHandler := NewHandler("/list", http.MethodGet)
	listHandler.SetDescription("获取所有记录")
	listHandler.AddProcessor(PreProcess, OnPhase, func(ctx *ProcessContext) error {
		if ctx.Chain == nil {
			instance := reflect.New(reflect.TypeOf(c.model).Elem()).Interface()
			ctx.Chain = c.db.Chain().From(instance)
		}
		return nil
	})
	listHandler.AddProcessor(Execute, OnPhase, func(ctx *ProcessContext) error {
		result := ctx.Chain.List()
		if result.Error != nil {
			return result.Error
		}
		if ctx.Data == nil {
			ctx.Data = make(map[string]interface{})
		}
		ctx.Data["result"] = result.Data
		return nil
	})
	listHandler.AddProcessor(PostProcess, OnPhase, func(ctx *ProcessContext) error {
		JsonOk(ctx.GinContext, ctx.Data["result"])
		return nil
	})
	return listHandler
}

// initPageHandler 初始化分页处理器
func (c *Crud) initPageHandler() *ItemHandler {
	h := NewHandler("/page", http.MethodGet)
	h.SetDescription("分页查询记录")

	// 初始化上下文
	h.AddProcessor(PreProcess, OnPhase, func(ctx *ProcessContext) error {
		// 设置默认分页参数
		page := 1
		size := 10

		// 从请求中获取分页参数
		if p := ctx.GinContext.Query("page"); p != "" {
			if pInt, err := strconv.Atoi(p); err == nil && pInt > 0 {
				page = pInt
			}
		}
		if s := ctx.GinContext.Query("size"); s != "" {
			if sInt, err := strconv.Atoi(s); err == nil && sInt > 0 {
				size = sInt
			}
		}

		ctx.Data = map[string]interface{}{
			"page": page,
			"size": size,
			"crud": c,
		}
		return nil
	})

	// 构建查询
	h.AddProcessor(BuildQuery, OnPhase, func(ctx *ProcessContext) error {
		// 初始化查询链
		crud := ctx.Data["crud"].(*Crud)
		ctx.Chain = crud.db.Chain().Table(crud.tableName)

		// 处理查询条件
		if err := ParseQueryConditions(ctx.GinContext, ctx.Chain); err != nil {
			return err
		}

		// 处理排序
		if sort := ctx.GinContext.Query("sort"); sort != "" {
			if strings.HasPrefix(sort, "-") {
				field := strings.TrimPrefix(sort, "-")
				ctx.Chain = ctx.Chain.OrderByDesc(field)
				debugf("Applied descending sort on field: %s", field)
			} else {
				ctx.Chain = ctx.Chain.OrderBy(sort)
				debugf("Applied ascending sort on field: %s", sort)
			}
		}

		// 处理分页
		page := ctx.Data["page"].(int)
		size := ctx.Data["size"].(int)
		offset := (page - 1) * size
		ctx.Chain = ctx.Chain.Offset(offset).Limit(size)
		debugf("Applied pagination: offset=%d, limit=%d", offset, size)

		return nil
	})

	// 执行查询
	h.AddProcessor(Execute, OnPhase, func(ctx *ProcessContext) error {
		// 执行查询
		result := ctx.Chain.List()
		if result.Error != nil {
			JsonErr(ctx.GinContext, CodeError, fmt.Sprintf("failed to execute query: %v", result.Error))
			return result.Error
		}

		// 获取总记录数
		crud := ctx.Data["crud"].(*Crud)
		countChain := crud.db.Chain().Table(crud.tableName)
		// 复制查询条件
		if err := ParseQueryConditions(ctx.GinContext, countChain); err != nil {
			JsonErr(ctx.GinContext, CodeError, fmt.Sprintf("failed to parse query conditions: %v", err))
			return err
		}
		total, err := countChain.Count()
		if err != nil {
			JsonErr(ctx.GinContext, CodeError, "failed to get total count")
			return fmt.Errorf("failed to get total count: %v", err)
		}

		// 确保 result.Data 不为空
		if result.Data == nil {
			result.Data = []map[string]interface{}{}
		}

		ctx.Data["result"] = map[string]interface{}{
			"list":  result.Data,
			"total": total,
			"page":  ctx.Data["page"].(int),
			"size":  ctx.Data["size"].(int),
		}
		return nil
	})

	// 响应处理
	h.AddProcessor(PostProcess, OnPhase, func(ctx *ProcessContext) error {
		if result, ok := ctx.Data["result"].(map[string]interface{}); ok {
			JsonOk(ctx.GinContext, result)
			return nil
		}
		return fmt.Errorf("invalid result format")
	})

	return h
}

// initSingleHandler 初始化单条处理器
func (c *Crud) initSingleHandler() *ItemHandler {
	singleHandler := NewHandler("/detail/:id", http.MethodGet)
	singleHandler.SetDescription("获取单条记录详情")
	singleHandler.AddProcessor(PreProcess, OnPhase, func(ctx *ProcessContext) error {
		id := ctx.GinContext.Param("id")
		if id == "" {
			JsonErr(ctx.GinContext, CodeInvalid, "id is required")
			return fmt.Errorf("id is required")
		}
		ctx.Data = map[string]interface{}{
			"id": id,
		}
		return nil
	})
	singleHandler.AddProcessor(BuildQuery, OnPhase, func(ctx *ProcessContext) error {
		if ctx.Chain == nil {
			ctx.Chain = c.db.Chain().Table(c.tableName)
		}
		id := ctx.Data["id"].(string)
		ctx.Chain = ctx.Chain.Eq("id", id)
		return nil
	})
	singleHandler.AddProcessor(Execute, OnPhase, func(ctx *ProcessContext) error {
		result := ctx.Chain.First()
		if result.Error != nil {
			if result.Error.Error() == "sql: no rows in result set" {
				JsonErr(ctx.GinContext, CodeNotFound, "record not found")
				ctx.GinContext.Abort()
				return nil
			}
			JsonErr(ctx.GinContext, CodeError, fmt.Sprintf("failed to retrieve record: %v", result.Error))
			return result.Error
		}
		if len(result.Data) == 0 {
			JsonErr(ctx.GinContext, CodeNotFound, "record not found")
			ctx.GinContext.Abort()
			return nil
		}

		ctx.Data["result"] = result.Data[0]
		return nil
	})
	singleHandler.AddProcessor(PostProcess, OnPhase, func(ctx *ProcessContext) error {
		if result, ok := ctx.Data["result"].(map[string]interface{}); ok {
			JsonOk(ctx.GinContext, result)
			return nil
		}
		return fmt.Errorf("invalid result format")
	})
	return singleHandler
}

// initDeleteHandler 初始化删除处理器
func (c *Crud) initDeleteHandler() *ItemHandler {
	deleteHandler := NewHandler("/delete/:id", http.MethodDelete)
	deleteHandler.SetDescription("删除指定记录")
	deleteHandler.AddProcessor(PreProcess, OnPhase, func(ctx *ProcessContext) error {
		id := ctx.GinContext.Param("id")
		if id == "" {
			JsonErr(ctx.GinContext, CodeInvalid, "id is required")
			return fmt.Errorf("id is required")
		}
		ctx.Data = map[string]interface{}{
			"id": id,
		}
		return nil
	})
	deleteHandler.AddProcessor(Execute, OnPhase, func(ctx *ProcessContext) error {
		id := ctx.Data["id"]

		result := c.db.Chain().Table(c.tableName).Eq("id", id).Delete()
		if result.Error != nil {
			JsonErr(ctx.GinContext, CodeError, fmt.Sprintf("failed to delete record: %v", result.Error))
			return result.Error
		}

		affected, err := result.RowsAffected()
		if err != nil {
			JsonErr(ctx.GinContext, CodeError, fmt.Sprintf("failed to get affected rows: %v", err))
			return err
		}

		if affected == 0 {
			JsonErr(ctx.GinContext, CodeNotFound, "record not found")
			return fmt.Errorf("record not found")
		}

		ctx.Data["result"] = map[string]interface{}{
			"id": id,
		}
		return nil
	})
	deleteHandler.AddProcessor(PostProcess, OnPhase, func(ctx *ProcessContext) error {
		if result, ok := ctx.Data["result"].(map[string]interface{}); ok {
			JsonOk(ctx.GinContext, result)
			return nil
		}
		return fmt.Errorf("invalid result format")
	})
	return deleteHandler
}

// parseQueryConditions 解析查询条件
func ParseQueryConditions(c *gin.Context, chain *gom.Chain) error {
	// 处理字段选择
	if fields := c.Query("fields"); fields != "" {
		chain = chain.Fields(strings.Split(fields, ",")...)
	}

	// 从URL参数中解析查询条件
	for k, v := range c.Request.URL.Query() {
		if len(v) > 0 && k != "page" && k != "size" && k != "sort" && k != "fields" {
			// 处理范围查询
			if strings.HasSuffix(k, "_gte") {
				field := strings.TrimSuffix(k, "_gte")
				chain = chain.Where(field, define.OpGe, v[0])
			} else if strings.HasSuffix(k, "_lte") {
				field := strings.TrimSuffix(k, "_lte")
				chain = chain.Where(field, define.OpLe, v[0])
			} else if strings.HasSuffix(k, "_gt") {
				field := strings.TrimSuffix(k, "_gt")
				chain = chain.Where(field, define.OpGt, v[0])
			} else if strings.HasSuffix(k, "_lt") {
				field := strings.TrimSuffix(k, "_lt")
				chain = chain.Where(field, define.OpLt, v[0])
			} else if strings.HasSuffix(k, "_ne") {
				field := strings.TrimSuffix(k, "_ne")
				chain = chain.Where(field, define.OpNe, v[0])
			} else if strings.HasSuffix(k, "_like") {
				field := strings.TrimSuffix(k, "_like")
				chain = chain.Where(field, define.OpLike, fmt.Sprintf("%%%s%%", v[0]))
			} else if strings.HasSuffix(k, "_in") {
				field := strings.TrimSuffix(k, "_in")
				values := strings.Split(v[0], ",")
				chain = chain.Where(field, define.OpIn, values)
			} else {
				chain = chain.Where(k, define.OpEq, v[0])
			}
		}
	}
	return nil
}

// Page 处理器
func (dp *DefaultProcessors) Page() *ItemHandler {
	h := &ItemHandler{
		Path:       "/page",
		Method:     http.MethodGet,
		processors: make(map[string]map[string][]ProcessorFunc),
	}

	// 预处理
	h.AddProcessor(PreProcess, OnPhase, func(ctx *ProcessContext) error {
		debugf("Processing pagination parameters")
		// 获取分页参数
		page := 1
		size := 10
		if p := ctx.GinContext.Query("page"); p != "" {
			if v, err := strconv.Atoi(p); err == nil && v > 0 {
				page = v
			}
		}
		if s := ctx.GinContext.Query("size"); s != "" {
			if v, err := strconv.Atoi(s); err == nil && v > 0 {
				size = v
			}
		}
		ctx.Data["page"] = page
		ctx.Data["size"] = size
		debugf("Pagination: page=%d, size=%d", page, size)
		return nil
	})

	// 构建对象
	h.AddProcessor(BuildQuery, BeforePhase, func(ctx *ProcessContext) error {
		debugf("Building query chain")
		if v, exists := ctx.GinContext.Get("chain"); exists {
			ctx.Chain = v.(*gom.Chain)
			debugf("Using existing chain")
		} else {
			if dp.crud.tableName == "" {
				return fmt.Errorf("table name is not set")
			}
			if dp.crud.db == nil {
				return fmt.Errorf("database connection is not set")
			}
			ctx.Chain = dp.crud.db.Chain().Table(dp.crud.tableName)
			debugf("Created new chain for table: %s", dp.crud.tableName)
		}
		return nil
	})

	// 构建SQL
	h.AddProcessor(BuildQuery, OnPhase, func(ctx *ProcessContext) error {
		debugf("Building SQL query")
		debugf("Parsing query conditions from URL: %s", ctx.GinContext.Request.URL.String())

		// 处理查询条件
		if err := ParseQueryConditions(ctx.GinContext, ctx.Chain); err != nil {
			debugf("Error parsing query conditions: %v", err)
			return err
		}

		// 处理字段选择
		if len(h.AllowedFields) > 0 {
			ctx.Chain = ctx.Chain.Fields(h.AllowedFields...)
			debugf("Selected fields: %v", h.AllowedFields)
		}

		// 处理排序
		if sort := ctx.GinContext.Query("sort"); sort != "" {
			if strings.HasPrefix(sort, "-") {
				field := strings.TrimPrefix(sort, "-")
				ctx.Chain = ctx.Chain.OrderByDesc(field)
				debugf("Applied descending sort on field: %s", field)
			} else {
				ctx.Chain = ctx.Chain.OrderBy(sort)
				debugf("Applied ascending sort on field: %s", sort)
			}
		}

		// 处理分页
		page := ctx.Data["page"].(int)
		size := ctx.Data["size"].(int)
		offset := (page - 1) * size
		ctx.Chain = ctx.Chain.Offset(offset).Limit(size)
		debugf("Applied pagination: offset=%d, limit=%d", offset, size)

		// 执行查询
		result := ctx.Chain.List()
		if result.Error != nil {
			JsonErr(ctx.GinContext, CodeError, fmt.Sprintf("failed to execute query: %v", result.Error))
			return result.Error
		}

		// 获取总记录数
		crud := ctx.Data["crud"].(*Crud)
		countChain := crud.db.Chain().Table(crud.tableName)
		// 复制查询条件
		if err := ParseQueryConditions(ctx.GinContext, countChain); err != nil {
			JsonErr(ctx.GinContext, CodeError, fmt.Sprintf("failed to parse query conditions: %v", err))
			return err
		}
		total, err := countChain.Count()
		if err != nil {
			JsonErr(ctx.GinContext, CodeError, "failed to get total count")
			return fmt.Errorf("failed to get total count: %v", err)
		}

		ctx.Data["result"] = map[string]interface{}{
			"list":  result.Data,
			"total": total,
			"page":  page,
			"size":  size,
		}
		return nil
	})

	// 响应处理
	h.AddProcessor(PostProcess, OnPhase, func(ctx *ProcessContext) error {
		debugf("Processing response")
		JsonOk(ctx.GinContext, ctx.Data["result"])
		return nil
	})

	return h
}

// AddProcessor 添加处理器
func (h *ItemHandler) AddProcessor(phase string, timing string, processor ProcessorFunc) *ItemHandler {
	if h.processors == nil {
		h.processors = make(map[string]map[string][]ProcessorFunc)
	}
	if h.processors[phase] == nil {
		h.processors[phase] = make(map[string][]ProcessorFunc)
	}
	h.processors[phase][timing] = append(h.processors[phase][timing], processor)
	return h
}

// HandleRequest 处理请求
func (h *ItemHandler) HandleRequest(c *gin.Context) {
	if _, exists := c.Get("response_sent"); exists {
		return
	}

	ctx := &ProcessContext{
		GinContext: c,
		Data:       make(map[string]interface{}),
	}
	c.Set("process_context", ctx)

	// 处理 JSON 绑定错误
	if c.Request.Method == http.MethodPost || c.Request.Method == http.MethodPut {
		if c.Request.Header.Get("Content-Type") != "application/json" {
			JsonErr(c, CodeInvalid, "Content-Type must be application/json")
			return
		}
	}

	for _, phase := range []string{PreProcess, BuildQuery, Execute, PostProcess} {
		if err := h.executeProcessors(phase, BeforePhase, ctx); err != nil {
			if _, exists := c.Get("response_sent"); !exists {
				if err.Error() == "sql: no rows in result set" {
					JsonErr(c, CodeNotFound, "record not found")
				} else {
					JsonErr(c, CodeInvalid, err.Error())
				}
				c.Set("response_sent", true)
			}
			return
		}
		if err := h.executeProcessors(phase, OnPhase, ctx); err != nil {
			if _, exists := c.Get("response_sent"); !exists {
				if err.Error() == "sql: no rows in result set" {
					JsonErr(c, CodeNotFound, "record not found")
				} else {
					JsonErr(c, CodeInvalid, err.Error())
				}
				c.Set("response_sent", true)
			}
			return
		}
		if err := h.executeProcessors(phase, AfterPhase, ctx); err != nil {
			if _, exists := c.Get("response_sent"); !exists {
				if err.Error() == "sql: no rows in result set" {
					JsonErr(c, CodeNotFound, "record not found")
				} else {
					JsonErr(c, CodeInvalid, err.Error())
				}
				c.Set("response_sent", true)
			}
			return
		}
		if c.IsAborted() {
			return
		}
	}
}

// executeProcessors 执行处理器
func (h *ItemHandler) executeProcessors(phase string, timing string, ctx *ProcessContext) error {
	if h.processors == nil {
		return nil
	}
	if h.processors[phase] == nil {
		return nil
	}
	if h.processors[phase][timing] == nil {
		return nil
	}
	for _, processor := range h.processors[phase][timing] {
		if err := processor(ctx); err != nil {
			return err
		}
	}
	return nil
}

// SetDescription 设置描述
func (h *ItemHandler) SetDescription(desc string) *ItemHandler {
	h.Description = desc
	return h
}

// GetHandler 获取处理器
func (c *Crud) GetHandler(handlerType string) (*ItemHandler, bool) {
	handler, ok := c.handlers[handlerType]
	return handler, ok
}

// AddHandler 添加处理器
func (c *Crud) AddHandler(handlerType string, method string, handler *ItemHandler) {
	c.handlers[handlerType] = handler
}

// SetDescription 设置描述
func (c *Crud) SetDescription(desc string) *Crud {
	c.Description = desc
	return c
}

// RegisterRoutes 注册路由
func (c *Crud) RegisterRoutes(group *gin.RouterGroup, prefix string) {
	if Debug {
		debugf("Registering routes for table: %s", c.tableName)
	}
	for _, handler := range c.handlers {
		path := prefix + handler.Path
		if Debug {
			debugf("  %s %s", handler.Method, path)
			if handler.Description != "" {
				debugf("    Description: %s", handler.Description)
			}
			if len(handler.AllowedFields) > 0 {
				debugf("    Allowed Fields: %v", handler.AllowedFields)
			}
		}
		group.Handle(handler.Method, path, handler.HandleRequest)
	}
	if Debug {
		debugf("Route registration completed")
	}
}

// RegisterApi 注册 API 信息
func (c *Crud) RegisterApi() map[string]interface{} {
	apis := make(map[string]interface{})
	for handlerType, handler := range c.handlers {
		apiInfo := map[string]interface{}{
			"path":        handler.Path,
			"method":      handler.Method,
			"type":        handlerType,
			"fields":      handler.AllowedFields,
			"description": handler.Description,
		}
		apis[handlerType] = apiInfo
	}
	return apis
}

// RegisterApi 注册 API 信息
func RegisterApi(r *gin.Engine, path string) {
	apiDocs := make(map[string]interface{})
	for _, crud := range registeredCruds {
		modelType := reflect.TypeOf(crud.model)
		if modelType.Kind() == reflect.Ptr {
			modelType = modelType.Elem()
		}
		apiDocs[crud.tableName] = map[string]interface{}{
			"model_name":  modelType.Name(),
			"description": crud.Description,
			"apis":        crud.RegisterApi(),
		}
	}

	r.GET(path, func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"code": 0,
			"data": apiDocs,
		})
	})
}

// RegisterApiDoc 注册 API 文档
func RegisterApiDoc(r *gin.Engine, path string) {
	apiDocs := make(map[string]interface{})
	for _, crud := range registeredCruds {
		modelType := reflect.TypeOf(crud.model)
		if modelType.Kind() == reflect.Ptr {
			modelType = modelType.Elem()
		}
		apiDocs[crud.tableName] = map[string]interface{}{
			"model_name":  modelType.Name(),
			"description": crud.Description,
			"apis":        crud.RegisterApi(),
		}
	}

	r.GET(path, func(c *gin.Context) {
		html := `<!DOCTYPE html>
			<html>
			<head>
				<title>API Documentation</title>
				<style>
					body { font-family: Arial, sans-serif; margin: 20px; }
					.table { width: 100%; border-collapse: collapse; margin-bottom: 20px; }
					.table th, .table td { padding: 8px; border: 1px solid #ddd; }
					.table th { background-color: #f5f5f5; }
					.method-get { color: #2196F3; }
					.method-post { color: #4CAF50; }
					.method-put { color: #FF9800; }
					.method-delete { color: #F44336; }
				</style>
			</head>
			<body>
				<h1>API Documentation</h1>`

		for table, doc := range apiDocs {
			if docMap, ok := doc.(map[string]interface{}); ok {
				html += fmt.Sprintf("<h2>%s (%s)</h2>", docMap["model_name"], table)
				if desc, ok := docMap["description"].(string); ok && desc != "" {
					html += fmt.Sprintf("<p>%s</p>", desc)
				}

				html += `<table class="table">
						<tr>
							<th>Type</th>
							<th>Method</th>
							<th>Path</th>
							<th>Fields</th>
							<th>Description</th>
						</tr>`

				if apis, ok := docMap["apis"].(map[string]interface{}); ok {
					for _, api := range apis {
						if apiInfo, ok := api.(map[string]interface{}); ok {
							method := apiInfo["method"].(string)
							methodClass := fmt.Sprintf("method-%s", strings.ToLower(method))
							fields := ""
							if apiInfo["fields"] != nil {
								if fieldSlice, ok := apiInfo["fields"].([]string); ok {
									fields = strings.Join(fieldSlice, ", ")
								}
							}
							html += fmt.Sprintf(`
									<tr>
										<td>%s</td>
										<td class="%s">%s</td>
										<td>%s</td>
										<td>%s</td>
										<td>%s</td>
									</tr>`,
								apiInfo["type"],
								methodClass,
								method,
								apiInfo["path"],
								fields,
								apiInfo["description"])
						}
					}
				}
				html += "</table>"
			}
		}

		html += "</body></html>"

		c.Header("Content-Type", "text/html")
		c.String(http.StatusOK, html)
	})
}

// 用于存储所有注册的 Crud 实例
var registeredCruds []*Crud

// RegisterCrud 注册一个 Crud 实例
func RegisterCrud(crud *Crud) {
	registeredCruds = append(registeredCruds, crud)
}

// NewHandler 创建新的处理器
func NewHandler(path string, method string) *ItemHandler {
	return &ItemHandler{
		Path:       path,
		Method:     method,
		processors: make(map[string]map[string][]ProcessorFunc),
	}
}

// GetHandlers 返回 Gin 处理器列表
func (h *ItemHandler) GetHandlers() []gin.HandlerFunc {
	handlers := make([]gin.HandlerFunc, 0)
	handlers = append(handlers, h.HandleRequest)
	return handlers
}

// mapDBToJSON 将数据库数据映射到 JSON
func (c *Crud) mapDBToJSON(dbData map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	modelType := reflect.TypeOf(c.model)
	if modelType.Kind() == reflect.Ptr {
		modelType = modelType.Elem()
	}

	// 直接映射 ID，保持原始类型
	if id, ok := dbData["id"]; ok {
		result["id"] = id
	}

	// Map other fields based on gom tags
	for i := 0; i < modelType.NumField(); i++ {
		field := modelType.Field(i)
		gomTag := field.Tag.Get("gom")
		if gomTag == "" {
			continue
		}
		parts := strings.Split(gomTag, ",")
		if len(parts) == 0 {
			continue
		}
		dbColumn := parts[0]

		// Skip m2m fields
		isM2M := false
		for _, part := range parts[1:] {
			if strings.HasPrefix(part, "m2m:") {
				isM2M = true
				break
			}
		}
		if isM2M {
			continue
		}

		jsonTag := field.Tag.Get("json")
		if jsonTag == "" {
			continue
		}
		jsonField := strings.Split(jsonTag, ",")[0]

		if val, ok := dbData[dbColumn]; ok {
			// 特殊处理 identifier 字段
			if dbColumn == "identifier" {
				result["domainName"] = val
			} else {
				result[jsonField] = val
			}
		}
	}

	return result
}

// Save 处理器
func (c *Crud) Save(ctx *gin.Context) {
	// 创建新的实例并绑定数据
	instance := reflect.New(reflect.TypeOf(c.model).Elem()).Interface()
	if err := ctx.ShouldBind(instance); err != nil {
		JsonErr(ctx, CodeInvalid, err.Error())
		return
	}

	// 设置时间字段
	now := time.Now()
	val := reflect.ValueOf(instance).Elem()

	if c.hasCreatedAt {
		createdAtField := val.FieldByName(c.createdAtField.Name)
		if createdAtField.IsValid() && createdAtField.CanSet() {
			createdAtField.Set(reflect.ValueOf(now))
		}
	}

	if c.hasUpdatedAt {
		updatedAtField := val.FieldByName(c.updatedAtField.Name)
		if updatedAtField.IsValid() && updatedAtField.CanSet() {
			updatedAtField.Set(reflect.ValueOf(now))
		}
	}

	// 使用 gom 的 From 方法保存数据
	result := c.db.Chain().From(instance).Save()
	if result.Error != nil {
		if strings.Contains(strings.ToLower(result.Error.Error()), "duplicate") {
			JsonErr(ctx, CodeError, fmt.Sprintf("duplicate record: %v", result.Error))
			return
		}
		JsonErr(ctx, CodeError, fmt.Sprintf("failed to save record: %v", result.Error))
		return
	}

	// 获取新创建的记录
	id, _ := result.LastInsertId()
	newRecord := c.db.Chain().From(instance).Eq("id", id).First()
	if newRecord.Error != nil {
		JsonErr(ctx, CodeError, fmt.Sprintf("failed to retrieve created record: %v", newRecord.Error))
		return
	}

	JsonOk(ctx, newRecord.Data[0])
}

// Update 处理器
func (c *Crud) Update(ctx *gin.Context) {
	// 创建新的实例并绑定数据
	instance := reflect.New(reflect.TypeOf(c.model).Elem()).Interface()
	if err := ctx.ShouldBind(instance); err != nil {
		JsonErr(ctx, CodeInvalid, err.Error())
		return
	}

	// 获取ID
	val := reflect.ValueOf(instance).Elem()
	idField := val.FieldByName("ID")
	if !idField.IsValid() || idField.IsZero() {
		JsonErr(ctx, CodeInvalid, "id is required")
		return
	}
	id := idField.Interface()

	// 设置更新时间
	if c.hasUpdatedAt {
		updatedAtField := val.FieldByName(c.updatedAtField.Name)
		if updatedAtField.IsValid() && updatedAtField.CanSet() {
			updatedAtField.Set(reflect.ValueOf(time.Now()))
		}
	}

	// 使用 gom 的 From 方法更新数据
	chain := c.db.Chain().From(instance)

	// 将实例转换为 map
	values := make(map[string]interface{})
	typ := val.Type()
	for i := 0; i < val.NumField(); i++ {
		field := typ.Field(i)
		if gomTag := field.Tag.Get("gom"); gomTag != "" {
			parts := strings.Split(gomTag, ",")
			if len(parts) > 0 {
				dbColumn := parts[0]
				if dbColumn != "id" { // Skip ID field
					fieldValue := val.Field(i)
					if fieldValue.IsValid() && !fieldValue.IsZero() {
						values[dbColumn] = fieldValue.Interface()
					}
				}
			}
		}
	}

	result := chain.Eq("id", id).Update(values)
	if result.Error != nil {
		JsonErr(ctx, CodeError, fmt.Sprintf("failed to update record: %v", result.Error))
		return
	}

	affected, err := result.RowsAffected()
	if err != nil {
		JsonErr(ctx, CodeError, fmt.Sprintf("failed to get affected rows: %v", err))
		return
	}

	if affected == 0 {
		JsonErr(ctx, CodeNotFound, "record not found")
		return
	}

	// 获取更新后的记录
	updatedRecord := chain.Eq("id", id).First()
	if updatedRecord.Error != nil {
		JsonErr(ctx, CodeError, fmt.Sprintf("failed to retrieve updated record: %v", updatedRecord.Error))
		return
	}

	JsonOk(ctx, updatedRecord.Data[0])
}
