package crud

import "C"
import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/gin-gonic/gin"
	"github.com/kmlixh/gom/v4"
	"github.com/kmlixh/gom/v4/define"
)

var prefix = "auto_crud_inject_"

var (
	primaryKeys  []string
	primaryAuto  []string
	columnNames  []string
	columnIdxMap map[string]string
)

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

func SetContextEntity(i any) gin.HandlerFunc {
	return SetContextAny("entity", i)
}

func GetContextEntity(c *gin.Context) (any, bool) {
	return c.Get(prefix + "entity")
}

func HasEntity(c *gin.Context) bool {
	_, ok := c.Keys[prefix+"entity"]
	return ok
}
func DefaultGenPageFromRstQuery(c *gin.Context) {
	pageNumt := c.Query("pageNum")
	if pageNumt == "" {
		pageNumt = "1"
	}
	pageNum, er := strconv.Atoi(pageNumt)
	if er != nil {
		c.Abort()
		RenderErrs(c, er)
		return
	}
	SetContextPageNumber(pageNum)(c)
	pageSizet := c.Query("pageSize")
	if pageSizet == "" {
		pageSizet = "10"
	}
	pageSize, er := strconv.Atoi(pageSizet)
	if er != nil {
		c.Abort()
		RenderErrs(c, er)
		return
	}
	SetContextPageSize(pageSize)(c)
}

func SetContextPageNumber(num int) gin.HandlerFunc {
	return SetContextAny("pageNum", num)
}

func SetContextPageSize(size int) gin.HandlerFunc {
	return SetContextAny("pageSize", size)
}
func getContextPageNumber(c *gin.Context) int {
	pageNumber := 0
	i, ok := c.Get(prefix + "pageNum")
	if ok {
		pageNumber = i.(int)
	} else {
		pp, er := strconv.Atoi(c.Param("pageNum"))
		if er == nil {
			pageNumber = pp
		}
	}
	if pageNumber == 0 {
		pageNumber = 1
	}
	return pageNumber
}
func getContextPageSize(c *gin.Context) int {
	pageSize := 0
	i, ok := c.Get(prefix + "pageSize")
	if ok {
		pageSize = i.(int)
	} else {
		pp, er := strconv.Atoi(c.Param("pageSize"))
		if er == nil {
			pageSize = pp
		}
	}
	if pageSize == 0 {
		pageSize = 20
	}
	return pageSize

}

func DefaultUnMarshFunc(i any) gin.HandlerFunc {
	return func(context *gin.Context) {
		err := context.ShouldBindJSON(i)
		if err != nil {
			context.Abort()
			RenderErrs(context, err)
			return
		}
		context.Set(prefix+"entity", i)
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

func SetOrderBys(orderBys []define.OrderBy) gin.HandlerFunc {
	return SetContextAny("orderBys", orderBys)
}
func GetOrderBys(c *gin.Context) ([]define.OrderBy, bool) {
	i, ok := c.Get("orderBys")
	if ok {
		return i.([]define.OrderBy), ok
	}
	return nil, ok
}
func HasEntityOfName(c *gin.Context, name string) bool {
	_, ok := c.Get(name)
	return ok
}

func SetColumns(columns []string) gin.HandlerFunc {
	return SetContextAny("cols", columns)
}

func SetConditionParamAsCnd(queryParam []ConditionParam) gin.HandlerFunc {
	return func(c *gin.Context) {
		cnd, _, er := MapToParamCondition(c, queryParam)
		if er != nil {
			c.Abort()
			RenderErrs(c, er)
			return
		}
		if cnd != nil && cnd.Field != "" {
			c.Set(prefix+"cnd", cnd)
		}
	}
}

// RouteHandler represents a route handler configuration
type RouteHandler struct {
	Path       string
	HttpMethod string
	Handlers   []gin.HandlerFunc
}

// ICrud represents the CRUD interface
type ICrud interface {
	Register(routes gin.IRoutes, prefix ...string) error
	AddHandler(routeHandler RouteHandler) error
	GetHandler(name string) (RouteHandler, error)
	DeleteHandler(name string) error
	AppendHandler(name string, handler gin.HandlerFunc, appendType HandlerAppendType, position HandlerPosition) error
}

// HandlerAppendType represents how to append a handler
type HandlerAppendType int

const (
	Before HandlerAppendType = iota
	After
	Replace
)

// HandlerPosition represents where to append a handler
type HandlerPosition int

const (
	BeforeCommit HandlerPosition = iota
	AfterCommit
)

// Crud represents the CRUD implementation
type Crud struct {
	Name     string
	Handlers []RouteHandler
	IdxMap   map[string]int
}

// ConditionPayload represents a condition's operation type and value
type ConditionPayload struct {
	Type  define.OpType
	Value interface{}
}

// Condition represents a where condition with multiple payloads
type Condition struct {
	PayLoads map[string]ConditionPayload
}

// ConditionParam represents a condition parameter
type ConditionParam struct {
	QueryName string
	ColName   string
	Operation define.OpType
}

func NewCrud2(prefix string, i any, db *gom.DB, queryCols []string, queryConditionParam []ConditionParam, queryDetailCols []string, detailConditionParam []ConditionParam, insertCols []string, updateCols []string, updateConditionParam []ConditionParam, deleteConditionParam []ConditionParam) (ICrud, error) {
	listHandler := GetQueryListHandler(SetContextDatabase(db), SetContextEntity(i), DoNothingFunc, SetConditionParamAsCnd(queryConditionParam), SetColumns(queryCols), DefaultGenPageFromRstQuery, DoNothingFunc)
	detailHandler := GetQuerySingleHandler(SetContextDatabase(db), SetContextEntity(i), DoNothingFunc, SetConditionParamAsCnd(detailConditionParam), SetColumns(queryDetailCols), DoNothingFunc, DoNothingFunc)
	insertHandler := GetInsertHandler(SetContextDatabase(db), DoNothingFunc, DefaultUnMarshFunc(i), DoNothingFunc, SetColumns(insertCols), DoNothingFunc, DoNothingFunc)
	updateHandler := GetUpdateHandler(SetContextDatabase(db), DoNothingFunc, DefaultUnMarshFunc(i), SetConditionParamAsCnd(updateConditionParam), SetColumns(updateCols), DoNothingFunc, DoNothingFunc)
	deleteHandler := GetDeleteHandler(SetContextDatabase(db), SetContextEntity(i), DoNothingFunc, SetConditionParamAsCnd(deleteConditionParam), DoNothingFunc, DoNothingFunc, DoNothingFunc)
	tableStructHandler := GetTableStructHandler(SetContextDatabase(db), SetContextEntity(i), DoNothingFunc, DoNothingFunc, DoNothingFunc, DoNothingFunc, DoNothingFunc)
	return GenHandlerRegister(prefix, listHandler, insertHandler, detailHandler, updateHandler, deleteHandler, tableStructHandler)
}

func GetQueryListHandler(beforeCommitFunc ...gin.HandlerFunc) RouteHandler {
	return GetRouteHandler(string(PathList), "GET", append(beforeCommitFunc, QueryList())...)
}

func GetQuerySingleHandler(beforeCommitFunc ...gin.HandlerFunc) RouteHandler {
	return GetRouteHandler(string(PathDetail), "GET", append(beforeCommitFunc, QuerySingle())...)
}

func GetInsertHandler(beforeCommitFunc ...gin.HandlerFunc) RouteHandler {
	return GetRouteHandler(string(PathAdd), "POST", append(beforeCommitFunc, DoInsert())...)
}

func GetUpdateHandler(beforeCommitFunc ...gin.HandlerFunc) RouteHandler {
	return GetRouteHandler(string(PathUpdate), "POST", append(beforeCommitFunc, DoUpdate())...)
}

func GetDeleteHandler(beforeCommitFunc ...gin.HandlerFunc) RouteHandler {
	return GetRouteHandler(string(PathDelete), "POST", append(beforeCommitFunc, DoDelete())...)
}

func GetTableStructHandler(beforeCommitFunc ...gin.HandlerFunc) RouteHandler {
	return GetRouteHandler(string(PathTableStruct), "GET", append(beforeCommitFunc, DoTableStruct())...)
}

func GetRouteHandler(path string, method string, handlers ...gin.HandlerFunc) RouteHandler {
	return RouteHandler{
		Path:       path,
		HttpMethod: method,
		Handlers:   handlers,
	}
}

func (d DefaultRoutePath) String() string {
	return string(d)
}

func (h Crud) AddHandler(routeHandler RouteHandler) error {
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
func (h Crud) GetHandler(name string) (RouteHandler, error) {
	idx, ok := h.IdxMap[name]
	if !ok {
		return RouteHandler{}, fmt.Errorf("handler [%s] not found", name)
	} else {
		return h.Handlers[idx], nil
	}
}
func (h Crud) DeleteHandler(name string) error {
	idx, ok := h.IdxMap[name]
	if !ok {
		return fmt.Errorf("handler [%s] not found", name)
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
func (h Crud) AppendHandler(name string, handler gin.HandlerFunc, appendType HandlerAppendType, position HandlerPosition) error {
	_, ok := h.IdxMap[name]
	var routeHandler RouteHandler
	if !ok {
		return fmt.Errorf("handler [%s] not found", name)
	} else {
		routeHandler = h.Handlers[h.IdxMap[name]]

	}
	funcs := routeHandler.Handlers[position]
	if appendType == Before {
		oldFunc := funcs
		funcs = func(c *gin.Context) {
			handler(c)
			oldFunc(c)
		}
	} else if appendType == Replace {
		funcs = handler
	} else if appendType == After {
		oldFunc := funcs
		funcs = func(c *gin.Context) {
			oldFunc(c)
			handler(c)
		}
	}
	h.Handlers[h.IdxMap[name]] = routeHandler
	return nil
}

func (h Crud) Register(routes gin.IRoutes, prefix ...string) error {
	name := h.Name
	if len(prefix) == 1 {
		name = prefix[0]
	}
	if len(h.Handlers) == 0 {
		return errors.New("route handler could not be empty or nil")
	}
	for _, handler := range h.Handlers {
		if handler.HttpMethod != "Any" {
			routes.Handle(handler.HttpMethod, name+"/"+handler.Path, handler.Handlers...)
		} else {
			routes.Any(name+"/"+handler.Path, handler.Handlers...)
		}
	}
	return nil
}

func GenHandlerRegister(name string, handlers ...RouteHandler) (ICrud, error) {
	if len(handlers) == 0 {
		return Crud{}, errors.New("route handler could not be empty or nil")
	}
	handlerIdxMap := make(map[string]int)
	for i, handler := range handlers {
		handlerIdxMap[handler.Path] = i
	}
	return Crud{
		Name:     name,
		Handlers: handlers,
		IdxMap:   handlerIdxMap,
	}, nil
}

func NewCrud(db *gom.DB, tableName string, model interface{}) (ICrud, error) {
	if tableName == "" {
		return nil, errors.New("table name cannot be empty")
	}
	if model == nil {
		return nil, errors.New("model cannot be nil")
	}

	t := reflect.TypeOf(model)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return nil, errors.New("model must be a struct")
	}

	// 获取模型的所有字段作为默认的查询和操作字段
	fields := make([]string, 0)
	defaultCondParams := make([]ConditionParam, 0)

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if !field.IsExported() {
			continue
		}

		fields = append(fields, field.Name)
		fieldType := field.Type.Kind()

		// 根据字段类型生成不同的条件参数
		switch fieldType {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
			reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
			reflect.Float32, reflect.Float64:
			// 数值类型：支持等于、不等于、大于、大于等于、小于、小于等于
			defaultCondParams = append(defaultCondParams,
				ConditionParam{QueryName: field.Name + "Eq", ColName: field.Name, Operation: define.OpEq},
				ConditionParam{QueryName: field.Name + "Ne", ColName: field.Name, Operation: define.OpNe},
				ConditionParam{QueryName: field.Name + "Gt", ColName: field.Name, Operation: define.OpGt},
				ConditionParam{QueryName: field.Name + "Ge", ColName: field.Name, Operation: define.OpGe},
				ConditionParam{QueryName: field.Name + "Lt", ColName: field.Name, Operation: define.OpLt},
				ConditionParam{QueryName: field.Name + "Le", ColName: field.Name, Operation: define.OpLe},
			)

		case reflect.String:
			// 字符串类型：支持等于、不等于、包含、左包含、右包含、不包含
			defaultCondParams = append(defaultCondParams,
				ConditionParam{QueryName: field.Name + "Eq", ColName: field.Name, Operation: define.OpEq},
				ConditionParam{QueryName: field.Name + "Ne", ColName: field.Name, Operation: define.OpNe},
				ConditionParam{QueryName: field.Name + "Like", ColName: field.Name, Operation: define.OpLike},
				ConditionParam{QueryName: field.Name + "LikeLeft", ColName: field.Name, Operation: define.OpLike},
				ConditionParam{QueryName: field.Name + "LikeRight", ColName: field.Name, Operation: define.OpLike},
				ConditionParam{QueryName: field.Name + "NotLike", ColName: field.Name, Operation: define.OpNotLike},
			)

		case reflect.Slice, reflect.Array:
			// 数组/切片类型：支持包含和不包含
			defaultCondParams = append(defaultCondParams,
				ConditionParam{QueryName: field.Name + "In", ColName: field.Name, Operation: define.OpIn},
				ConditionParam{QueryName: field.Name + "NotIn", ColName: field.Name, Operation: define.OpNotIn},
			)

		case reflect.Struct:
			// 检查是否是时间类型
			if field.Type == reflect.TypeOf(time.Time{}) {
				defaultCondParams = append(defaultCondParams,
					ConditionParam{QueryName: field.Name + "Eq", ColName: field.Name, Operation: define.OpEq},
					ConditionParam{QueryName: field.Name + "Ne", ColName: field.Name, Operation: define.OpNe},
					ConditionParam{QueryName: field.Name + "Gt", ColName: field.Name, Operation: define.OpGt},
					ConditionParam{QueryName: field.Name + "Ge", ColName: field.Name, Operation: define.OpGe},
					ConditionParam{QueryName: field.Name + "Lt", ColName: field.Name, Operation: define.OpLt},
					ConditionParam{QueryName: field.Name + "Le", ColName: field.Name, Operation: define.OpLe},
				)
			} else {
				// 普通结构体类型：只支持等于和不等于
				defaultCondParams = append(defaultCondParams,
					ConditionParam{QueryName: field.Name + "Eq", ColName: field.Name, Operation: define.OpEq},
					ConditionParam{QueryName: field.Name + "Ne", ColName: field.Name, Operation: define.OpNe},
				)
			}

		default:
			// 其他类型：默认只支持等于和不等于
			defaultCondParams = append(defaultCondParams,
				ConditionParam{QueryName: field.Name + "Eq", ColName: field.Name, Operation: define.OpEq},
				ConditionParam{QueryName: field.Name + "Ne", ColName: field.Name, Operation: define.OpNe},
			)
		}
	}

	// 调用 NewCrud2 创建路由处理器
	return NewCrud2(
		tableName,         // 表名作为路由前缀
		model,             // 模型对象
		db,                // 数据库连接
		fields,            // 查询字段
		defaultCondParams, // 查询条件参数
		fields,            // 详情查询字段
		defaultCondParams, // 详情查询条件参数
		fields,            // 插入字段
		fields,            // 更新字段
		defaultCondParams, // 更新条件参数
		defaultCondParams, // 删除条件参数
	)
}

func QueryPage() gin.HandlerFunc {
	return func(c *gin.Context) {
		db, ok := GetContextDatabase(c)
		if !ok {
			panic("can't find database")
		}
		i, ok := GetContextEntity(c)
		if !ok {
			panic("can't find data entity")
		}

		// 获取条件
		cond := buildWhereCondition(c)

		// 获取分页参数
		page := 1
		size := 10
		if p, err := strconv.Atoi(c.DefaultQuery("page", "1")); err == nil {
			page = p
		}
		if s, err := strconv.Atoi(c.DefaultQuery("size", "10")); err == nil {
			size = s
		}

		// 获取排序参数
		sort := c.DefaultQuery("sort", "")

		// 构建查询
		chain := db.Chain().Table(getTableName(i))
		if cond != nil {
			chain = chain.Where(cond.Field, cond.Op, cond.Value)
		}

		// 应用排序
		if sort != "" {
			if strings.HasPrefix(sort, "-") {
				chain = chain.OrderByDesc(strings.TrimPrefix(sort, "-"))
			} else {
				chain = chain.OrderBy(sort)
			}
		}

		// 设置分页
		chain = chain.Page(page, size)

		// 执行查询
		result := chain.List()
		if result.Error != nil {
			RenderErrs(c, result.Error)
			return
		}

		// 获取总数
		total, err := chain.Count()
		if err != nil {
			RenderErrs(c, err)
			return
		}

		// 返回分页结果
		pageResult := gin.H{
			"total": total,
			"page":  page,
			"size":  size,
			"data":  result.Data,
		}
		SetContextEntity(pageResult)(c)
	}
}

// 辅助函数
func getTableName(i interface{}) string {
	if t, ok := i.(interface{ TableName() string }); ok {
		return t.TableName()
	}
	return ""
}

func getSelectColumns(c *gin.Context) []string {
	cols := make([]string, 0)
	if cc, ok := GetContextAny(c, "cols"); ok {
		cols = cc.([]string)
	}
	return cols
}

func buildWhereCondition(c *gin.Context) *define.Condition {
	if cnd, ok := getContextCondition(c); ok && cnd != nil {
		var condition *define.Condition
		for field, payload := range cnd.PayLoads {
			var newCond *define.Condition
			switch payload.Type {
			case define.OpEq:
				newCond = define.Eq(field, payload.Value)
			case define.OpNe:
				newCond = define.Ne(field, payload.Value)
			case define.OpGt:
				newCond = define.Gt(field, payload.Value)
			case define.OpGe:
				newCond = define.Ge(field, payload.Value)
			case define.OpLt:
				newCond = define.Lt(field, payload.Value)
			case define.OpLe:
				newCond = define.Le(field, payload.Value)
			case define.OpLike:
				newCond = define.Like(field, payload.Value)
			case define.OpNotLike:
				newCond = define.NotLike(field, payload.Value)
			case define.OpIn:
				if slice, ok := payload.Value.([]interface{}); ok {
					newCond = define.In(field, slice...)
				}
			case define.OpNotIn:
				if slice, ok := payload.Value.([]interface{}); ok {
					newCond = define.NotIn(field, slice...)
				}
			}
			if newCond != nil {
				if condition == nil {
					condition = newCond
				} else {
					condition = condition.And(newCond)
				}
			}
		}
		return condition
	}
	return nil
}

func RenderJSON(c *gin.Context) {
	results, ok := GetContextEntity(c)
	if !ok {
		RenderErr2(c, 0, "can't find result")
		return
	}
	// 处理指针类型
	if v := reflect.ValueOf(results); v.Kind() == reflect.Ptr {
		results = v.Elem().Interface()
	}
	// 设置状态码并输出JSON响应
	RenderOk(c, results)
}
func RenderJSONP(c *gin.Context) {
	results, ok := GetContextEntity(c)
	if ok {
		c.JSONP(200, results)
	} else {
		c.JSONP(200, nil)
	}
}

var Operators = []string{"Eq", "Le", "Lt", "Ge", "Gt", "Like", "LikeLeft", "LikeRight", "In", "NotIn", "NotLike", "NotEq"}

func MapToParamCondition(c *gin.Context, conditionParams []ConditionParam) (*define.Condition, map[string]interface{}, error) {
	maps, err := GetMapFromRst(c)
	hasValMap := make(map[string]string)
	if err != nil {
		return nil, nil, err
	}
	if len(maps) > 0 && len(conditionParams) > 0 {
		var cnd *define.Condition
		for _, param := range conditionParams {
			oldName, hasOldVal := hasValMap[param.QueryName]
			if hasOldVal {
				return nil, nil, fmt.Errorf("u have a query condition like [%s]", oldName)
			}
			for _, oper := range Operators {
				val, hasVal := maps[param.QueryName+oper]
				if hasVal {
					hasValMap[param.ColName] = param.QueryName + oper
					var newCond *define.Condition
					switch oper {
					case "Eq":
						newCond = define.Eq(param.ColName, val)
					case "NotEq":
						newCond = define.Ne(param.ColName, val)
					case "Le":
						newCond = define.Le(param.ColName, val)
					case "Lt":
						newCond = define.Lt(param.ColName, val)
					case "Ge":
						newCond = define.Ge(param.ColName, val)
					case "Gt":
						newCond = define.Gt(param.ColName, val)
					case "Like":
						newCond = define.Like(param.ColName, val)
					case "LikeLeft":
						newCond = define.Like(param.ColName, "%"+val.(string))
					case "LikeRight":
						newCond = define.Like(param.ColName, val.(string)+"%")
					case "In":
						if slice, ok := val.([]interface{}); ok {
							newCond = define.In(param.ColName, slice...)
						}
					case "NotIn":
						if slice, ok := val.([]interface{}); ok {
							newCond = define.NotIn(param.ColName, slice...)
						}
					case "NotLike":
						newCond = define.NotLike(param.ColName, val)
					}
					if newCond != nil {
						if cnd == nil {
							cnd = newCond
						} else {
							cnd = cnd.And(newCond)
						}
					}
				}
			}
		}
		if cnd != nil {
			return cnd, maps, nil
		}
		return nil, nil, nil
	}
	return nil, nil, nil
}

func DoInsert() gin.HandlerFunc {
	return func(c *gin.Context) {
		db, ok := GetContextDatabase(c)
		if !ok {
			RenderErr2(c, 0, "can't find database")
			return
		}
		i, ok := GetContextEntity(c)
		if !ok {
			RenderErr2(c, 0, "can't find data entity")
			return
		}

		// 执行插入操作
		chain := db.Chain().Table(getTableName(i))
		result := chain.Save(i)
		if result.Error != nil {
			RenderErr2(c, 0, result.Error.Error())
			return
		}

		// 获取插入后的ID
		val := reflect.ValueOf(i)
		if val.Kind() == reflect.Ptr {
			val = val.Elem()
		}
		idField := val.FieldByName("ID")
		if !idField.IsValid() {
			RenderErr2(c, 0, "no ID field found")
			return
		}

		// 查询完整记录
		result = chain.Where("id", define.OpEq, idField.Interface()).First()
		if result.Error != nil {
			RenderErr2(c, 0, result.Error.Error())
			return
		}

		// 将结果数据复制回原始实体
		if err := result.Into(i); err != nil {
			RenderErr2(c, 0, err.Error())
			return
		}

		// 返回更新后的结构体
		RenderOk(c, i)
	}
}

func DoUpdate() gin.HandlerFunc {
	return func(c *gin.Context) {
		db, ok := GetContextDatabase(c)
		if !ok {
			RenderErr2(c, 0, "can't find database")
			return
		}
		i, ok := GetContextEntity(c)
		if !ok {
			RenderErr2(c, 0, "can't find data entity")
			return
		}

		// 获取主键值
		val := reflect.ValueOf(i)
		if val.Kind() == reflect.Ptr {
			val = val.Elem()
		}

		idField := val.FieldByName("ID")
		if !idField.IsValid() {
			RenderErr2(c, 0, "no ID field found")
			return
		}

		// 执行更新操作
		chain := db.Chain().Table(getTableName(i))
		result := chain.Where("id", define.OpEq, idField.Interface()).Update(i)
		if result.Error != nil {
			RenderErr2(c, 0, result.Error.Error())
			return
		}

		// 查询更新后的完整记录
		result = chain.Where("id", define.OpEq, idField.Interface()).First()
		if result.Error != nil {
			RenderErr2(c, 0, result.Error.Error())
			return
		}

		// 将结果数据复制回原始实体
		if err := result.Into(i); err != nil {
			RenderErr2(c, 0, err.Error())
			return
		}

		RenderOk(c, i)
	}
}

func DoDelete() gin.HandlerFunc {
	return func(c *gin.Context) {
		db, ok := GetContextDatabase(c)
		if !ok {
			RenderErr2(c, 0, "can't find database")
			return
		}
		i, ok := GetContextEntity(c)
		if !ok {
			RenderErr2(c, 0, "can't find data entity")
			return
		}

		chain := db.Chain().Table(getTableName(i)).From(i)
		result := chain.Delete()
		if result.Error != nil {
			RenderErr2(c, 0, result.Error.Error())
			return
		}

		RenderOk(c, nil)
	}
}

func QueryList() gin.HandlerFunc {
	return func(c *gin.Context) {
		db, ok := GetContextDatabase(c)
		if !ok {
			RenderErr2(c, 0, "can't find database")
			return
		}
		i, ok := GetContextEntity(c)
		if !ok {
			RenderErr2(c, 0, "can't find data entity")
			return
		}

		// 获取分页参数
		pageNum := getContextPageNumber(c)
		pageSize := getContextPageSize(c)

		// 获取条件
		cond := buildWhereCondition(c)

		// 获取要查询的字段
		cols := getSelectColumns(c)

		// 执行查询
		chain := db.Chain().Table(getTableName(i))
		if len(cols) > 0 {
			chain = chain.Fields(cols...)
		}
		if cond != nil {
			chain = chain.Where2(cond)
		}

		// 执行分页查询
		result, er := chain.Page(pageNum, pageSize).PageInfo()
		if er != nil {
			RenderErr2(c, 0, er.Error())
			return
		}

		// 创建一个新的切片来存储结果
		sliceType := reflect.SliceOf(reflect.TypeOf(i).Elem())
		slice := reflect.MakeSlice(sliceType, 0, len(result.List.([]map[string]interface{})))
		slicePtr := reflect.New(sliceType)
		slicePtr.Elem().Set(slice)

		// 将结果数据转换为目标类型
		for _, item := range result.List.([]map[string]interface{}) {
			// 创建一个新的结构体实例
			newStruct := reflect.New(reflect.TypeOf(i).Elem()).Interface()

			// 将map数据转换为JSON
			jsonData, err := json.Marshal(item)
			if err != nil {
				RenderErr2(c, 0, err.Error())
				return
			}

			// 将JSON解析为结构体
			if err := json.Unmarshal(jsonData, newStruct); err != nil {
				RenderErr2(c, 0, err.Error())
				return
			}

			// 将结构体添加到切片中
			slice = reflect.Append(slice, reflect.ValueOf(newStruct).Elem())
		}
		slicePtr.Elem().Set(slice)

		// 设置响应数据
		pageResult := PageInfo{
			PageNum:    int64(result.PageNum),
			PageSize:   int64(result.PageSize),
			TotalSize:  result.Total,
			TotalPages: int64(result.Pages),
			Data:       slicePtr.Interface(),
		}
		RenderOk(c, pageResult)
	}
}

func QuerySingle() gin.HandlerFunc {
	return func(c *gin.Context) {
		db, ok := GetContextDatabase(c)
		if !ok {
			RenderErr2(c, 0, "can't find database")
			return
		}
		i, ok := GetContextEntity(c)
		if !ok {
			RenderErr2(c, 0, "can't find data entity")
			return
		}

		// 获取条件
		cond := buildWhereCondition(c)

		// 获取要查询的字段
		cols := getSelectColumns(c)

		// 执行查询
		chain := db.Chain().Table(getTableName(i))
		if len(cols) > 0 {
			chain = chain.Fields(cols...)
		}
		if cond != nil {
			chain = chain.Where2(cond)
		}

		result := chain.First()
		if result.Error != nil {
			if result.Error.Error() == "sql: no rows in result set" {
				RenderOk(c, nil)
				return
			}
			RenderErr2(c, 0, result.Error.Error())
			return
		}

		// 创建一个新的结构体实例
		newStruct := reflect.New(reflect.TypeOf(i).Elem()).Interface()

		// 将结果转换为目标类型
		if err := result.Into(newStruct); err != nil {
			RenderErr2(c, 0, err.Error())
			return
		}
		RenderOk(c, newStruct)
	}
}

func getContextCondition(c *gin.Context) (*Condition, bool) {
	i, ok := GetContextAny(c, "cnd")
	if ok {
		if cnd, ok := i.(Condition); ok {
			return &cnd, true
		}
	}
	return nil, false
}

func SetContextCondition(cnd *Condition) gin.HandlerFunc {
	return SetContextAny("cnd", cnd)
}

func GetContextDatabase(c *gin.Context) (*gom.DB, bool) {
	i, ok := GetContextAny(c, "db")
	if ok {
		return i.(*gom.DB), ok
	}
	return nil, ok
}
func SetContextDatabase(db *gom.DB) gin.HandlerFunc {
	return SetContextAny("db", db)
}
func SetContextAny(name string, i any) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set(prefix+name, i)
	}
}
func GetContextAny(c *gin.Context, name string) (i any, ok bool) {
	return c.Get(prefix + name)
}

func DoTableStruct() gin.HandlerFunc {
	return func(c *gin.Context) {
		db, ok := GetContextDatabase(c)
		if !ok {
			RenderErr2(c, 0, "can't find database")
			return
		}

		i, ok := GetContextEntity(c)
		if !ok {
			RenderErr2(c, 0, "can't find data entity")
			return
		}

		tableStruct, er := db.GetTableStruct2(i)
		if er != nil {
			RenderErr2(c, 0, er.Error())
			return
		}
		RenderOk(c, tableStruct)
	}
}
