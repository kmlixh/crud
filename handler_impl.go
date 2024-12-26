package crud

import (
	"errors"
	"net/http"
	"reflect"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/kmlixh/gom/v4"
)

// handlerImpl CRUD 处理器实现
type handlerImpl struct {
	db         *gom.DB
	model      interface{}
	tableName  string
	opts       Options
	fieldNames []string
}

// newHandler 创建处理器
func newHandler(db *gom.DB, model interface{}, opts Options) Handler {
	// 获取表名
	tableName := getTableName(model)
	if tableName == "" {
		panic("invalid model: cannot get table name")
	}

	// 获取字段名
	fieldNames := getFieldNames(model)
	if len(fieldNames) == 0 {
		panic("invalid model: no fields found")
	}

	// 设置默认值
	if opts.PrimaryKey == "" {
		opts.PrimaryKey = "id"
	}

	return &handlerImpl{
		db:         db,
		model:      model,
		tableName:  tableName,
		opts:       opts,
		fieldNames: fieldNames,
	}
}

// List 列表查询
func (h *handlerImpl) List(c *gin.Context) {
	// 获取分页参数
	pageNum, _ := strconv.ParseInt(c.DefaultQuery("pageNum", "1"), 10, 64)
	pageSize, _ := strconv.ParseInt(c.DefaultQuery("pageSize", "10"), 10, 64)

	// 构建查询
	query := h.db.Chain().From(h.model)

	// 应用查询条件
	for _, field := range h.getQueryFields() {
		if value := c.Query(field); value != "" {
			// 处理特殊查询
			if strings.HasSuffix(field, "_like") {
				field = strings.TrimSuffix(field, "_like")
				query = query.Where(field, "LIKE", "%"+value+"%")
			} else if strings.HasSuffix(field, "_gt") {
				field = strings.TrimSuffix(field, "_gt")
				query = query.Where(field, ">", value)
			} else if strings.HasSuffix(field, "_gte") {
				field = strings.TrimSuffix(field, "_gte")
				query = query.Where(field, ">=", value)
			} else if strings.HasSuffix(field, "_lt") {
				field = strings.TrimSuffix(field, "_lt")
				query = query.Where(field, "<", value)
			} else if strings.HasSuffix(field, "_lte") {
				field = strings.TrimSuffix(field, "_lte")
				query = query.Where(field, "<=", value)
			} else {
				query = query.Where(field, "=", value)
			}
		}
	}

	// 应用排序
	if orderBy := c.Query("orderBy"); orderBy != "" {
		fields := strings.Split(orderBy, ",")
		for _, field := range fields {
			field = strings.TrimSpace(field)
			if field == "" {
				continue
			}

			if strings.HasPrefix(field, "-") {
				query = query.OrderByDesc(field[1:])
			} else {
				query = query.OrderBy(field)
			}
		}
	}

	// 执行查询
	query = query.Limit(int(pageSize)).Offset(int((pageNum - 1) * pageSize))
	result := query.List()
	if result.Error() != nil {
		c.JSON(http.StatusOK, fail(result.Error()))
		return
	}

	var results []map[string]interface{}
	if err := result.Into(&results); err != nil {
		c.JSON(http.StatusOK, fail(err))
		return
	}

	// 计算总页数
	total := int64(result.Size())
	totalPages := total / pageSize
	if total%pageSize > 0 {
		totalPages++
	}

	// 返回结果
	c.JSON(http.StatusOK, success(PageResult{
		PageNum:    pageNum,
		PageSize:   pageSize,
		Total:      total,
		TotalPages: totalPages,
		Data:       results,
	}))
}

// Get 获取单条记录
func (h *handlerImpl) Get(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusOK, fail(errors.New("id is required")))
		return
	}

	// 执行查询
	result := h.db.Chain().From(h.model).Where(h.opts.PrimaryKey, "=", id).One()
	if result.Error() != nil {
		c.JSON(http.StatusOK, fail(result.Error()))
		return
	}

	var data map[string]interface{}
	if err := result.Into(&data); err != nil {
		c.JSON(http.StatusOK, fail(err))
		return
	}

	// 返回结果
	c.JSON(http.StatusOK, success(data))
}

// Create 创建记录
func (h *handlerImpl) Create(c *gin.Context) {
	// 获取请求数据
	var data map[string]interface{}
	if err := c.ShouldBindJSON(&data); err != nil {
		c.JSON(http.StatusOK, fail(err))
		return
	}

	// 过滤字段
	filteredData := make(map[string]interface{})
	for _, field := range h.getCreateFields() {
		if value, exists := data[field]; exists {
			filteredData[field] = value
		}
	}

	// 执行插入
	result, err := h.db.Chain().From(h.model).Values(filteredData).Save()
	if err != nil {
		c.JSON(http.StatusOK, fail(err))
		return
	}

	// 返回结果
	c.JSON(http.StatusOK, success(result))
}

// Update 更新记录
func (h *handlerImpl) Update(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusOK, fail(errors.New("id is required")))
		return
	}

	// 获取请求数据
	var data map[string]interface{}
	if err := c.ShouldBindJSON(&data); err != nil {
		c.JSON(http.StatusOK, fail(err))
		return
	}

	// 过滤字段
	chain := h.db.Chain().From(h.model).Where(h.opts.PrimaryKey, "=", id)
	for _, field := range h.getUpdateFields() {
		if value, exists := data[field]; exists {
			chain = chain.Set(field, value)
		}
	}

	// 执行更新
	result, err := chain.Update()
	if err != nil {
		c.JSON(http.StatusOK, fail(err))
		return
	}

	// 返回结果
	c.JSON(http.StatusOK, success(result))
}

// Delete 删除记录
func (h *handlerImpl) Delete(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusOK, fail(errors.New("id is required")))
		return
	}

	// 执行删除
	result, err := h.db.Chain().From(h.model).Where(h.opts.PrimaryKey, "=", id).Delete()
	if err != nil {
		c.JSON(http.StatusOK, fail(err))
		return
	}

	// 返回结果
	c.JSON(http.StatusOK, success(result))
}

// getQueryFields 获取可查询字段
func (h *handlerImpl) getQueryFields() []string {
	if len(h.opts.QueryFields) > 0 {
		return h.opts.QueryFields
	}
	return h.getAvailableFields()
}

// getUpdateFields 获取可更新字段
func (h *handlerImpl) getUpdateFields() []string {
	if len(h.opts.UpdateFields) > 0 {
		return h.opts.UpdateFields
	}
	return h.getAvailableFields()
}

// getCreateFields 获取可创建字段
func (h *handlerImpl) getCreateFields() []string {
	if len(h.opts.CreateFields) > 0 {
		return h.opts.CreateFields
	}
	return h.getAvailableFields()
}

// getAvailableFields 获取可用字段
func (h *handlerImpl) getAvailableFields() []string {
	if len(h.opts.ExcludeFields) == 0 {
		return h.fieldNames
	}

	excludeMap := make(map[string]bool)
	for _, field := range h.opts.ExcludeFields {
		excludeMap[field] = true
	}

	var fields []string
	for _, field := range h.fieldNames {
		if !excludeMap[field] {
			fields = append(fields, field)
		}
	}
	return fields
}

// getTableName 获取表名
func getTableName(model interface{}) string {
	t := reflect.TypeOf(model)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	// 尝试从 TableName 方法获取
	if m, ok := model.(interface{ TableName() string }); ok {
		return m.TableName()
	}

	// 使用结构体名转换为蛇形命名
	return toSnakeCase(t.Name())
}

// getFieldNames 获取字段名
func getFieldNames(model interface{}) []string {
	t := reflect.TypeOf(model)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	var fields []string
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		tag := field.Tag.Get("gom")
		if tag == "" {
			continue
		}

		parts := strings.Split(tag, ",")
		if len(parts) > 0 && parts[0] != "" {
			fields = append(fields, parts[0])
		} else {
			fields = append(fields, toSnakeCase(field.Name))
		}
	}
	return fields
}

// toSnakeCase 转换为蛇形命名
func toSnakeCase(s string) string {
	var result []rune
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result = append(result, '_')
		}
		result = append(result, r)
	}
	return strings.ToLower(string(result))
}
