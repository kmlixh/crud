package crud

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/kmlixh/gom/v4"
	"github.com/kmlixh/gom/v4/define"
)

// Handler CRUD 处理器
type Handler struct {
	db     *gom.DB
	config Config
	table  *Table
}

// NewHandler 创建处理器
func NewHandler(db *gom.DB, config Config) *Handler {
	return &Handler{
		db:     db,
		config: config,
		table:  NewTable(db, config.Model),
	}
}

// List 列表查询
func (h *Handler) List(c *gin.Context) {
	// 获取分页参数
	pageNum, _ := strconv.ParseInt(c.DefaultQuery("pageNum", "1"), 10, 64)
	pageSize, _ := strconv.ParseInt(c.DefaultQuery("pageSize", "10"), 10, 64)

	// 构建查询
	query := h.db.Model(h.table.GetTableName())

	// 应用查询条件
	for paramName, fieldName := range h.config.QueryMapping {
		if value := c.Query(paramName); value != "" {
			// 解析操作符
			op := "eq" // 默认使用等于操作符
			if strings.Contains(paramName, "_") {
				parts := strings.Split(paramName, "_")
				if len(parts) > 1 {
					op = parts[len(parts)-1]
					paramName = strings.Join(parts[:len(parts)-1], "_")
				}
			}

			// 根据操作符构建条件
			switch strings.ToLower(op) {
			case "eq":
				query.Where(define.Eq(fieldName, value))
			case "ne":
				query.Where(define.Ne(fieldName, value))
			case "gt":
				query.Where(define.Gt(fieldName, value))
			case "gte":
				query.Where(define.Ge(fieldName, value))
			case "lt":
				query.Where(define.Lt(fieldName, value))
			case "lte":
				query.Where(define.Le(fieldName, value))
			case "like":
				query.Where(define.Like(fieldName, value))
			case "notlike":
				query.Where(define.NotLike(fieldName, value))
			case "in":
				values := strings.Split(value, ",")
				query.Where(define.In(fieldName, values))
			case "notin":
				values := strings.Split(value, ",")
				query.Where(define.NotIn(fieldName, values))
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
				query.OrderByDesc(field[1:])
			} else {
				query.OrderBy(field)
			}
		}
	}

	// 应用分页
	query.Page(pageNum, pageSize)

	// 执行查询
	var results []map[string]interface{}
	total, err := query.Select(&results, h.config.QueryFields...)
	if err != nil {
		c.JSON(http.StatusOK, Response{
			Code:    500,
			Message: err.Error(),
		})
		return
	}

	// 计算总页数
	totalPages := total / pageSize
	if total%pageSize > 0 {
		totalPages++
	}

	// 返回结果
	c.JSON(http.StatusOK, Response{
		Code: 0,
		Data: PageResult{
			PageNum:    pageNum,
			PageSize:   pageSize,
			Total:      total,
			TotalPages: totalPages,
			Data:       results,
		},
	})
}

// Get 获取单条记录
func (h *Handler) Get(c *gin.Context) {
	// 构建查询
	query := h.db.Model(h.table.GetTableName())

	// 应用主键条件
	for _, pk := range h.config.PrimaryKeys {
		if value := c.Param(pk); value != "" {
			query.Where(define.Eq(pk, value))
		}
	}

	// 执行查询
	var result map[string]interface{}
	_, err := query.Select(&result, h.config.QueryFields...)
	if err != nil {
		c.JSON(http.StatusOK, Response{
			Code:    500,
			Message: err.Error(),
		})
		return
	}

	// 返回结果
	c.JSON(http.StatusOK, Response{
		Code: 0,
		Data: result,
	})
}

// Create 创建记录
func (h *Handler) Create(c *gin.Context) {
	// 获取请求数据
	var data map[string]interface{}
	if err := c.ShouldBindJSON(&data); err != nil {
		c.JSON(http.StatusOK, Response{
			Code:    500,
			Message: err.Error(),
		})
		return
	}

	// 过滤字段
	filteredData := make(map[string]interface{})
	for _, field := range h.config.CreateFields {
		if value, exists := data[field]; exists {
			filteredData[field] = value
		}
	}

	// 执行插入
	result, err := h.db.Model(h.table.GetTableName()).Insert(filteredData)
	if err != nil {
		c.JSON(http.StatusOK, Response{
			Code:    500,
			Message: err.Error(),
		})
		return
	}

	// 返回结果
	c.JSON(http.StatusOK, Response{
		Code: 0,
		Data: result,
	})
}

// Update 更新记录
func (h *Handler) Update(c *gin.Context) {
	// 获取请求数据
	var data map[string]interface{}
	if err := c.ShouldBindJSON(&data); err != nil {
		c.JSON(http.StatusOK, Response{
			Code:    500,
			Message: err.Error(),
		})
		return
	}

	// 构建更新
	query := h.db.Model(h.table.GetTableName())

	// 应用主键条件
	hasPrimaryKey := false
	for _, pk := range h.config.PrimaryKeys {
		if value := c.Param(pk); value != "" {
			query.Where(define.Eq(pk, value))
			hasPrimaryKey = true
		}
	}

	if !hasPrimaryKey {
		c.JSON(http.StatusOK, Response{
			Code:    500,
			Message: "primary key is required",
		})
		return
	}

	// 过滤字段
	filteredData := make(map[string]interface{})
	for _, field := range h.config.UpdateFields {
		if value, exists := data[field]; exists {
			filteredData[field] = value
		}
	}

	// 执行更新
	result, err := query.Update(filteredData)
	if err != nil {
		c.JSON(http.StatusOK, Response{
			Code:    500,
			Message: err.Error(),
		})
		return
	}

	// 返回结果
	c.JSON(http.StatusOK, Response{
		Code: 0,
		Data: result,
	})
}

// Delete 删除记录
func (h *Handler) Delete(c *gin.Context) {
	// 构建删除
	query := h.db.Model(h.table.GetTableName())

	// 应用主键条件
	hasPrimaryKey := false
	for _, pk := range h.config.PrimaryKeys {
		if value := c.Param(pk); value != "" {
			query.Where(define.Eq(pk, value))
			hasPrimaryKey = true
		}
	}

	if !hasPrimaryKey {
		c.JSON(http.StatusOK, Response{
			Code:    500,
			Message: "primary key is required",
		})
		return
	}

	// 执行删除
	result, err := query.Delete()
	if err != nil {
		c.JSON(http.StatusOK, Response{
			Code:    500,
			Message: err.Error(),
		})
		return
	}

	// 返回结果
	c.JSON(http.StatusOK, Response{
		Code: 0,
		Data: result,
	})
}
