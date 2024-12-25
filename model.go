package crud

import (
	"reflect"
	"strings"

	"github.com/kmlixh/gom/v4"
)

// ModelInfo 模型信息
type ModelInfo struct {
	TableName     string            // 表名
	PrimaryKeys   []string          // 主键
	ColumnNames   []string          // 列名
	ColumnMap     map[string]string // 列名映射
	ColumnTypes   map[string]string // 列类型
	AutoIncrement string            // 自增列
}

// GetModelInfo 获取模型信息
func GetModelInfo(model interface{}) *ModelInfo {
	t := reflect.TypeOf(model)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	info := &ModelInfo{
		TableName:   getTableName(t),
		PrimaryKeys: make([]string, 0),
		ColumnNames: make([]string, 0),
		ColumnMap:   make(map[string]string),
		ColumnTypes: make(map[string]string),
	}

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		tag := field.Tag.Get("gom")
		if tag == "" {
			continue
		}

		parts := strings.Split(tag, ",")
		columnName := parts[0]
		if columnName == "" {
			columnName = toSnakeCase(field.Name)
		}

		info.ColumnNames = append(info.ColumnNames, columnName)
		info.ColumnMap[columnName] = field.Name
		info.ColumnTypes[columnName] = field.Type.String()

		for _, part := range parts[1:] {
			switch part {
			case "primary":
				info.PrimaryKeys = append(info.PrimaryKeys, columnName)
			case "auto_increment":
				info.AutoIncrement = columnName
			}
		}
	}

	return info
}

// getTableName 获取表名
func getTableName(t reflect.Type) string {
	if t.Kind() != reflect.Struct {
		return ""
	}

	// 尝试从 TableName 方法获取
	method, exists := t.MethodByName("TableName")
	if exists {
		result := method.Func.Call([]reflect.Value{reflect.New(t)})
		if len(result) > 0 {
			if tableName, ok := result[0].Interface().(string); ok {
				return tableName
			}
		}
	}

	// 使用结构体名转换为蛇形命名
	return toSnakeCase(t.Name())
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

// Table 表操作
type Table struct {
	db    *gom.DB
	model interface{}
	info  *ModelInfo
}

// NewTable 创建表操作
func NewTable(db *gom.DB, model interface{}) *Table {
	return &Table{
		db:    db,
		model: model,
		info:  GetModelInfo(model),
	}
}

// GetColumnNames 获取列名
func (t *Table) GetColumnNames() []string {
	return t.info.ColumnNames
}

// GetPrimaryKeys 获取主键
func (t *Table) GetPrimaryKeys() []string {
	return t.info.PrimaryKeys
}

// GetColumnMap 获取列名映射
func (t *Table) GetColumnMap() map[string]string {
	return t.info.ColumnMap
}

// GetTableName 获取表名
func (t *Table) GetTableName() string {
	return t.info.TableName
}

// GetAutoIncrement 获取自增列
func (t *Table) GetAutoIncrement() string {
	return t.info.AutoIncrement
}
