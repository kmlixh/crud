package crud

// Config CRUD 配置
type Config struct {
	// 路由前缀
	PathPrefix string
	// 模型
	Model interface{}
	// 主键字段
	PrimaryKeys []string
	// 查询字段
	QueryFields []string
	// 可更新字段
	UpdateFields []string
	// 可插入字段
	CreateFields []string
	// 排除字段
	ExcludeFields []string
	// 查询条件映射
	QueryMapping map[string]string
}

// GetDefaultConfig 获取默认配置
func GetDefaultConfig(pathPrefix string, model interface{}) Config {
	table := NewTable(nil, model)
	columnNames := table.GetColumnNames()

	return Config{
		PathPrefix:    pathPrefix,
		Model:         model,
		PrimaryKeys:   table.GetPrimaryKeys(),
		QueryFields:   columnNames,
		UpdateFields:  columnNames,
		CreateFields:  columnNames,
		ExcludeFields: []string{},
		QueryMapping:  GetDefaultQueryMapping(columnNames),
	}
}

// GetDefaultQueryMapping 获取默认的查询映射
func GetDefaultQueryMapping(fields []string) map[string]string {
	mapping := make(map[string]string)
	for _, field := range fields {
		// 基本匹配
		mapping[field] = field
		// 模糊匹配
		mapping[field+"_like"] = field
		// 范围查询
		mapping[field+"_gt"] = field
		mapping[field+"_gte"] = field
		mapping[field+"_lt"] = field
		mapping[field+"_lte"] = field
		// IN 查询
		mapping[field+"_in"] = field
		mapping[field+"_not_in"] = field
	}
	return mapping
}

// ConfigOption 配置选项函数
type ConfigOption func(*Config)

// WithPrimaryKeys 设置主键
func WithPrimaryKeys(keys ...string) ConfigOption {
	return func(c *Config) {
		c.PrimaryKeys = keys
	}
}

// WithQueryFields 设置查询字段
func WithQueryFields(fields ...string) ConfigOption {
	return func(c *Config) {
		c.QueryFields = fields
	}
}

// WithUpdateFields 设置可更新字段
func WithUpdateFields(fields ...string) ConfigOption {
	return func(c *Config) {
		c.UpdateFields = fields
	}
}

// WithCreateFields 设置可创建字段
func WithCreateFields(fields ...string) ConfigOption {
	return func(c *Config) {
		c.CreateFields = fields
	}
}

// WithExcludeFields 设置排除字段
func WithExcludeFields(fields ...string) ConfigOption {
	return func(c *Config) {
		c.ExcludeFields = fields
	}
}

// WithQueryMapping 设置查询映射
func WithQueryMapping(mapping map[string]string) ConfigOption {
	return func(c *Config) {
		c.QueryMapping = mapping
	}
}

// NewConfig 创建新配置
func NewConfig(pathPrefix string, model interface{}, opts ...ConfigOption) Config {
	// 获取默认配置
	config := GetDefaultConfig(pathPrefix, model)

	// 应用选项
	for _, opt := range opts {
		opt(&config)
	}

	return config
}

// PageResult 分页结果
type PageResult struct {
	PageNum    int64       `json:"pageNum"`    // 当前页码
	PageSize   int64       `json:"pageSize"`   // 每页大小
	Total      int64       `json:"total"`      // 总记录数
	TotalPages int64       `json:"totalPages"` // 总页数
	Data       interface{} `json:"data"`       // 数据列表
}

// Response API 响应
type Response struct {
	Code    int         `json:"code"`    // 状态码
	Message string      `json:"message"` // 消息
	Data    interface{} `json:"data"`    // 数据
}

// SortOrder 排序
type SortOrder struct {
	Column string // 字段名
	Desc   bool   // 是否降序
}

// Pagination 分页
type Pagination struct {
	PageNum  int64 // 页码
	PageSize int64 // 每页大小
}
