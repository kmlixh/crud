package crud

import (
	"fmt"
	"math"
	"net/http"
	"reflect"
	"strconv"
	"strings"

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
	db          *gom.DB
	tableName   string
	model       interface{}
	handlers    map[string]*ItemHandler
	Description string
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

	// 初始化默认处理器
	crud.initDefaultHandlers()

	return crud
}

// initDefaultHandlers 初始化默认处理器
func (c *Crud) initDefaultHandlers() {
	// 列表处理器
	listHandler := NewHandler("/list", http.MethodGet)
	listHandler.SetDescription("获取所有记录")
	listHandler.AddProcessor(PreProcess, OnPhase, func(ctx *ProcessContext) error {
		if ctx.Chain == nil {
			ctx.Chain = c.db.Chain()
		}
		ctx.Chain = ctx.Chain.Table(c.tableName)
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
		if result, ok := ctx.Data["result"].([]map[string]interface{}); ok {
			ctx.Data["result"] = result
		}
		JsonOk(ctx.GinContext, ctx.Data["result"])
		return nil
	})
	c.AddHandler(LIST, http.MethodGet, listHandler)

	// 分页处理器
	pageHandler := NewHandler("/page", http.MethodGet)
	pageHandler.SetDescription("分页查询记录")
	pageHandler.AddProcessor(PreProcess, OnPhase, func(ctx *ProcessContext) error {
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
		ctx.Data = map[string]interface{}{
			"page": page,
			"size": size,
		}
		return nil
	})
	pageHandler.AddProcessor(BuildQuery, OnPhase, func(ctx *ProcessContext) error {
		if ctx.Chain == nil {
			ctx.Chain = c.db.Chain()
		}
		ctx.Chain = ctx.Chain.Table(c.tableName)

		// 处理查询条件
		if err := ParseQueryConditions(ctx.GinContext, ctx.Chain); err != nil {
			return err
		}

		// 处理排序
		if sort := ctx.GinContext.Query("sort"); sort != "" {
			if sort[0] == '-' {
				ctx.Chain = ctx.Chain.OrderByDesc(sort[1:])
			} else {
				ctx.Chain = ctx.Chain.OrderBy(sort)
			}
		}

		// 处理分页
		page := ctx.Data["page"].(int)
		size := ctx.Data["size"].(int)
		offset := (page - 1) * size
		ctx.Chain = ctx.Chain.Offset(offset).Limit(size)
		return nil
	})
	pageHandler.AddProcessor(Execute, OnPhase, func(ctx *ProcessContext) error {
		debugf("Executing SQL query")
		// 获取分页参数
		page := ctx.Data["page"].(int)
		size := ctx.Data["size"].(int)

		// 获取总数
		countChain := c.db.Chain().Table(c.tableName)
		if err := ParseQueryConditions(ctx.GinContext, countChain); err != nil {
			debugf("Error parsing query conditions for count: %v", err)
			return err
		}

		// 执行 COUNT 查询
		total, err := countChain.Count()
		if err != nil {
			debugf("Error getting total count: %v", err)
			return err
		}
		debugf("Total count: %d", total)

		// 获取分页数据
		result := ctx.Chain.List()
		if result.Error != nil {
			debugf("Error getting page data: %v", result.Error)
			return result.Error
		}

		// 构建分页响应
		ctx.Data["result"] = map[string]interface{}{
			"page":  page,
			"size":  size,
			"total": total,
			"pages": int(math.Ceil(float64(total) / float64(size))),
			"list":  result.Data,
		}
		debugf("Page result: %+v", ctx.Data["result"])

		return nil
	})
	pageHandler.AddProcessor(PostProcess, OnPhase, func(ctx *ProcessContext) error {
		if result, ok := ctx.Data["result"].(map[string]interface{}); ok {
			if list, ok := result["list"].([]map[string]interface{}); ok {
				result["list"] = list
			}
		}
		JsonOk(ctx.GinContext, ctx.Data["result"])
		return nil
	})
	c.AddHandler(PAGE, http.MethodGet, pageHandler)

	// 单条处理器
	singleHandler := NewHandler("/detail/:id", http.MethodGet)
	singleHandler.SetDescription("获取单条记录详情")
	singleHandler.AddProcessor(PreProcess, OnPhase, func(ctx *ProcessContext) error {
		id := ctx.GinContext.Param("id")
		if id == "" {
			return fmt.Errorf("id is required")
		}
		ctx.Data = map[string]interface{}{
			"id": id,
		}
		return nil
	})
	singleHandler.AddProcessor(BuildQuery, OnPhase, func(ctx *ProcessContext) error {
		if ctx.Chain == nil {
			ctx.Chain = c.db.Chain()
		}
		ctx.Chain = ctx.Chain.Table(c.tableName).Eq("id", ctx.Data["id"])
		return nil
	})
	singleHandler.AddProcessor(Execute, OnPhase, func(ctx *ProcessContext) error {
		result := ctx.Chain.First()
		if result.Error != nil {
			if result.Error.Error() == "sql: no rows in result set" {
				ctx.Data["result"] = nil
				JsonErr(ctx.GinContext, CodeNotFound, "record not found")
				ctx.GinContext.Abort()
				return nil
			}
			return result.Error
		}
		if len(result.Data) == 0 {
			ctx.Data["result"] = nil
			JsonErr(ctx.GinContext, CodeNotFound, "record not found")
			ctx.GinContext.Abort()
			return nil
		}
		ctx.Data["result"] = result.Data[0]
		return nil
	})
	singleHandler.AddProcessor(PostProcess, OnPhase, func(ctx *ProcessContext) error {
		if ctx.Data["result"] == nil {
			return nil
		}
		if result, ok := ctx.Data["result"].(map[string]interface{}); ok {
			ctx.Data["result"] = result
			JsonOk(ctx.GinContext, result)
		} else {
			JsonErr(ctx.GinContext, CodeInternalError, "invalid result format")
		}
		return nil
	})
	c.AddHandler(SINGLE, http.MethodGet, singleHandler)

	// 保存处理器
	saveHandler := NewHandler("/save", http.MethodPost)
	saveHandler.SetDescription("创建新记录")
	saveHandler.AddProcessor(PreProcess, OnPhase, func(ctx *ProcessContext) error {
		t := reflect.TypeOf(c.model)
		if t.Kind() != reflect.Ptr {
			t = t.Elem()
		}
		v := reflect.Indirect(reflect.New(t)).Interface()
		if err := ctx.GinContext.ShouldBindJSON(&v); err != nil {
			JsonErr(ctx.GinContext, CodeInvalid, "invalid request data")
			return err
		}
		ctx.Data = map[string]interface{}{
			"values": v,
		}
		return nil
	})
	saveHandler.AddProcessor(Execute, OnPhase, func(ctx *ProcessContext) error {
		values, ok := ctx.Data["values"].(map[string]interface{})
		if !ok {
			JsonErr(ctx.GinContext, CodeInvalid, "invalid values")
			return fmt.Errorf("invalid values")
		}

		result := c.db.Chain().Table(c.tableName).Values(values).Save()
		if result.Error != nil {
			if strings.Contains(result.Error.Error(), "Duplicate entry") {
				JsonErr(ctx.GinContext, CodeConflict, "domain with same name or identifier already exists")
				return nil
			}
			JsonErr(ctx.GinContext, CodeInternalError, result.Error.Error())
			return result.Error
		}

		id, err := result.LastInsertId()
		if err != nil {
			JsonErr(ctx.GinContext, CodeInternalError, err.Error())
			return err
		}

		// 查询新创建的记录
		queryResult := c.db.Chain().Table(c.tableName).Eq("id", id).First()
		if queryResult.Error != nil {
			JsonErr(ctx.GinContext, CodeInternalError, queryResult.Error.Error())
			return queryResult.Error
		}

		// Map identifier back to domainName in response
		data := queryResult.Data[0]
		data["domainName"] = data["identifier"]
		delete(data, "identifier")

		ctx.Data["result"] = data
		return nil
	})
	saveHandler.AddProcessor(PostProcess, OnPhase, func(ctx *ProcessContext) error {
		JsonOk(ctx.GinContext, ctx.Data["result"])
		return nil
	})
	c.AddHandler(SAVE, http.MethodPost, saveHandler)

	// 更新处理器
	updateHandler := NewHandler("/update/:id", http.MethodPut)
	updateHandler.SetDescription("更新指定记录")
	updateHandler.AddProcessor(PreProcess, OnPhase, func(ctx *ProcessContext) error {
		id := ctx.GinContext.Param("id")
		if id == "" {
			JsonErr(ctx.GinContext, CodeInvalid, "id is required")
			return fmt.Errorf("id is required")
		}
		var data map[string]interface{}
		if err := ctx.GinContext.ShouldBindJSON(&data); err != nil {
			JsonErr(ctx.GinContext, CodeInvalid, "invalid request data")
			return err
		}
		// Validate data types
		if age, ok := data["age"]; ok {
			switch v := age.(type) {
			case float64:
				data["age"] = int(v)
			case string:
				JsonErr(ctx.GinContext, CodeInvalid, "age must be a number")
				return fmt.Errorf("invalid age value")
			}
		}
		ctx.Data = map[string]interface{}{
			"id":     id,
			"values": data,
		}
		return nil
	})
	updateHandler.AddProcessor(Execute, OnPhase, func(ctx *ProcessContext) error {
		values, ok := ctx.Data["values"].(map[string]interface{})
		if !ok {
			return fmt.Errorf("invalid values")
		}
		result := c.db.Chain().Table(c.tableName).Eq("id", ctx.Data["id"]).Update(values)
		if result.Error != nil {
			return result.Error
		}
		// 查询更新后的记录
		queryResult := c.db.Chain().Table(c.tableName).Eq("id", ctx.Data["id"]).First()
		if queryResult.Error != nil {
			return queryResult.Error
		}
		ctx.Data["result"] = queryResult.Data[0]
		return nil
	})
	updateHandler.AddProcessor(PostProcess, OnPhase, func(ctx *ProcessContext) error {
		JsonOk(ctx.GinContext, ctx.Data["result"])
		return nil
	})
	c.AddHandler(UPDATE, http.MethodPut, updateHandler)

	// 删除处理器
	deleteHandler := NewHandler("/delete/:id", http.MethodDelete)
	deleteHandler.SetDescription("删除指定记录")
	deleteHandler.AddProcessor(PreProcess, OnPhase, func(ctx *ProcessContext) error {
		id := ctx.GinContext.Param("id")
		if id == "" {
			return fmt.Errorf("id is required")
		}
		ctx.Data = map[string]interface{}{
			"id": id,
		}
		return nil
	})
	deleteHandler.AddProcessor(Execute, OnPhase, func(ctx *ProcessContext) error {
		// 先查询记录是否存在
		queryResult := c.db.Chain().Table(c.tableName).Eq("id", ctx.Data["id"]).First()
		if queryResult.Error != nil {
			if queryResult.Error.Error() == "sql: no rows in result set" {
				JsonErr(ctx.GinContext, CodeNotFound, "record not found")
				return nil
			}
			return queryResult.Error
		}

		result := c.db.Chain().Table(c.tableName).Eq("id", ctx.Data["id"]).Delete()
		if result.Error != nil {
			return result.Error
		}
		affected, err := result.RowsAffected()
		if err != nil {
			return err
		}
		ctx.Data["result"] = map[string]interface{}{
			"affected": affected,
		}
		return nil
	})
	deleteHandler.AddProcessor(PostProcess, OnPhase, func(ctx *ProcessContext) error {
		JsonOk(ctx.GinContext, ctx.Data["result"])
		return nil
	})
	c.AddHandler(DELETE, http.MethodDelete, deleteHandler)
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
			if sort[0] == '-' {
				ctx.Chain = ctx.Chain.OrderByDesc(sort[1:])
				debugf("Applied descending sort on field: %s", sort[1:])
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

	// 执行SQL
	h.AddProcessor(Execute, OnPhase, func(ctx *ProcessContext) error {
		debugf("Executing SQL query")
		// 获取分页参数
		page := ctx.Data["page"].(int)
		size := ctx.Data["size"].(int)

		// 获取总数
		countChain := dp.crud.db.Chain().Table(dp.crud.tableName)
		if err := ParseQueryConditions(ctx.GinContext, countChain); err != nil {
			debugf("Error parsing query conditions for count: %v", err)
			return err
		}

		// 执行 COUNT 查询
		total, err := countChain.Count()
		if err != nil {
			debugf("Error getting total count: %v", err)
			return err
		}
		debugf("Total count: %d", total)

		// 获取分页数据
		result := ctx.Chain.List()
		if result.Error != nil {
			debugf("Error getting page data: %v", result.Error)
			return result.Error
		}

		// 构建分页响应
		ctx.Data["result"] = map[string]interface{}{
			"page":  page,
			"size":  size,
			"total": total,
			"pages": int(math.Ceil(float64(total) / float64(size))),
			"list":  result.Data,
		}
		debugf("Page result: %+v", ctx.Data["result"])

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
