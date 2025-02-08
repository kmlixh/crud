package crud

import (
	"sync"

	"github.com/gin-gonic/gin"
)

// SwaggerInfo 存储API文档信息
type SwaggerInfo struct {
	Openapi    string                 `json:"openapi"`
	Info       map[string]interface{} `json:"info"`
	Paths      map[string]PathItem    `json:"paths"`
	Components Components             `json:"components"`
}

// PathItem OpenAPI路径项
type PathItem struct {
	Get     *Operation `json:"get,omitempty"`
	Post    *Operation `json:"post,omitempty"`
	Put     *Operation `json:"put,omitempty"`
	Delete  *Operation `json:"delete,omitempty"`
	Options *Operation `json:"options,omitempty"`
}

// Operation OpenAPI操作
type Operation struct {
	Summary     string              `json:"summary"`
	Description string              `json:"description"`
	Parameters  []Parameter         `json:"parameters,omitempty"`
	RequestBody *RequestBody        `json:"requestBody,omitempty"`
	Responses   map[string]Response `json:"responses"`
	Tags        []string            `json:"tags"`
}

// Parameter OpenAPI参数
type Parameter struct {
	Name        string `json:"name"`
	In          string `json:"in"` // query, path, header, cookie
	Description string `json:"description"`
	Required    bool   `json:"required"`
	Schema      Schema `json:"schema"`
}

// RequestBody OpenAPI请求体
type RequestBody struct {
	Description string               `json:"description"`
	Required    bool                 `json:"required"`
	Content     map[string]MediaType `json:"content"`
}

// Response OpenAPI响应
type Response struct {
	Description string               `json:"description"`
	Content     map[string]MediaType `json:"content,omitempty"`
}

// MediaType OpenAPI媒体类型
type MediaType struct {
	Schema Schema `json:"schema"`
}

// Schema OpenAPI模式
type Schema struct {
	Type       string            `json:"type,omitempty"`
	Properties map[string]Schema `json:"properties,omitempty"`
	Items      *Schema           `json:"items,omitempty"`
	Ref        string            `json:"$ref,omitempty"`
}

// Components OpenAPI组件
type Components struct {
	Schemas map[string]Schema `json:"schemas"`
}

// 全局Swagger信息存储
var (
	swaggerInstance *SwaggerInfo
	swaggerMutex    sync.RWMutex
)

// initSwagger 初始化Swagger文档
func initSwagger() {
	swaggerMutex.Lock()
	defer swaggerMutex.Unlock()

	if swaggerInstance == nil {
		swaggerInstance = &SwaggerInfo{
			Openapi: "3.0.0",
			Info: map[string]interface{}{
				"title":       "AutoCrudGo API",
				"description": "基于Gin和Gom的CRUD API框架",
				"version":     "1.0.0",
			},
			Paths:      make(map[string]PathItem),
			Components: Components{Schemas: make(map[string]Schema)},
		}
	}
}

// RegisterSwaggerEndpoint 注册Swagger文档端点
func RegisterSwaggerEndpoint(engine *gin.Engine, path string) {
	initSwagger()

	// 默认路径为/swagger.json
	if path == "" {
		path = "/swagger.json"
	}

	// 注册获取Swagger文档的路由
	engine.GET(path, func(c *gin.Context) {
		swaggerMutex.RLock()
		defer swaggerMutex.RUnlock()
		c.JSON(200, swaggerInstance)
	})
}

// addPath 添加API路径信息
func addPath(path string, method string, operation Operation) {
	swaggerMutex.Lock()
	defer swaggerMutex.Unlock()

	pathItem, exists := swaggerInstance.Paths[path]
	if !exists {
		pathItem = PathItem{}
	}

	switch method {
	case "GET":
		pathItem.Get = &operation
	case "POST":
		pathItem.Post = &operation
	case "PUT":
		pathItem.Put = &operation
	case "DELETE":
		pathItem.Delete = &operation
	case "OPTIONS":
		pathItem.Options = &operation
	}

	swaggerInstance.Paths[path] = pathItem
}

// addSchema 添加模型Schema
func addSchema(name string, schema Schema) {
	swaggerMutex.Lock()
	defer swaggerMutex.Unlock()

	swaggerInstance.Components.Schemas[name] = schema
}
