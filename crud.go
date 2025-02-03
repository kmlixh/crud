package crud

import "C"
import (
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
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

func GetRouteHandler2(prefix string, i any, db *gom.DB, queryCols []string, queryConditionParam []ConditionParam, queryDetailCols []string, detailConditionParam []ConditionParam, insertCols []string, updateCols []string, updateConditionParam []ConditionParam, deleteConditionParam []ConditionParam) (ICrud, error) {
	listHandler := GetQueryListHandler(SetContextDatabase(db), SetContextEntity(i), DoNothingFunc, SetConditionParamAsCnd(queryConditionParam), SetColumns(queryCols), DefaultGenPageFromRstQuery, DoNothingFunc)
	detailHandler := GetQuerySingleHandler(SetContextDatabase(db), SetContextEntity(i), DoNothingFunc, SetConditionParamAsCnd(detailConditionParam), SetColumns(queryDetailCols), DoNothingFunc, DoNothingFunc)
	insertHandler := GetInsertHandler(SetContextDatabase(db), DoNothingFunc, DefaultUnMarshFunc(i), DoNothingFunc, SetColumns(insertCols), DoNothingFunc, DoNothingFunc)
	updateHandler := GetUpdateHandler(SetContextDatabase(db), DoNothingFunc, DefaultUnMarshFunc(i), SetConditionParamAsCnd(updateConditionParam), SetColumns(updateCols), DoNothingFunc, DoNothingFunc)
	deleteHandler := GetDeleteHandler(SetContextDatabase(db), SetContextEntity(i), DoNothingFunc, SetConditionParamAsCnd(deleteConditionParam), DoNothingFunc, DoNothingFunc, DoNothingFunc)
	tableStructHandler := GetTableStructHandler(SetContextDatabase(db), SetContextEntity(i), DoNothingFunc, DoNothingFunc, DoNothingFunc, DoNothingFunc, DoNothingFunc)
	return GenHandlerRegister(prefix, listHandler, insertHandler, detailHandler, updateHandler, deleteHandler, tableStructHandler)
}

func GetQueryListHandler(beforeCommitFunc ...gin.HandlerFunc) RouteHandler {
	return GetRouteHandler(string(PathList), "GET", append(beforeCommitFunc, QueryList(), RenderJSON)...)
}

func GetQuerySingleHandler(beforeCommitFunc ...gin.HandlerFunc) RouteHandler {
	return GetRouteHandler(string(PathDetail), "GET", append(beforeCommitFunc, QuerySingle(), RenderJSON)...)
}

func GetInsertHandler(beforeCommitFunc ...gin.HandlerFunc) RouteHandler {
	return GetRouteHandler(string(PathAdd), "POST", append(beforeCommitFunc, DoInsert(), RenderJSON)...)
}

func GetUpdateHandler(beforeCommitFunc ...gin.HandlerFunc) RouteHandler {
	return GetRouteHandler(string(PathUpdate), "POST", append(beforeCommitFunc, DoUpdate(), RenderJSON)...)
}

func GetDeleteHandler(beforeCommitFunc ...gin.HandlerFunc) RouteHandler {
	return GetRouteHandler(string(PathDelete), "POST", append(beforeCommitFunc, DoDelete(), RenderJSON)...)
}

func GetTableStructHandler(beforeCommitFunc ...gin.HandlerFunc) RouteHandler {
	return GetRouteHandler(string(PathTableStruct), "GET", append(beforeCommitFunc, DoTableStruct(), RenderJSON)...)
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
		return RouteHandler{}, errors.New(fmt.Sprintf("handler [%s] not found", name))
	} else {
		return h.Handlers[idx], nil
	}
}
func (h Crud) DeleteHandler(name string) error {
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
func (h Crud) AppendHandler(name string, handler gin.HandlerFunc, appendType HandlerAppendType, position HandlerPosition) error {
	_, ok := h.IdxMap[name]
	var routeHandler RouteHandler
	if !ok {
		return errors.New(fmt.Sprintf("handler [%s] not found", name))
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
	if prefix != nil && len(prefix) == 1 {
		name = prefix[0]
	}
	if h.Handlers == nil || len(h.Handlers) == 0 {
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
	if handlers == nil || len(handlers) == 0 {
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

type CrudDB struct {
	db        *gom.DB
	tableName string
	model     interface{}
}

func NewCrud(db *gom.DB, tableName string, model interface{}) *CrudDB {
	return &CrudDB{
		db:        db,
		tableName: tableName,
		model:     model,
	}
}

func (c *CrudDB) DoDelete(i interface{}) error {
	chain := c.db.Chain().Table(c.tableName)
	if id, ok := i.(int64); ok {
		chain = chain.Where("id", define.OpEq, id)
	} else {
		chain = chain.From(i)
	}
	result := chain.Delete()
	if result.Error != nil {
		return result.Error
	}
	return nil
}

func (c *CrudDB) DoUpdate(i interface{}) error {
	chain := c.db.Chain().Table(c.tableName).From(i)
	result := chain.Save()
	if result.Error != nil {
		return result.Error
	}
	return nil
}

func (c *CrudDB) DoInsert(i interface{}) error {
	// 获取结构体的值
	v := reflect.ValueOf(i)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	// 创建map来存储字段值
	fields := make(map[string]interface{})
	t := v.Type()
	for i := 0; i < v.NumField(); i++ {
		field := t.Field(i)
		// 跳过ID字段，因为它是自增的
		if field.Name != "ID" {
			fields[ToSnakeCase(field.Name)] = v.Field(i).Interface()
		}
	}

	// 执行插入操作
	chain := c.db.Chain().Table(c.tableName)
	result := chain.Save(fields)
	if result.Error != nil {
		return result.Error
	}

	// 获取插入后的ID
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}

	// 设置ID值到结构体
	if idField := v.FieldByName("ID"); idField.IsValid() && idField.CanSet() {
		idField.SetInt(id)
	}

	return nil
}

func (c *CrudDB) QuerySingle(i interface{}, id int64) error {
	chain := c.db.Chain().Table(c.tableName).Where("id", define.OpEq, id)
	result := chain.First()
	if result.Error != nil {
		return result.Error
	}
	return result.Into(i)
}

func (c *CrudDB) QueryList(i interface{}, page, size int, sort string) (int64, error) {
	chain := c.db.Chain().Table(c.tableName)

	// Apply sorting
	if sort != "" {
		if strings.HasPrefix(sort, "-") {
			chain = chain.OrderByDesc(strings.TrimPrefix(sort, "-"))
		} else {
			chain = chain.OrderBy(sort)
		}
	}

	// Get total count
	total, err := chain.Count()
	if err != nil {
		return 0, err
	}

	// Apply pagination
	if page > 0 && size > 0 {
		offset := (page - 1) * size
		chain = chain.Offset(offset).Limit(size)
	}

	// Execute query
	result := chain.List()
	if result.Error != nil {
		return 0, result.Error
	}

	err = result.Into(i)
	if err != nil {
		return 0, err
	}

	return total, nil
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
		RenderErr2(c, 500, "can't find result")
		return
	}
	// 处理指针类型
	if v := reflect.ValueOf(results); v.Kind() == reflect.Ptr {
		results = v.Elem().Interface()
	}
	// 设置状态码并输出JSON响应
	c.JSON(200, RawCodeMsg(200, "ok", results))
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
				return nil, nil, errors.New(fmt.Sprintf("u have a query condition like [%s]", oldName))
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
			panic("can't find database")
		}
		i, ok := GetContextEntity(c)
		if !ok {
			panic("can't find data entity")
		}

		// 执行插入操作
		result := db.Chain().From(i).Save()
		if result.Error != nil {
			RenderErrs(c, result.Error)
			c.Abort()
			return
		}

		// 返回更新后的结构体
		SetContextEntity(result)(c)
	}
}

func DoUpdate() gin.HandlerFunc {
	return func(c *gin.Context) {
		db, ok := GetContextDatabase(c)
		if !ok {
			panic("can't find database")
		}
		i, ok := GetContextEntity(c)
		if !ok {
			panic("can't find data entity")
		}

		chain := db.Chain().Table(getTableName(i)).From(i)
		result := chain.Update()
		if result.Error != nil {
			RenderErrs(c, result.Error)
			return
		}
		SetContextEntity(result)(c)
	}
}

func DoDelete() gin.HandlerFunc {
	return func(c *gin.Context) {
		db, ok := GetContextDatabase(c)
		if !ok {
			panic("can't find database")
		}
		i, ok := GetContextEntity(c)
		if !ok {
			panic("can't find data entity")
		}

		chain := db.Chain().Table(getTableName(i)).From(i)
		result := chain.Delete()
		if result.Error != nil {
			RenderErrs(c, result.Error)
			return
		}
		SetContextEntity(result)(c)
	}
}

func DoTableStruct() gin.HandlerFunc {
	return func(c *gin.Context) {
		db, ok := GetContextDatabase(c)
		if !ok {
			panic("can't find database")
		}

		i, ok := GetContextEntity(c)
		if !ok {
			panic("can't find data entity")
		}

		tableStruct, er := db.GetTableStruct2(i)
		if er != nil {
			RenderErrs(c, er)
			return
		}
		SetContextEntity(tableStruct)(c)
	}
}

func QueryList() gin.HandlerFunc {
	return func(c *gin.Context) {
		db, ok := GetContextDatabase(c)
		if !ok {
			panic("can't find database")
		}
		i, ok := GetContextEntity(c)
		if !ok {
			panic("can't find data entity")
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

		// 应用分页
		offset := (pageNum - 1) * pageSize
		chain = chain.Offset(offset).Limit(pageSize)

		// 执行分页查询
		result, er := chain.Page(pageNum, pageSize).PageInfo()
		if er != nil {
			RenderErr2(c, 500, er.Error())
			c.Abort()
			return
		}

		// 设置响应数据
		SetContextEntity(result)(c)

	}
}

func QuerySingle() gin.HandlerFunc {
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
			RenderErrs(c, result.Error)
			return
		}

		// 创建一个新的结构体实例
		newStruct := reflect.New(reflect.TypeOf(i).Elem()).Interface()

		// 将结果转换为目标类型
		if err := result.Into(newStruct); err != nil {
			RenderErrs(c, err)
			c.Abort()
			return
		}
		SetContextEntity(newStruct)(c)
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
