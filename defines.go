package AutoCrudGo

import (
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/kmlixh/gom/v3"
)

type RouteHandler struct {
	Path       string
	HttpMethod string
	Handlers   []gin.HandlerFunc
}

func RegisterHandler(name string, routes gin.IRoutes, handlers ...RouteHandler) error {
	if handlers == nil || len(handlers) == 0 {
		return errors.New("route handler could not be empty or nil")
	}
	for _, handler := range handlers {
		if handler.HttpMethod != "Any" {
			routes.Handle(handler.HttpMethod, name+"/"+handler.Path, handler.Handlers...)
		} else {
			routes.Any(name+"/"+handler.Path, handler.Handlers...)
		}
	}
	return nil
}

type QueryOpera int

const (
	_ QueryOpera = iota
	Le
	Lt
	Ge
	Gt
	Eq
	Like
	LikeIgnoreLeft
	LikeIgnoreRight
	Between
)

type ConditionParam struct {
	QueryName string
	ColName   string
	Required  bool
	QueryOpera
}

func RegisterDefaultLite(prefix string, i interface{}, routes gin.IRoutes, db *gom.DB) {

}
func RegisterDefault(prefix string, i interface{}, routes gin.IRoutes, db *gom.DB, queryCols []string, insertCols []string, updateCols []string) {

}
func GetQueryListHandler(i interface{}, db *gom.DB, queryParam []ConditionParam, columns []string) RouteHandler {
	return RouteHandler{
		Path:       "list",
		HttpMethod: "Any",
		Handlers:   []gin.HandlerFunc{QueryList(i, db, queryParam, columns)},
	}
}
func GetQuerySingleHandler(i interface{}, db *gom.DB, queryParam []ConditionParam, columns []string) RouteHandler {
	return RouteHandler{
		Path:       "detail",
		HttpMethod: http.MethodGet,
		Handlers:   []gin.HandlerFunc{QuerySingle(i, db, queryParam, columns)},
	}
}
func GetInsertHandler(i interface{}, db *gom.DB, columns []string) RouteHandler {
	return RouteHandler{
		Path:       "add",
		HttpMethod: http.MethodPost,
		Handlers:   []gin.HandlerFunc{DoInsert(i, db, columns)},
	}
}
func GetUpdateHandler(i interface{}, db *gom.DB, queryParam []ConditionParam, columns []string) RouteHandler {
	return RouteHandler{
		Path:       "update",
		HttpMethod: http.MethodPost,
		Handlers:   []gin.HandlerFunc{DoUpdate(i, db, queryParam, columns)},
	}
}
func GetDeleteHandler(i interface{}, db *gom.DB, queryParam []ConditionParam, columns []string) RouteHandler {
	return RouteHandler{
		Path:       "delete",
		HttpMethod: http.MethodPost,
		Handlers:   []gin.HandlerFunc{DoDelete(i, db, queryParam)},
	}
}

func QueryList(i interface{}, db *gom.DB, queryParam []ConditionParam, columns []string) gin.HandlerFunc {
	return func(c *gin.Context) {
		codeMsg := Ok()
		results := reflect.New(reflect.SliceOf(reflect.TypeOf(i))).Interface()
		cnd, maps, err := MapToParamCondition(c, queryParam)

		pageNum := int64(0)
		pageSize := int64(20)
		pNum, hasPageNum := maps["pageNum"]
		pSize, hasPageSize := maps["pageSize"]
		if hasPageSize {
			switch pNum.(type) {
			case string:
				t, er := strconv.Atoi(pSize.(string))
				if er == nil {
					pageSize = int64(t)
				}
			default:
				pageSize = pSize.(int64)
			}
		}
		if hasPageNum {
			switch pNum.(type) {
			case string:
				t, er := strconv.Atoi(pNum.(string))
				if er == nil {
					pageNum = int64(t)
				}
			default:
				pageNum = pNum.(int64)
			}
			if cnd != nil && err == nil {
				db.Where(cnd)
			}
			count, err := db.Count("*")
			if err != nil {
				RenderErr2(c, 500, "统计数据出错！")
				return
			}
			codeMsg.Set("total", count)
			codeMsg.Set("pageNum", pageNum)
			codeMsg.Set("pageSize", pageSize)
			db.Page(pageNum, pageSize)
		}
		if cnd != nil && err == nil {
			db.Where(cnd)
		}
		_, err = db.Select(results, columns...)
		if err != nil {
			RenderErr2(c, 500, "服务器内部错误："+err.Error())
			return
		}
		codeMsg.SetData(results)
		RenderJson(c, codeMsg)
	}
}
func QuerySingle(i interface{}, db *gom.DB, queryParam []ConditionParam, columns []string) gin.HandlerFunc {
	return func(c *gin.Context) {
		results := reflect.New(reflect.TypeOf(i)).Interface()
		cnd, _, err := MapToParamCondition(c, queryParam)
		if cnd != nil && err == nil {
			db.Where(cnd)
		}
		_, err = db.Select(results, columns...)
		if err != nil {
			RenderErr2(c, 500, "服务器内部错误："+err.Error())
			return
		}
		RenderOk(c, results)
	}
}
func DoUpdate(i interface{}, db *gom.DB, param []ConditionParam, columns []string) gin.HandlerFunc {
	return func(c *gin.Context) {
		cnd, _, er := MapToParamCondition(c, param)
		if er != nil {
			RenderErrs(c, er)
			return
		}
		if cnd != nil {
			db.Where(cnd)
		}
		results := reflect.New(reflect.TypeOf(i)).Interface()
		err := c.ShouldBind(&results)
		if err != nil {
			RenderErrs(c, err)
			return
		}
		rs, er := db.Insert(results, columns...)
		if er != nil {
			RenderErrs(c, err)
			return
		}
		RenderOk(c, rs)
	}
}
func DoInsert(i interface{}, db *gom.DB, columns []string) gin.HandlerFunc {
	return func(c *gin.Context) {
		results := reflect.New(reflect.TypeOf(i)).Interface()
		err := c.ShouldBind(&results)
		if err != nil {
			RenderErrs(c, err)
			return
		}
		rs, er := db.Insert(results, columns...)
		if er != nil {
			RenderErrs(c, err)
			return
		}
		RenderOk(c, rs)
	}
}
func DoDelete(i interface{}, db *gom.DB, param []ConditionParam) gin.HandlerFunc {
	return func(c *gin.Context) {
		cnd, _, er := MapToParamCondition(c, param)
		if er != nil {
			RenderErrs(c, er)
			return
		}
		rs, er := db.Where(cnd).Delete(i)
		if er != nil {
			RenderErrs(c, er)
			return
		}
		RenderOk(c, rs)
	}
}

func MapToParamCondition(c *gin.Context, queryParam []ConditionParam) (gom.Condition, map[string]interface{}, error) {
	maps, err := GetConditionMapFromRst(c)
	if err != nil {
		return nil, nil, err
	}
	if len(maps) > 0 && len(queryParam) > 0 {
		var cnd = gom.CndEmpty()
		for _, param := range queryParam {
			val, hasVal := maps[param.QueryName]
			if param.Required && !hasVal {
				return nil, maps, errors.New(fmt.Sprintf("%s is requied!", param.QueryName))
			}
			if hasVal {
				switch param.QueryOpera {
				case Eq:
					t := reflect.TypeOf(val)
					if t.Kind() == reflect.Ptr {
						t = t.Elem()
					}
					if t.Kind() != reflect.Struct || t.Kind() == reflect.TypeOf(time.Now()).Kind() || ((t.Kind() == reflect.Slice || t.Kind() == reflect.Array) && t.Elem().Kind() != reflect.Struct) {
						value := val
						if (t.Kind() == reflect.Slice || t.Kind() == reflect.Array) && t.Elem().Kind() != reflect.Struct {
							cnd.In(param.ColName, gom.UnZipSlice(value)...)
						} else {
							cnd.Eq(param.ColName, value)
						}
					}
				case Ge:
					cnd.Ge(param.ColName, val)
				case Gt:
					cnd.Gt(param.ColName, val)
				case Lt:
					cnd.Lt(param.ColName, val)
				case Le:
					cnd.Le(param.ColName, val)
				case Like:
					cnd.Like(param.ColName, val)
				case LikeIgnoreLeft:
					cnd.LikeIgnoreStart(param.ColName, val)
				case LikeIgnoreRight:
					cnd.LikeIgnoreEnd(param.ColName, val)

				}
			}
		}
		return cnd, maps, nil
	} else {
		return nil, nil, nil
	}
}
