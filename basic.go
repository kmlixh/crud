package AutoCrudGo

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type CodeMsg map[string]interface{}

func Ok(data ...interface{}) CodeMsg {
	c := RawCodeMsg(0, "ok", nil)
	if data != nil && len(data) >= 1 {
		if len(data) == 1 {
			c.SetData(data[0])
		} else {
			c.SetData(data)
		}
	}
	return c
}

func (c CodeMsg) Code() int {
	return c["code"].(int)
}
func (c CodeMsg) SetCode(code int) CodeMsg {
	c["code"] = code
	return c
}
func (c CodeMsg) Msg() string {
	return c["msg"].(string)
}
func (c CodeMsg) SetMsg(msg string) CodeMsg {
	c["msg"] = msg
	return c
}
func (c CodeMsg) Data() int {
	return c["data"].(int)
}
func (c CodeMsg) SetData(data interface{}) CodeMsg {
	c.Set("data", data)
	return c
}
func (c CodeMsg) Set(name string, data interface{}) CodeMsg {
	c[name] = data
	return c
}
func RawCodeMsg(code int, msg string, data interface{}) CodeMsg {
	codeMsg := CodeMsg{}
	codeMsg["code"] = code
	codeMsg["msg"] = msg
	if data != nil {
		codeMsg["data"] = data
	}
	return codeMsg
}

func Err() CodeMsg {
	return RawCodeMsg(-1, "error", nil)
}

func Err2(code int, msg string) CodeMsg {
	return RawCodeMsg(code, msg, nil)
}
func RenderJson(c *gin.Context, data interface{}) {
	c.JSON(200, data)
}
func RenderOk(c *gin.Context, data ...interface{}) {
	c.JSON(200, Ok(data...))
}
func RenderErr(c *gin.Context) {
	c.JSON(200, Err())
}
func RenderErrs(c *gin.Context, er error) {
	c.JSON(200, Err2(500, er.Error()))
}
func RenderErr2(c *gin.Context, code int, msg string) {
	c.JSON(200, Err2(code, msg))
}

func Cors(allowList map[string]bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		if origin := c.Request.Header.Get("Origin"); allowList[origin] {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Access-Control-Allow-Headers", "Content-Type, AccessToken, X-CSRF-Token, Authorization, Token,token")
			c.Header("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
			c.Header("Access-Control-Expose-Headers", "Content-Length, Access-Control-Allow-Origin, Access-Control-Allow-Headers, Content-Type")
			c.Header("Access-Control-Allow-Credentials", "true")
		}

		// 允许放行OPTIONS请求
		method := c.Request.Method
		if method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
		}
		c.Next()
	}
}

type Server struct {
	server *http.Server
	engine *gin.Engine
}

func NewServer() *Server {
	engine := gin.Default()
	return NewServer2(engine, ":80", 60*time.Second, 30*time.Second, 1<<20)
}

func NewServer2(engine *gin.Engine, addr string, readTimeout time.Duration, writeTimeout time.Duration, maxHeaderBytes int) *Server {
	return &Server{&http.Server{
		Addr:           addr,
		Handler:        engine,
		ReadTimeout:    readTimeout,
		WriteTimeout:   writeTimeout,
		MaxHeaderBytes: maxHeaderBytes,
	}, engine}
}
func (s Server) SetAddr(addr string) {
	s.server.Addr = addr
}
func (s Server) SetReadTimeout(time time.Duration) {
	s.server.ReadTimeout = time
}
func (s Server) SetWriteTimeout(time time.Duration) {
	s.server.WriteTimeout = time
}
func (s Server) SetMaxHeaderBytes(max int) {
	s.server.MaxHeaderBytes = max
}
func (s Server) SetHttpServer(server *http.Server) {
	s.server = server
	s.server.Handler = s.engine
}
func (s Server) SetEngine(engine *gin.Engine) {
	s.engine = engine
	s.server.Handler = s.engine
}

func (s Server) getEngine() *gin.Engine {
	return s.engine
}
func (s Server) getServer() *http.Server {
	return s.server
}
func (s Server) ListenAndServe() error {
	return s.server.ListenAndServe()
}

func GetConditionMapFromRst(c *gin.Context) (map[string]interface{}, error) {
	var maps map[string]interface{}
	var er error
	if c.Request.Method == "POST" {
		contentType := c.GetHeader("Content-Type")
		if strings.Contains(contentType, "application/x-www-form-urlencoded") {
			maps = make(map[string]interface{})
			er = c.Request.ParseForm()
			if er != nil {
				return nil, er
			}
			values := c.Request.Form
			for k, v := range values {
				if len(v) == 1 {
					maps[k] = v[0]
				} else {
					maps[k] = v
				}
			}

		} else if strings.Contains(contentType, "application/json") {
			bbs, er := io.ReadAll(c.Request.Body)
			if er != nil {
				return nil, er
			}
			er = json.Unmarshal(bbs, &maps)
		}
	} else if c.Request.Method == http.MethodGet {
		maps = make(map[string]interface{})
		values := c.Request.URL.Query()
		for k, v := range values {
			if len(v) == 1 {
				maps[k] = v[0]
			} else {
				maps[k] = v
			}
		}
	}
	if er != nil {
		return nil, er
	}
	return maps, nil

}
