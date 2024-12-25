package crud

import (
	"github.com/gin-gonic/gin"
	"github.com/kmlixh/gom/v4"
)

// RegisterCrud 注册 CRUD 路由
func RegisterCrud(router gin.IRouter, db *gom.DB, config Config) error {
	// 验证配置
	if err := validateConfig(config); err != nil {
		return err
	}

	// 创建处理器
	handler := NewHandler(db, config)

	// 注册路由
	group := router.Group(config.PathPrefix)
	{
		// 列表查询
		group.GET("", handler.List)
		// 获取单条记录
		group.GET("/:id", handler.Get)
		// 创建记录
		group.POST("", handler.Create)
		// 更新记录
		group.PUT("/:id", handler.Update)
		group.PATCH("/:id", handler.Update) // 支持 PATCH 方法
		// 删除记录
		group.DELETE("/:id", handler.Delete)
	}

	return nil
}

// RegisterCrudByStruct 通过结构体注册 CRUD 路由
func RegisterCrudByStruct(router gin.IRouter, db *gom.DB, model interface{}, prefix string) error {
	// 获取结构体信息
	table := NewTable(db, model)
	if table == nil {
		return ErrInvalidConfig
	}

	// 构建配置
	config := Config{
		PathPrefix:    prefix,
		Model:         model,
		PrimaryKeys:   table.GetPrimaryKeys(),
		QueryFields:   table.GetColumnNames(),
		UpdateFields:  table.GetColumnNames(),
		CreateFields:  table.GetColumnNames(),
		ExcludeFields: []string{},
		QueryMapping:  make(map[string]string),
	}

	// 注册路由
	return RegisterCrud(router, db, config)
}

// RegisterCrudByTable 通过表名注册 CRUD 路由
func RegisterCrudByTable(router gin.IRouter, db *gom.DB, tableName string, prefix string) error {
	// 构建配置
	config := Config{
		PathPrefix:    prefix,
		Model:         tableName,
		PrimaryKeys:   []string{"id"}, // 默认使用 id 作为主键
		QueryFields:   []string{"*"},  // 查询所有字段
		UpdateFields:  []string{"*"},  // 更新所有字段
		CreateFields:  []string{"*"},  // 插入所有字段
		ExcludeFields: []string{},
		QueryMapping:  make(map[string]string),
	}

	// 注册路由
	return RegisterCrud(router, db, config)
}

// validateConfig 验证配置
func validateConfig(config Config) error {
	if config.PathPrefix == "" {
		return ErrInvalidConfig
	}
	if config.Model == nil {
		return ErrInvalidConfig
	}
	if len(config.PrimaryKeys) == 0 {
		config.PrimaryKeys = []string{"id"} // 默认使用 id 作为主键
	}
	if len(config.QueryFields) == 0 {
		config.QueryFields = []string{"*"} // 默认查询所有字段
	}
	return nil
}
