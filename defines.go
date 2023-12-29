package AutoCrudGo

import "github.com/gin-gonic/gin"

type RouteHandler struct {
	Path       string
	HttpMethod string
	Handlers   []gin.HandlerFunc
}

func AutoHandle(name string, routes gin.IRoutes, handlers []RouteHandler) {
	for _, handler := range handlers {
		if handler.HttpMethod != "Any" {
			routes.Handle(handler.HttpMethod, name+"/"+handler.Path, handler.Handlers...)
		} else {
			routes.Any(name+"/"+handler.Path, handler.Handlers...)
		}
	}
}
