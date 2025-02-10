package crud

import (
	"github.com/gin-gonic/gin"
)

// RegisterAPIDocHandler 注册API文档处理器
func RegisterAPIDocHandler(engine *gin.Engine, path string) {
	if path == "" {
		path = "/api/doc"
	}
	engine.GET(path, GetAPIDoc)
	engine.GET(path+"/:group", GetGroupAPIDoc)
}

// GetAPIDoc 获取所有API文档
func GetAPIDoc(c *gin.Context) {
	RenderOk(c, globalAPIRegistry.GetAPIs())
}

// GetGroupAPIDoc 获取分组API文档
func GetGroupAPIDoc(c *gin.Context) {
	group := c.Param("group")
	apis := globalAPIRegistry.GetAPIsByGroup(group)
	if apis == nil {
		RenderErr2(c, 404, "API group not found")
		return
	}
	RenderOk(c, apis)
}
