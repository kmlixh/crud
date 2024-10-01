package crud

import (
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/kmlixh/gom/v3"
	"github.com/kmlixh/gom/v3/define"
	"net/http"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"unicode"
)

type NameMethods int

func (n NameMethods) Original() int {
	return int(n)
}

const (
	Original NameMethods = iota
	CamelCase
	SnakeCase
)

type DefaultRoutePath string

var prefix = "auto_crud_inject_"

func ToCamelCaseWithRegex(s string) string {
	// 正则表达式匹配一个或多个下划线，后面跟一个字母
	regex := regexp.MustCompile(`_+([a-zA-Z])`)
	// 将每个匹配项中的字母转换为大写
	return regex.ReplaceAllStringFunc(s, func(sub string) string {
		return strings.ToUpper(sub[len(sub)-1:])
	})
}
func ToSnakeCase(s string) string {
	var result []rune
	for i, r := range s {
		if unicode.IsUpper(r) {
			// 如果是大写字母且不是第一个字符，前面加下划线
			if i > 0 {
				result = append(result, '_')
			}
			result = append(result, unicode.ToLower(r))
		} else {
			result = append(result, r)
		}
	}
	return string(result)
}

type IHandlerRegister interface {
	Register(routes gin.IRoutes) error
	AppendHandler(name string, handler gin.HandlerFunc, asFirst bool) error
	AddHandler(routeHandler RouteHandler) error
}

func setContextEntity(c *gin.Context, entity any) {
	c.Set(prefix+"entity", entity)
}
func setContextCondition(c *gin.Context, cnd define.Condition) {
	c.Set(prefix+"cnd", cnd)
}
func getContextEntity(c *gin.Context) (any, bool) {
	return c.Get(prefix + "entity")
}
func hasEntity(c *gin.Context) bool {
	_, ok := c.Keys[prefix+"entity"]
	return ok
}

func DefaultUnMarshFunc(i any) gin.HandlerFunc {
	return func(context *gin.Context) {
		err := context.ShouldBind(i)
		if err != nil {
			context.Abort()
			return
		}
		setContextEntity(context, i)
	}
}
func StructToMap(input any) (bool, map[string]string) {
	// 创建一个空的map用来存储结果
	result := make(map[string]string)

	// 获取传入结构体的值和类型
	val := reflect.ValueOf(input)
	typ := reflect.TypeOf(input)

	// 确保传入的是struct
	if val.Kind() == reflect.Struct {
		// 遍历结构体的所有字段
		for i := 0; i < val.NumField(); i++ {
			// 获取字段的名称和值的类型
			field := typ.Field(i)
			fieldName := field.Name
			fieldType := field.Type.String()

			// 将字段名称和类型添加到map中
			result[fieldName] = fieldType
		}
	} else {
		return false, nil
	}

	return true, result
}
func NameToNameMap(methods NameMethods, names ...string) map[string]string {

}
func UnMarshFunc(i any, nameMap map[string]string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Request.Method
	}
}
func NameMapFrom(i any, methods NameMethods) map[string]string {
	ok, maps := StructToMap(i)
	if !ok {
		panic("input not a Struct")
	}
	nameMap := make(map[string]string)
	for key := range maps {
		if methods == CamelCase {
			nameMap[key] = ToCamelCaseWithRegex(key)
		} else if methods == SnakeCase {
			nameMap[key] = ToSnakeCase(key)
		} else {
			nameMap[key] = key
		}
	}
	return nameMap

}
func DefaultConditionFunc(i any, methods NameMethods) gin.HandlerFunc {
	return func(c *gin.Context) {

	}
}
func DefaultConditionFunc2(queryParam []ConditionParam) gin.HandlerFunc {
	return func(c *gin.Context) {
		cnd, _, er := MapToParamCondition(c, queryParam)
		if er == nil {
			setContextCondition(c, cnd)
		} else {
			c.Abort()
		}
	}
}

func NoneEntityToOperateError(c *gin.Context) {
	RenderErrs(c, errors.New("no entity find to operate!"))
}
func SetContextEntityHandleFunc(i any) gin.HandlerFunc {
	return func(c *gin.Context) {
		setContextEntity(c, i)
	}
}

func getContextCondition(c *gin.Context) (define.Condition, bool) {
	i, ok := c.Get("cnd")
	if ok {
		return i.(define.Condition), ok
	}
	return nil, ok
}
func getContextDatabase(c *gin.Context) (gom.DB, bool) {

}
func GetContextError(c *gin.Context) (error, bool) {
	i, ok := c.Get("error")
	if ok {
		return i.(error), ok
	}
	return nil, ok
}
func SetContextError(c *gin.Context, err error) {
	c.Set("error", err)
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

type RouteHandler struct {
	Path       string
	HttpMethod string
	Handlers   []gin.HandlerFunc
}
type ConditionParam struct {
	QueryName  string
	ColName    string
	Operations []define.Operation
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
		insertHandler := GetInsertHandler(db, queryCols, DefaultUnMarshFunc(i), nil)
		detailHandler := GetQuerySingleHandler(i, db, GetConditionParam(queryCols, columnIdxMap, i), queryCols, nil)
		updateHandler := GetUpdateHandler(db, GetConditionParam(queryCols, columnIdxMap, i), queryCols, DefaultUnMarshFunc(i), nil)
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
	insertHandler := GetInsertHandler(db, insertCols, DefaultUnMarshFunc(i), nil)
	detailHandler := GetQuerySingleHandler(i, db, GetConditionParam(queryDetailCols, columnFieldMap, i), queryDetailCols, nil)
	updateHandler := GetUpdateHandler(db, GetConditionParam(updateCols, columnFieldMap, i), updateCols, DefaultUnMarshFunc(i), nil)
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
		Operations: []define.Operation{define.Eq},
	}
}
func GetFieldDefaultQueryName(f reflect.StructField) string {
	return strings.ToLower(f.Name[0:1]) + f.Name[1:]
}
func GetQueryListHandler(i any, db *gom.DB, queryParam []ConditionParam, columns []string, beforeCommitFunc gin.HandlerFunc) RouteHandler {
	return RouteHandler{
		Path:       string(PathList),
		HttpMethod: "Any",
		Handlers:   []gin.HandlerFunc{QueryList(i, db, queryParam, columns, beforeCommitFunc)},
	}
}
func GetQuerySingleHandler(i any, db *gom.DB, queryParam []ConditionParam, columns []string, beforeCommitFunc gin.HandlerFunc) RouteHandler {
	return RouteHandler{
		Path:       string(PathDetail),
		HttpMethod: http.MethodGet,
		Handlers:   []gin.HandlerFunc{QuerySingle(i, db, queryParam, columns, beforeCommitFunc)},
	}
}
func GetInsertHandler(db *gom.DB, columns []string, unMarshalFunc gin.HandlerFunc, beforeCommitFunc gin.HandlerFunc) RouteHandler {
	return RouteHandler{
		Path:       string(PathAdd),
		HttpMethod: http.MethodPost,
		Handlers:   []gin.HandlerFunc{DoInsert(db, columns, unMarshalFunc, beforeCommitFunc)},
	}
}
func GetUpdateHandler(db *gom.DB, queryParam []ConditionParam, columns []string, unMarshalFunc gin.HandlerFunc, beforeCommitFunc gin.HandlerFunc) RouteHandler {
	return RouteHandler{
		Path:       string(PathUpdate),
		HttpMethod: http.MethodPost,
		Handlers:   []gin.HandlerFunc{DoUpdate(db, queryParam, columns, unMarshalFunc, beforeCommitFunc)},
	}
}
func GetDeleteHandler(i any, db *gom.DB, queryParam []ConditionParam, beforeCommitFunc *gin.HandlerFunc) RouteHandler {
	return RouteHandler{
		Path:       string(PathDelete),
		HttpMethod: http.MethodPost,
		Handlers:   []gin.HandlerFunc{DoDelete(db, queryParam, beforeCommitFunc)},
	}
}

func QueryList(i any, db *gom.DB, queryParam []ConditionParam, columns []string, beforeCommitFunc gin.HandlerFunc) gin.HandlerFunc {
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
			setContextEntity(c, results)
			setContextCondition(c, cnd)
			beforeCommitFunc(c)
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
func QuerySingle(i any, db *gom.DB, queryParam []ConditionParam, columns []string, beforeCommitFunc gin.HandlerFunc) gin.HandlerFunc {
	return func(c *gin.Context) {

		var err error
		cnd := gom.CndEmpty()
		if beforeCommitFunc != nil {
			beforeCommitFunc(c)
		}
		cnd, _, err = MapToParamCondition(c, queryParam)
		if cnd != nil && err == nil {
			db.Where(cnd)
		}
		if beforeCommitFunc != nil {
			beforeCommitFunc(c)
			if err != nil {
				RenderErr2(c, 500, "服务器内部错误："+err.Error())
				return
			}
		}
		_, err = db.Select(&i, columns...)
		if err != nil {
			RenderErr2(c, 500, "服务器内部错误："+err.Error())
			return
		}
		RenderOk(c, i)
	}
}
func DoUpdate(db *gom.DB, param []ConditionParam, columns []string, unMarshalFunc gin.HandlerFunc, beforeCommitFunc gin.HandlerFunc) gin.HandlerFunc {
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
		if unMarshalFunc != nil {
			unMarshalFunc(c)
			_, ok := getContextEntity(c)
			if !ok {
				RenderErrs(c, err)
				return
			}
		} else {
			RenderErrs(c, errors.New("unMarshalFunc is nil"))
			return
		}
		if beforeCommitFunc != nil {
			beforeCommitFunc(c)

			if err != nil {
				RenderErrs(c, err)
				return
			}
		}
		result, ok := getContextEntity(c)
		if ok {
			rs, er := db.Update(result, columns...)
			if er != nil {
				RenderErrs(c, err)
				return
			}
			RenderOk(c, rs)
		} else {
			RenderErrs(c, errors.New("更新失败"))
		}

	}
}
func DoInsert(db *gom.DB, columns []string, unMarshalFunc gin.HandlerFunc, beforeCommitFunc gin.HandlerFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		var err error
		if unMarshalFunc != nil {
			unMarshalFunc(c)
			_, ok := getContextEntity(c)
			if !ok {
				RenderErrs(c, err)
				return
			}
		} else {
			RenderErrs(c, errors.New("unMarshalFunc is nil"))
			return
		}
		if beforeCommitFunc != nil {
			beforeCommitFunc(c)
			if err != nil {
				RenderErrs(c, err)
				return
			}
		}
		result, ok := getContextEntity(c)
		if ok {
			rs, er := db.Insert(result, columns...)
			if er != nil {
				RenderErrs(c, err)
				return
			}
			RenderOk(c, rs)
		} else {
			NoneEntityToOperateError(c)
		}

	}
}

func DoDelete(i any, db *gom.DB, beforeDeleteFunc ...gin.HandlerFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		cnd, ok := getContextCondition(c)
		if !ok {
			RenderErrs(c, errors.New("can't get Cnd"))
			return
		}
		i, ok := getContextEntity(c)
		if !ok {

		}
		rs, er := db.Where(cnd).Delete(i)
		if er != nil {
			RenderErrs(c, er)
			return
		}
		RenderOk(c, rs)
	}
}
func DoQuery() gin.HandlerFunc {
	return func(c *gin.Context) {
		cnd, ok := getContextCondition(c)
		if !ok {
			RenderErrs(c, errors.New("can't get Cnd"))
			return
		}
		i, ok := getContextEntity(c)
		if !ok {

		}
		rs, er := db.Where(cnd).Delete(i)
		if er != nil {
			RenderErrs(c, er)
			return
		}
		RenderOk(c, rs)
	}
}
}

var Operators = []string{"Eq", "Le", "Lt", "Ge", "Gt", "Like", "LikeLeft", "LikeRight", "In", "NotIn", "NotLike", "NotEq"}

func MapToParamCondition(c *gin.Context, queryParam []ConditionParam) (define.Condition, map[string]interface{}, error) {
	maps, err := GetMapFromRst(c)
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
			for _, oper := range Operators {
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
