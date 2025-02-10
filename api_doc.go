package crud

import (
	"reflect"
	"sync"
	"time"

	"github.com/kmlixh/gom/v4/define"
)

// APIParam 描述API参数
type ApiProperty struct {
	Name        string        `json:"name"`        // 参数名称
	Type        string        `json:"type"`        // 参数类型
	Required    bool          `json:"required"`    // 是否必须
	Description string        `json:"description"` // 参数说明
	Location    string        `json:"location"`    // 参数位置(query/body/path)
	Fields      []ApiProperty `json:"fields"`      // 用于对象类型的子属性定义
}

// GeneratePageInfoApiProperty 生成描述 PageInfo 结构体的 ApiProperty 对象
func GeneratePageInfoApiProperty(listProperties []ApiProperty) ApiProperty {
	return ApiProperty{
		Name:        "PageInfo",
		Type:        "object",
		Description: "分页信息",
		Fields: []ApiProperty{
			{
				Name:        "pageNum",
				Type:        "integer",
				Required:    true,
				Description: "当前页码",
			},
			{
				Name:        "pageSize",
				Type:        "integer",
				Required:    true,
				Description: "每页大小",
			},
			{
				Name:        "total",
				Type:        "integer",
				Required:    true,
				Description: "总记录数",
			},
			{
				Name:        "pages",
				Type:        "integer",
				Required:    true,
				Description: "总页数",
			},
			{
				Name:        "hasPrev",
				Type:        "boolean",
				Required:    true,
				Description: "是否有上一页",
			},
			{
				Name:        "hasNext",
				Type:        "boolean",
				Required:    true,
				Description: "是否有下一页",
			},
			{
				Name:        "list",
				Type:        "array",
				Required:    true,
				Description: "当前页数据",
				Fields:      listProperties,
			},
			{
				Name:        "isFirstPage",
				Type:        "boolean",
				Required:    true,
				Description: "是否是第一页",
			},
			{
				Name:        "isLastPage",
				Type:        "boolean",
				Required:    true,
				Description: "是否是最后页",
			},
		},
	}
}

// GenerateColumnInfoApiProperty 生成描述 ColumnInfo 结构体的 ApiProperty 对象
func GenerateColumnInfoApiProperty(cols []define.ColumnInfo) []ApiProperty {
	var properties []ApiProperty
	for _, col := range cols {
		properties = append(properties, ApiProperty{
			Name:        col.Name,
			Type:        col.TypeName,
			Required:    !col.IsNullable,
			Description: col.Comment,
		})
	}
	return properties
}

// GenerateApiPropertiesFromStruct 生成描述结构体属性的 ApiProperty 数组
func GenerateApiPropertiesFromStruct(instance interface{}) []ApiProperty {
	var properties []ApiProperty
	val := reflect.ValueOf(instance)
	typ := reflect.TypeOf(instance)

	// 确保传入的是结构体或结构体指针
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
		val = val.Elem()
	}
	if typ.Kind() != reflect.Struct {
		return properties
	}

	// 遍历结构体的所有字段
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		fieldValue := val.Field(i)

		// 获取字段的标签信息
		jsonTag := field.Tag.Get("json")
		if jsonTag == "" {
			jsonTag = field.Name
		}

		// 确定字段类型
		var fieldType string
		switch fieldValue.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			fieldType = "integer"
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			fieldType = "integer"
		case reflect.Float32, reflect.Float64:
			fieldType = "number"
		case reflect.Bool:
			fieldType = "boolean"
		case reflect.String:
			fieldType = "string"
		case reflect.Slice, reflect.Array:
			fieldType = "array"
		case reflect.Struct:
			if fieldValue.Type() == reflect.TypeOf(time.Time{}) {
				fieldType = "string"
			} else {
				fieldType = "object"
			}
		default:
			fieldType = "string"
		}

		// 创建 ApiProperty 对象
		properties = append(properties, ApiProperty{
			Name:        jsonTag,
			Type:        fieldType,
			Required:    !field.Anonymous && fieldValue.CanSet(),
			Description: field.Tag.Get("description"),
		})
	}

	return properties
}

// APIResponse 描述API响应
type APIResponse struct {
	Description string               `json:"description"`       // 响应说明
	Content     map[string]MediaType `json:"content"`           // 响应内容
	Headers     map[string]Header    `json:"headers,omitempty"` // 响应头
}

// NewCodeMsgResponse 创建一个包含CodeMsg结构体的APIResponse
func NewCodeMsgResponse(description string, code int, msg string) APIResponse {
	return APIResponse{
		Description: description,
		Content: map[string]MediaType{
			"application/json": {
				Schema: &ApiProperty{
					Type: "object",
					Fields: []ApiProperty{
						{
							Name: "code",
							Type: "integer",
						},
						{
							Name: "msg",
							Type: "string",
						},
					},
				},
				Examples: map[string]interface{}{
					"example": map[string]interface{}{
						"code": code,
						"msg":  msg,
					},
				},
			},
		},
	}
}

// MediaType 描述响应内容的媒体类型
type MediaType struct {
	Schema   *ApiProperty           `json:"schema,omitempty"`   // 响应结构
	Examples map[string]interface{} `json:"examples,omitempty"` // 示例
}

// Header 描述响应头
type Header struct {
	Description string `json:"description"` // 响应头说明
	Type        string `json:"type"`        // 响应头类型
}

// APIDoc 描述单个API文档
type APIDoc struct {
	Name        string                 `json:"name"`        // 接口名称
	Path        string                 `json:"path"`        // 接口路径
	Method      string                 `json:"method"`      // HTTP方法
	Description string                 `json:"description"` // 接口说明
	Group       string                 `json:"group"`       // 接口分组
	Parameters  []ApiProperty          `json:"parameters"`  // 入参列表
	Response    APIResponse            `json:"response"`    // 响应说明
	Metadata    map[string]interface{} `json:"metadata"`    // 扩展元数据
}

// APIRegistry 全局API注册表
type APIRegistry struct {
	sync.RWMutex
	apis map[string][]APIDoc // group -> apis
}

var (
	globalAPIRegistry = &APIRegistry{
		apis: make(map[string][]APIDoc),
	}
)

// RegisterAPI 注册API文档
func (r *APIRegistry) RegisterAPI(group string, doc APIDoc) {
	r.Lock()
	defer r.Unlock()

	if _, exists := r.apis[group]; !exists {
		r.apis[group] = make([]APIDoc, 0)
	}
	r.apis[group] = append(r.apis[group], doc)
}

// GetAPIs 获取所有API文档
func (r *APIRegistry) GetAPIs() map[string][]APIDoc {
	r.RLock()
	defer r.RUnlock()

	// 创建副本避免并发问题
	result := make(map[string][]APIDoc)
	for k, v := range r.apis {
		result[k] = append([]APIDoc{}, v...)
	}
	return result
}

// GetAPIsByGroup 获取指定分组的API文档
func (r *APIRegistry) GetAPIsByGroup(group string) []APIDoc {
	r.RLock()
	defer r.RUnlock()

	if apis, exists := r.apis[group]; exists {
		return append([]APIDoc{}, apis...)
	}
	return nil
}
