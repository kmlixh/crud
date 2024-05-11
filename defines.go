package crud

import (
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/kmlixh/gom/v3"
	"github.com/kmlixh/gom/v3/define"
	"net/http"
	"reflect"
	"strconv"
	"strings"
)

type DefaultRoutePath string

type IHandlerRegister interface {
	Register(routes gin.IRoutes) error
	AppendHandler(name string, handler gin.HandlerFunc, asFirst bool) error
	AddHandler(routeHandler RouteHandler) error
}
type InterceptorFunc func(c *gin.Context, i any) (any, error)

func DefaultUnMarshalFunc(c *gin.Context, i any) (any, error) {
	if err := c.ShouldBind(i); err != nil {
		return nil, err
	}
	return i, nil
}

const (
	PathList   DefaultRoutePath = "list"
	PathAdd    DefaultRoutePath = "add"
	PathDetail DefaultRoutePath = "detail"
	PathUpdate DefaultRoutePath = "update"
	PathDelete DefaultRoutePath = "delete"
)

func (d DefaultRoutePath) String() string {
	return string(d)
}

type ConditionParam struct {
	QueryName string
	ColName   string
	Required  bool
}
type HandlerRegister struct {
	Name     string
	Handlers []RouteHandler
	IdxMap   map[string]int
}

func (h HandlerRegister) AddHandler(routeHandler RouteHandler) error {
	if h.Handlers == nil {
		h.Handlers = make([]RouteHandler, 0)
	}
	if h.IdxMap == nil {
		h.IdxMap = make(map[string]int)
	}
	if _, ok := h.IdxMap[routeHandler.Path]; ok {
		h.Handlers[h.IdxMap[routeHandler.Path]] = routeHandler
	} else {
		h.Handlers = append(h.Handlers, routeHandler)
		h.IdxMap[routeHandler.Path] = len(h.Handlers) - 1
	}
	return nil
}
func (h HandlerRegister) GetHandler(name string) (RouteHandler, error) {
	idx, ok := h.IdxMap[name]
	if !ok {
		return RouteHandler{}, errors.New(fmt.Sprintf("handler [%s] not found", name))
	} else {
		return h.Handlers[idx], nil
	}
}
func (h HandlerRegister) DeleteHandler(name string) error {
	idx, ok := h.IdxMap[name]
	if !ok {
		return errors.New(fmt.Sprintf("handler [%s] not found", name))
	} else {
		h.Handlers = append(h.Handlers[:idx], h.Handlers[idx+1:]...)
		for k, v := range h.IdxMap {
			if v > idx {
				h.IdxMap[k] = v - 1
			}
		}
		return nil
	}
}
func (h HandlerRegister) AppendHandler(name string, handler gin.HandlerFunc, asFirst bool) error {
	_, ok := h.IdxMap[name]
	var routeHandler RouteHandler
	if !ok {
		return errors.New(fmt.Sprintf("handler [%s] not found", name))
	} else {
		routeHandler = h.Handlers[h.IdxMap[name]]

	}
	if asFirst {
		routeHandler.Handlers = append([]gin.HandlerFunc{handler}, routeHandler.Handlers...)
	} else {
		routeHandler.Handlers = append(routeHandler.Handlers, handler)
	}
	h.Handlers[h.IdxMap[name]] = routeHandler
	return nil
}

func (h HandlerRegister) Register(routes gin.IRoutes) error {
	if h.Handlers == nil || len(h.Handlers) == 0 {
		return errors.New("route handler could not be empty or nil")
	}
	for _, handler := range h.Handlers {
		if handler.HttpMethod != "Any" {
			routes.Handle(handler.HttpMethod, h.Name+"/"+handler.Path, handler.Handlers...)
		} else {
			routes.Any(h.Name+"/"+handler.Path, handler.Handlers...)
		}
	}
	return nil
}

type RouteHandler struct {
	Path       string
	HttpMethod string
	Handlers   []gin.HandlerFunc
}

func RegisterHandler(name string, routes gin.IRoutes, handlers ...RouteHandler) error {
	if handlers == nil || len(handlers) == 0 {
		return errors.New("route handler could not be empty or nil")
	}
	register, err := GenHandlerRegister(name, handlers...)
	if err != nil {
		return err
	}
	return register.Register(routes)
}
func GenHandlerRegister(name string, handlers ...RouteHandler) (IHandlerRegister, error) {
	if handlers == nil || len(handlers) == 0 {
		return HandlerRegister{}, errors.New("route handler could not be empty or nil")
	}
	handlerIdxMap := make(map[string]int)
	for i, handler := range handlers {
		handlerIdxMap[handler.Path] = i
	}
	return HandlerRegister{
		Name:     name,
		Handlers: handlers,
		IdxMap:   handlerIdxMap,
	}, nil
}
func GenDefaultHandlerRegister(prefix string, i any, db *gom.DB) (IHandlerRegister, error) {
	columnNames, primaryKeys, primaryAuto, columnIdxMap := gom.GetColumns(reflect.ValueOf(i))
	queryCols := append(primaryKeys, append(primaryAuto, columnNames...)...)
	if len(columnNames) > 0 {
		listHandler := GetQueryListHandler(i, db, GetConditionParam(queryCols, columnIdxMap, i), queryCols, nil)
		insertHandler := GetInsertHandler(i, db, queryCols, DefaultUnMarshalFunc, nil)
		detailHandler := GetQuerySingleHandler(i, db, GetConditionParam(queryCols, columnIdxMap, i), queryCols, nil)
		updateHandler := GetUpdateHandler(i, db, GetConditionParam(queryCols, columnIdxMap, i), queryCols, DefaultUnMarshalFunc, nil)
		deleteHandler := GetDeleteHandler(i, db, GetConditionParam(queryCols, columnIdxMap, i), nil)
		return GenHandlerRegister(prefix, listHandler, insertHandler, detailHandler, updateHandler, deleteHandler)
	} else {
		return nil, errors.New("Struct was empty")
	}
}

func RegisterDefaultLite(prefix string, i any, routes gin.IRoutes, db *gom.DB) error {
	columnNames, primaryKeys, primaryAuto, columnIdxMap := gom.GetColumns(reflect.ValueOf(i))
	if len(columnNames) > 0 {
		return RegisterDefault(prefix, i, routes, db, append(primaryKeys, append(primaryAuto, columnNames...)...), append(primaryKeys, append(primaryAuto, columnNames...)...), append(primaryKeys, columnNames...), columnNames, columnIdxMap)
	} else {
		return errors.New("Struct was empty")
	}

}
func RegisterDefault(prefix string, i any, routes gin.IRoutes, db *gom.DB, queryCols []string, queryDetailCols []string, insertCols []string, updateCols []string, columnFieldMap map[string]string) error {
	listHandler := GetQueryListHandler(i, db, GetConditionParam(queryCols, columnFieldMap, i), queryCols, nil)
	insertHandler := GetInsertHandler(i, db, insertCols, DefaultUnMarshalFunc, nil)
	detailHandler := GetQuerySingleHandler(i, db, GetConditionParam(queryDetailCols, columnFieldMap, i), queryDetailCols, nil)
	updateHandler := GetUpdateHandler(i, db, GetConditionParam(updateCols, columnFieldMap, i), updateCols, DefaultUnMarshalFunc, nil)
	deleteHandler := GetDeleteHandler(i, db, GetConditionParam(updateCols, columnFieldMap, i), nil)
	return RegisterHandler(prefix, routes, listHandler, insertHandler, detailHandler, updateHandler, deleteHandler)
}
func GetConditionParam(columns []string, columnFieldMap map[string]string, i any) []ConditionParam {
	t := reflect.TypeOf(i)
	params := make([]ConditionParam, 0)
	for _, col := range columns {
		fieldName := columnFieldMap[col]
		f, ok := t.FieldByName(fieldName)
		if !ok {
			panic(errors.New(fmt.Sprintf(" [%s] was not exist! ", fieldName)))
		}
		params = append(params, GenDefaultConditionParamByType(col, f))
	}
	return params
}
func GenDefaultConditionParamByType(column string, f reflect.StructField) ConditionParam {
	return ConditionParam{
		QueryName: GetFieldDefaultQueryName(f),
		ColName:   column,
		Required:  false,
	}
}
func GetFieldDefaultQueryName(f reflect.StructField) string {
	return strings.ToLower(f.Name[0:1]) + f.Name[1:]
}
func GetQueryListHandler(i any, db *gom.DB, queryParam []ConditionParam, columns []string, beforeCommitFunc InterceptorFunc) RouteHandler {
	return RouteHandler{
		Path:       string(PathList),
		HttpMethod: "Any",
		Handlers:   []gin.HandlerFunc{QueryList(i, db, queryParam, columns, beforeCommitFunc)},
	}
}
func GetQuerySingleHandler(i any, db *gom.DB, queryParam []ConditionParam, columns []string, beforeCommitFunc InterceptorFunc) RouteHandler {
	return RouteHandler{
		Path:       string(PathDetail),
		HttpMethod: http.MethodGet,
		Handlers:   []gin.HandlerFunc{QuerySingle(i, db, queryParam, columns, beforeCommitFunc)},
	}
}
func GetInsertHandler(i any, db *gom.DB, columns []string, unMarshalFunc InterceptorFunc, beforeCommitFunc InterceptorFunc) RouteHandler {
	return RouteHandler{
		Path:       string(PathAdd),
		HttpMethod: http.MethodPost,
		Handlers:   []gin.HandlerFunc{DoInsert(i, db, columns, unMarshalFunc, beforeCommitFunc)},
	}
}
func GetUpdateHandler(i any, db *gom.DB, queryParam []ConditionParam, columns []string, unMarshalFunc InterceptorFunc, beforeCommitFunc InterceptorFunc) RouteHandler {
	return RouteHandler{
		Path:       string(PathUpdate),
		HttpMethod: http.MethodPost,
		Handlers:   []gin.HandlerFunc{DoUpdate(i, db, queryParam, columns, unMarshalFunc, beforeCommitFunc)},
	}
}
func GetDeleteHandler(i any, db *gom.DB, queryParam []ConditionParam, beforeCommitFunc InterceptorFunc) RouteHandler {
	return RouteHandler{
		Path:       string(PathDelete),
		HttpMethod: http.MethodPost,
		Handlers:   []gin.HandlerFunc{DoDelete(i, db, queryParam, beforeCommitFunc)},
	}
}

func QueryList(i interface{}, db *gom.DB, queryParam []ConditionParam, columns []string, beforeCommitFunc InterceptorFunc) gin.HandlerFunc {
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
		if beforeCommitFunc != nil {
			results, err = beforeCommitFunc(c, results)
			if err != nil {
				RenderErr2(c, 500, "服务器内部错误："+err.Error())
				return
			}
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
func QuerySingle(i any, db *gom.DB, queryParam []ConditionParam, columns []string, beforeCommitFunc InterceptorFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		results := reflect.New(reflect.TypeOf(i)).Interface()
		cnd, _, err := MapToParamCondition(c, queryParam)
		if cnd != nil && err == nil {
			db.Where(cnd)
		}
		if beforeCommitFunc != nil {
			results, err = beforeCommitFunc(c, results)
			if err != nil {
				RenderErr2(c, 500, "服务器内部错误："+err.Error())
				return
			}
		}
		_, err = db.Select(results, columns...)
		if err != nil {
			RenderErr2(c, 500, "服务器内部错误："+err.Error())
			return
		}
		RenderOk(c, results)
	}
}
func DoUpdate(i any, db *gom.DB, param []ConditionParam, columns []string, unMarshalFunc InterceptorFunc, beforeCommitFunc InterceptorFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		var err error
		cnd, _, er := MapToParamCondition(c, param)
		if er != nil {
			RenderErrs(c, er)
			return
		}
		if cnd != nil {
			db.Where(cnd)
		}
		results := reflect.New(reflect.TypeOf(i)).Interface()
		if unMarshalFunc != nil {
			results, err = unMarshalFunc(c, results)
			if err != nil {
				RenderErrs(c, err)
				return
			}
		} else {
			RenderErrs(c, errors.New("unMarshalFunc is nil"))
			return
		}
		if beforeCommitFunc != nil {
			results, err = beforeCommitFunc(c, results)
			if err != nil {
				RenderErrs(c, err)
				return
			}
		}
		rs, er := db.Update(results, columns...)
		if er != nil {
			RenderErrs(c, err)
			return
		}
		RenderOk(c, rs)
	}
}
func DoInsert(i any, db *gom.DB, columns []string, unMarshalFunc InterceptorFunc, beforeCommitFunc InterceptorFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		var err error
		results := reflect.New(reflect.TypeOf(i)).Interface()
		if unMarshalFunc != nil {
			results, err = unMarshalFunc(c, results)
			if err != nil {
				RenderErrs(c, err)
				return
			}
		} else {
			RenderErrs(c, errors.New("unMarshalFunc is nil"))
			return
		}
		if beforeCommitFunc != nil {
			results, err = beforeCommitFunc(c, results)
			if err != nil {
				RenderErrs(c, err)
				return
			}
		}
		rs, er := db.Insert(results, columns...)
		if er != nil {
			RenderErrs(c, err)
			return
		}
		RenderOk(c, rs)
	}
}
func DoDelete(i any, db *gom.DB, param []ConditionParam, beforeCommitFunc InterceptorFunc) gin.HandlerFunc {
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

var operators = []string{"Eq", "Le", "Lt", "Ge", "Gt", "Like", "LikeLeft", "LikeRight", "In", "NotIn", "NotLike", "NotEq"}

func MapToParamCondition(c *gin.Context, queryParam []ConditionParam) (define.Condition, map[string]interface{}, error) {
	maps, err := GetConditionMapFromRst(c)
	hasValMap := make(map[string]string)
	if err != nil {
		return nil, nil, err
	}
	if len(maps) > 0 && len(queryParam) > 0 {
		var cnd = gom.CndEmpty()
		for _, param := range queryParam {
			oldName, hasOldVal := hasValMap[param.QueryName]
			if hasOldVal {
				return nil, nil, errors.New(fmt.Sprintf("u have a query condition like [%s]", oldName))
			}
			for _, oper := range operators {
				val, hasVal := maps[param.QueryName+oper]
				if hasVal {
					hasValMap[param.ColName] = param.QueryName + oper
					switch oper {
					case "Eq":
						cnd.Eq(param.ColName, val)
					case "NotEq":
						cnd.NotEq(param.ColName, val)
					case "Le":
						cnd.Le(param.ColName, val)
					case "Lt":
						cnd.Lt(param.ColName, val)
					case "Ge":
						cnd.Ge(param.ColName, val)
					case "Gt":
						cnd.Gt(param.ColName, val)
					case "Like":
						cnd.Like(param.ColName, val)
					case "LikeLeft":
						cnd.LikeIgnoreStart(param.ColName, val)
					case "LikeRight":
						cnd.LikeIgnoreEnd(param.ColName, val)
					case "In":
						cnd.In(param.ColName, gom.UnZipSlice(val)...)
					case "NotIn":
						cnd.NotIn(param.ColName, gom.UnZipSlice(val)...)
					case "NotLike":
						cnd.NotLike(param.ColName, val)
					}
				}
			}

		}
		return cnd, maps, nil
	} else {
		return nil, nil, nil
	}
}
