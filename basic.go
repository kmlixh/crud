package crud

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// 预定义响应码
const (
	CodeSuccess  = 200 // 成功
	CodeError    = 500 // 服务器错误
	CodeInvalid  = 400 // 请求无效
	CodeNotFound = 404 // 记录不存在
)

// 预定义消息
const (
	MsgSuccess = "success" // 成功
	MsgError   = "error"   // 错误
)

// CodeMsg 统一响应结构
type CodeMsg struct {
	Code    int         `json:"code"`           // 响应码
	Message string      `json:"message"`        // 响应消息
	Data    interface{} `json:"data,omitempty"` // 响应数据，可选
}

// JsonOk 返回成功响应
func JsonOk(c *gin.Context, data interface{}) {
	if _, exists := c.Get("response_sent"); exists {
		return
	}
	c.JSON(http.StatusOK, CodeMsg{
		Code:    CodeSuccess,
		Message: MsgSuccess,
		Data:    data,
	})
	c.Set("response_sent", true)
}

// JsonErr 返回错误响应
func JsonErr(c *gin.Context, code int, message string) {
	if _, exists := c.Get("response_sent"); exists {
		return
	}
	var httpStatus int
	switch code {
	case CodeInvalid:
		httpStatus = http.StatusBadRequest
	case CodeError:
		httpStatus = http.StatusInternalServerError
	case CodeNotFound:
		httpStatus = http.StatusNotFound
	default:
		httpStatus = http.StatusInternalServerError
	}
	c.JSON(httpStatus, CodeMsg{
		Code:    code,
		Message: message,
	})
	c.Set("response_sent", true)
}

// Json 直接返回对象
func Json(c *gin.Context, obj interface{}) {
	c.JSON(http.StatusOK, obj)
}

// CodeMsgFunc 返回自定义响应
func CodeMsgFunc(c *gin.Context, code int, message string, data interface{}) {
	var httpStatus int
	switch code {
	case CodeSuccess:
		httpStatus = http.StatusOK
	case CodeInvalid:
		httpStatus = http.StatusBadRequest
	case CodeError:
		httpStatus = http.StatusInternalServerError
	default:
		httpStatus = http.StatusOK
	}
	c.JSON(httpStatus, CodeMsg{
		Code:    code,
		Message: message,
		Data:    data,
	})
}

var AllowList = make(map[string]bool)

// Cors 跨域中间件
func Cors() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")
		result, ok := AllowList[origin]
		if ok && result {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Access-Control-Allow-Headers", "Content-Type, AccessToken, X-CSRF-Token, Authorization, Token,token,target,code")
			c.Header("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
			c.Header("Access-Control-Expose-Headers", "Content-Length, Access-Control-Allow-Origin, Access-Control-Allow-Headers, Content-Type")
			c.Header("Access-Control-Allow-Credentials", "true")
		}
		method := c.Request.Method
		if method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
		}
		// 允许放行OPTIONS请求

		c.Next()
	}
}
