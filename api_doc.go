package crud

import (
	"sync"
)

// APIParam 描述API参数
type APIParam struct {
	Name        string    `json:"name"`        // 参数名称
	Type        string    `json:"type"`        // 参数类型
	Required    bool      `json:"required"`    // 是否必须
	Description string    `json:"description"` // 参数说明
	Location    string    `json:"location"`    // 参数位置(query/body/path)
	Items       *APIParam `json:"items"`       // 用于数组类型的子项定义

}

// APIResponse 描述API响应
type APIResponse struct {
	Type        string    `json:"type"`             // 响应类型
	Description string    `json:"description"`      // 响应说明
	Schema      *APIParam `json:"schema,omitempty"` // 响应结构
}

// APIDoc 描述单个API文档
type APIDoc struct {
	Name        string                 `json:"name"`        // 接口名称
	Path        string                 `json:"path"`        // 接口路径
	Method      string                 `json:"method"`      // HTTP方法
	Description string                 `json:"description"` // 接口说明
	Group       string                 `json:"group"`       // 接口分组
	Parameters  []APIParam             `json:"parameters"`  // 入参列表
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
