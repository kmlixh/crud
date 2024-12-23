package crud

import "C"
import (
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"unicode"

	"github.com/gin-gonic/gin"
	"github.com/kmlixh/gom/v3"
	"github.com/kmlixh/gom/v3/define"
)

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
		err := context.ShouldBind(i)
		if err != nil {
			context.Abort()
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
		if er == nil {
			if cnd != nil {
				c.Set(prefix+"cnd", cnd)
			}
		} else {
			c.Abort()
		}
	}
}

func getContextCondition(c *gin.Context) (define.Condition, bool) {
	i, ok := GetContextAny(c, "cnd")
	if ok {
		return i.(define.Condition), ok
	}
	return nil, ok
}
func SetContextCondition(cnd define.Condition) gin.HandlerFunc {
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

func (d DefaultRoutePath) String() string {
	return string(d)
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
func (h HandlerRegister) AppendHandler(name string, handler gin.HandlerFunc, appendType HandlerAppendType, position HandlerPosition) error {
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

func (h HandlerRegister) Register(routes gin.IRoutes) error {
	if h.Handlers == nil || len(h.Handlers) == 0 {
		return errors.New("route handler could not be empty or nil")
	}
	for _, handler := range h.Handlers {
		if strings.ToLower(handler.HttpMethod) != "any" {
			routes.Handle(handler.HttpMethod, h.Name+"/"+handler.Path, handler.Handlers...)
		} else {
			routes.Any(h.Name+"/"+handler.Path, handler.Handlers...)
		}
	}
	return nil
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

func GetAutoRouteHandler(prefix string, i any, db *gom.DB) (IHandlerRegister, error) {
	columnNames, primaryKeys, primaryAuto, columnIdxMap := gom.GetColumns(reflect.Indirect(reflect.ValueOf(i)))
	queryCols := append(primaryKeys, append(primaryAuto, columnNames...)...)

	if len(columnNames) > 0 {
		listHandler := GetQueryListHandler(SetContextDatabase(db), SetContextEntity(i), DoNothingFunc, SetConditionParamAsCnd(GetConditionParam(queryCols, columnIdxMap, i)), SetColumns(queryCols), DefaultGenPageFromRstQuery, DoNothingFunc)
		detailHandler := GetQuerySingleHandler(SetContextDatabase(db), SetContextEntity(i), DoNothingFunc, SetConditionParamAsCnd(GetConditionParam(queryCols, columnIdxMap, i)), SetColumns(queryCols), DoNothingFunc, DoNothingFunc)
		insertHandler := GetInsertHandler(SetContextDatabase(db), DoNothingFunc, DefaultUnMarshFunc(i), DoNothingFunc, SetColumns(queryCols), DoNothingFunc, DoNothingFunc)
		updateHandler := GetUpdateHandler(SetContextDatabase(db), DoNothingFunc, DefaultUnMarshFunc(i), SetConditionParamAsCnd(GetConditionParam(append(primaryKeys, primaryAuto...), columnIdxMap, i)), SetColumns(append(primaryKeys, columnNames...)), DoNothingFunc, DoNothingFunc)
		deleteHandler := GetDeleteHandler(SetContextDatabase(db), SetContextEntity(i), DoNothingFunc, SetConditionParamAsCnd(GetConditionParam(append(primaryKeys, primaryAuto...), columnIdxMap, i)), DoNothingFunc, DoNothingFunc, DoNothingFunc)
		tableStructHandler := GetTableStructHandler(SetContextDatabase(db), SetContextEntity(i), DoNothingFunc, DoNothingFunc, DoNothingFunc, DoNothingFunc, DoNothingFunc)
		return GenHandlerRegister(prefix, listHandler, insertHandler, detailHandler, updateHandler, deleteHandler, tableStructHandler, GetAutoTableViewRouteHandler(TableConfig.String(), db, i))
	} else {
		return nil, errors.New("Struct was empty")
	}
}
func GetRouteHandler2(prefix string, i any, db *gom.DB, queryCols []string, queryConditionParam []ConditionParam, queryDetailCols []string, detailConditionParam []ConditionParam, insertCols []string, updateCols []string, updateConditionParam []ConditionParam, deleteConditionParam []ConditionParam) (IHandlerRegister, error) {
	listHandler := GetQueryListHandler(SetContextDatabase(db), SetContextEntity(i), DoNothingFunc, SetConditionParamAsCnd(queryConditionParam), SetColumns(queryCols), DefaultGenPageFromRstQuery, DoNothingFunc)
	detailHandler := GetQuerySingleHandler(SetContextDatabase(db), SetContextEntity(i), DoNothingFunc, SetConditionParamAsCnd(detailConditionParam), SetColumns(queryDetailCols), DoNothingFunc, DoNothingFunc)
	insertHandler := GetInsertHandler(SetContextDatabase(db), DoNothingFunc, DefaultUnMarshFunc(i), DoNothingFunc, SetColumns(insertCols), DoNothingFunc, DoNothingFunc)
	updateHandler := GetUpdateHandler(SetContextDatabase(db), DoNothingFunc, DefaultUnMarshFunc(i), SetConditionParamAsCnd(updateConditionParam), SetColumns(updateCols), DoNothingFunc, DoNothingFunc)
	deleteHandler := GetDeleteHandler(SetContextDatabase(db), SetContextEntity(i), DoNothingFunc, SetConditionParamAsCnd(deleteConditionParam), DoNothingFunc, DoNothingFunc, DoNothingFunc)
	tableStructHandler := GetTableStructHandler(SetContextDatabase(db), SetContextEntity(i), DoNothingFunc, DoNothingFunc, DoNothingFunc, DoNothingFunc, DoNothingFunc)
	return GenHandlerRegister(prefix, listHandler, insertHandler, detailHandler, updateHandler, deleteHandler, tableStructHandler, GetAutoTableViewRouteHandler(TableConfig.String(), db, i))
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
		Operation: define.Eq,
	}
}
func GetFieldDefaultQueryName(f reflect.StructField) string {
	return ToCamelCaseWithRegex(f.Name)
}

func GetQueryListHandler(beforeCommitFunc ...gin.HandlerFunc) RouteHandler {
	return GetRouteHandler(string(PathList), "Any", append(beforeCommitFunc, QueryList(), RenderJSON)...)
}
func GetQuerySingleHandler(beforeCommitFunc ...gin.HandlerFunc) RouteHandler {
	return GetRouteHandler(string(PathDetail), http.MethodGet, append(beforeCommitFunc, QuerySingle(), RenderJSON)...)
}
func GetInsertHandler(beforeCommitFunc ...gin.HandlerFunc) RouteHandler {
	return GetRouteHandler(string(PathAdd), http.MethodPost, append(beforeCommitFunc, DoInsert(), RenderJSON)...)
}
func GetUpdateHandler(beforeCommitFunc ...gin.HandlerFunc) RouteHandler {
	return GetRouteHandler(string(PathUpdate), http.MethodPost, append(beforeCommitFunc, DoUpdate(), RenderJSONP)...)
}
func GetDeleteHandler(beforeCommitFunc ...gin.HandlerFunc) RouteHandler {
	return GetRouteHandler(string(PathDelete), http.MethodDelete, append(beforeCommitFunc, DoDelete(), RenderJSON)...)
}
func GetTableStructHandler(beforeCommitFunc ...gin.HandlerFunc) RouteHandler {
	return GetRouteHandler(string(PathTableStruct), http.MethodDelete, append(beforeCommitFunc, DoTableStruct(), RenderJSONP)...)
}
func GetRouteHandler(path string, method string, handlers ...gin.HandlerFunc) RouteHandler {
	return RouteHandler{
		Path:       path,
		HttpMethod: method,
		Handlers:   handlers,
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

func DoDelete() gin.HandlerFunc {
	return func(c *gin.Context) {
		db, ok := GetContextDatabase(c)
		if !ok {
			panic("can't find database")
		}
		cnd, ok := getContextCondition(c)
		if !ok || cnd == nil {
			RenderErrs(c, errors.New("can't get Cnd"))
			return
		}
		i, ok := GetContextEntity(c)
		if !ok {
			panic("can't find data entity")

		}
		rs, er := db.Where(cnd).Delete(i)
		if er != nil {
			RenderErrs(c, er)
			return
		}
		SetContextEntity(rs)(c)
	}
}
func DoUpdate() gin.HandlerFunc {
	return func(c *gin.Context) {
		db, ok := GetContextDatabase(c)
		if !ok {
			panic("can't find database")
		}
		cnd, ok := getContextCondition(c)
		if !ok || cnd == nil {
			RenderErrs(c, errors.New("can't get Cnd"))
			return
		}
		i, ok := GetContextEntity(c)
		if !ok {
			panic("can't find data entity")

		}
		cols := make([]string, 0)
		cc, ok := GetContextAny(c, "cols")
		if !ok {
			cols = cc.([]string)
		}
		rs, er := db.Where(cnd).Update(i, cols...)
		if er != nil {
			RenderErrs(c, er)
			return
		}
		SetContextEntity(rs)(c)
	}
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
		cols := make([]string, 0)
		cc, ok := GetContextAny(c, "cols")
		if !ok {
			cols = cc.([]string)
		}
		rs, er := db.Insert(i, cols...)
		if er != nil {
			RenderErrs(c, er)
			return
		}
		SetContextEntity(rs)(c)

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
		cnd, ok := getContextCondition(c)
		if ok && cnd != nil {
			db.Where(cnd)
		}
		rawInfo := gom.GetRawTableInfo(i)
		result := reflect.New(reflect.TypeOf(rawInfo.Type)).Interface()
		cols := make([]string, 0)
		cc, ok := GetContextAny(c, "cols")
		if !ok {
			cols = cc.([]string)
		}
		_, er := db.Select(&result, cols...)
		if er != nil {
			RenderErrs(c, er)
			return
		}
		SetContextEntity(result)(c)
	}
}
func QueryList() gin.HandlerFunc {
	return func(c *gin.Context) {
		i, ok := GetContextEntity(c)
		if !ok {
			panic("can't find data entity")
		}
		rawInfo := gom.GetRawTableInfo(i)
		results := reflect.Indirect(reflect.New(reflect.SliceOf(rawInfo.Type)))
		db, ok := GetContextDatabase(c)
		if !ok {
			panic("can't find database")
		}
		cnd, ok := getContextCondition(c)
		if ok && cnd != nil {
			db.Where(cnd)
		}
		counts, er := db.Table(rawInfo.TableName).Count("*")
		pageNum := int64(getContextPageNumber(c))
		pageSize := int64(getContextPageSize(c))
		cols := make([]string, 0)
		cc, ok := GetContextAny(c, "cols")
		if !ok {
			cols = cc.([]string)
		}
		db.Page(pageNum, pageSize)
		orderBys, ok := GetOrderBys(c)
		if ok {
			db.OrderBys(orderBys)
		}
		_, er = db.Select(results, cols...)
		if er != nil {
			c.Abort()
			RenderErrs(c, er)
			return
		}
		totalPages := counts / pageSize
		if counts%pageSize > 0 {
			totalPages++
		}
		page := PageInfo{pageNum, pageSize, counts, totalPages, results.Interface()}
		SetContextEntity(page)(c)
	}
}

func RenderJSON(c *gin.Context) {
	results, ok := GetContextEntity(c)
	if ok {
		RenderOk(c, results)
	} else {
		RenderErr2(c, 500, "can't find result")
	}
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

func MapToParamCondition(c *gin.Context, conditionParams []ConditionParam) (define.Condition, map[string]interface{}, error) {
	maps, err := GetMapFromRst(c)
	hasValMap := make(map[string]string)
	if err != nil {
		return nil, nil, err
	}
	if len(maps) > 0 && len(conditionParams) > 0 {
		var cnd = gom.CndEmpty()
		for _, param := range conditionParams {
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
						cnd.In(param.ColName, UnZipSlice(val)...)
					case "NotIn":
						cnd.NotIn(param.ColName, UnZipSlice(val)...)
					case "NotLike":
						cnd.NotLike(param.ColName, val)
					}
				}
			}

		}
		if cnd.PayLoads() > 0 {
			return cnd, maps, nil
		}
		return nil, nil, nil
	} else {
		return nil, nil, nil
	}
}

func AutoTableViewHandler(name string, db *gom.DB, i any) gin.HandlerFunc {
	return func(c *gin.Context) {
		tableInfo, e := GetDefaultTableInfo(name, db, i)
		if e != nil {
			c.Abort()
			RenderErrs(c, e)
			return
		}
		SetContextEntity(tableInfo)(c)
	}
}
func GetDefaultTableInfo(name string, db *gom.DB, i any) (TableInfo, error) {
	v := reflect.ValueOf(i)
	_, _, _, columnIdxMap := gom.GetColumns(v)
	tableStruct, e := db.GetTableStruct2(i)
	if e != nil {
		return TableInfo{}, e
	}
	tableCols := tableStruct.(define.TableStruct).Columns
	cols := make([]ColumnInfo, 0)
	for _, cc := range tableCols {
		if fieldName, ok := columnIdxMap[cc.ColumnName]; ok {
			col := ColumnInfo{
				Name:          cc.ColumnName,
				Title:         cc.Comment,
				DateType:      fieldName,
				Constraint:    Constraint{},
				Options:       nil,
				ColumnVisible: true,
				Searchable:    false,
				Editable:      cc.PrimaryAuto || cc.Primary,
			}
			cols = append(cols, col)
		} else {
			return TableInfo{}, errors.New("[" + cc.ColumnName + "] can't found In [" + v.Type().Name() + "]")
		}
	}
	tableInfo := TableInfo{
		Name:    name,
		Title:   tableStruct.GetTableComment(),
		Columns: cols,
	}
	return tableInfo, nil
}
func GetAutoTableViewRouteHandler(name string, db *gom.DB, i any) RouteHandler {
	return GetRouteHandler(name, "any", AutoTableViewHandler(name, db, i), func(c *gin.Context) {
		method := c.Request.Method
		if method == "JSONP" {
			RenderJSONP(c)
		} else {
			RenderJSON(c)
		}
	})
}

// TableRouteInfo 定义表路由信息
type TableRouteInfo struct {
	TableName      string            // 数据库表名
	RoutePath      string            // 路由路径
	PrimaryKeys    []string          // 主键列表
	ColumnNames    []string          // 列名列表
	QueryParams    map[string]string // 查询参数映射
	ExcludeColumns []string          // 排除的列
}

// GetTableRouteHandler 根据表信息自动生成CRUD路由处理器
func GetTableRouteHandler(db *gom.DB, info TableRouteInfo) (IHandlerRegister, error) {
	// 验证必要参数
	if info.TableName == "" || info.RoutePath == "" {
		return nil, errors.New("table name and route path are required")
	}

	// 如果未指定列,则获取表的所有列
	if len(info.ColumnNames) == 0 {
		tableStruct, err := db.GetTableStruct(info.TableName)
		if err != nil {
			return nil, fmt.Errorf("failed to get table columns: %v", err)
		}

		// Extract column names from TableStruct
		tableCols := tableStruct.(define.TableStruct).Columns
		for _, col := range tableCols {
			if !contains(info.ExcludeColumns, col.ColumnName) {
				info.ColumnNames = append(info.ColumnNames, col.ColumnName)
			}
		}
	}

	// 如果未指定主键,使用id作为默认主键
	if len(info.PrimaryKeys) == 0 {
		info.PrimaryKeys = []string{"id"}
	}

	// 生成各个处理器
	listHandler := GetTableListHandler(db, info)
	detailHandler := GetTableDetailHandler(db, info)
	insertHandler := GetTableInsertHandler(db, info)
	updateHandler := GetTableUpdateHandler(db, info)
	deleteHandler := GetTableDeleteHandler(db, info)

	return GenHandlerRegister(info.RoutePath,
		listHandler,
		detailHandler,
		insertHandler,
		updateHandler,
		deleteHandler,
	)
}

// GetTableListHandler 生成列表查询处理器
func GetTableListHandler(db *gom.DB, info TableRouteInfo) RouteHandler {
	return GetRouteHandler(string(PathList), http.MethodGet,
		SetContextDatabase(db),
		DefaultGenPageFromRstQuery,
		func(c *gin.Context) {
			query := db.Table(info.TableName)

			// 处理查询条件
			if params, err := GetMapFromRst(c); err == nil {
				for queryName, colName := range info.QueryParams {
					if val, exists := params[queryName]; exists {
						query = query.Where(gom.CndEq(colName, val))
					}
				}
			}

			// 分页查询
			pageNum := getContextPageNumber(c)
			pageSize := getContextPageSize(c)
			query = query.Page(int64(pageNum), int64(pageSize))

			// 执行查询
			var results []map[string]interface{}
			total, err := query.Select(&results, info.ColumnNames...)
			if err != nil {
				RenderErrs(c, err)
				return
			}
			totalInt64, ok := total.(int64)
			if !ok {
				RenderErrs(c, errors.New("invalid total count type"))
				return
			}

			// 返回分页数据
			totalPages := totalInt64 / int64(pageSize)
			if totalInt64%int64(pageSize) > 0 {
				totalPages++
			}

			SetContextEntity(PageInfo{
				PageNum:    int64(pageNum),
				PageSize:   int64(pageSize),
				TotalSize:  totalInt64,
				TotalPages: totalPages,
				Data:       results,
			})(c)
		},
		RenderJSON,
	)
}

// GetTableDetailHandler 生成详情查询处理器
func GetTableDetailHandler(db *gom.DB, info TableRouteInfo) RouteHandler {
	return GetRouteHandler(string(PathDetail), http.MethodGet,
		SetContextDatabase(db),
		func(c *gin.Context) {
			query := db.Table(info.TableName)

			// 处理主键条件
			for _, pk := range info.PrimaryKeys {
				if val := c.Query(pk); val != "" {
					query = query.Where(gom.CndEq(pk, val))
				}
			}

			// 执行查询
			var result map[string]interface{}
			_, err := query.Select(&result, info.ColumnNames...)
			if err != nil {
				RenderErrs(c, err)
				return
			}

			SetContextEntity(result)(c)
		},
		RenderJSON,
	)
}

// GetTableInsertHandler 生成新增处理器
func GetTableInsertHandler(db *gom.DB, info TableRouteInfo) RouteHandler {
	return GetRouteHandler(string(PathAdd), http.MethodPost,
		SetContextDatabase(db),
		func(c *gin.Context) {
			// 获取请求数据
			data, err := GetMapFromRst(c)
			if err != nil {
				RenderErrs(c, err)
				return
			}

			// 执行插入
			result, err := db.Table(info.TableName).Insert(data)
			if err != nil {
				RenderErrs(c, err)
				return
			}

			SetContextEntity(result)(c)
		},
		RenderJSON,
	)
}

// GetTableUpdateHandler 生成更新处理器
func GetTableUpdateHandler(db *gom.DB, info TableRouteInfo) RouteHandler {
	return GetRouteHandler(string(PathUpdate), http.MethodPost,
		SetContextDatabase(db),
		func(c *gin.Context) {
			// 获取请求数据
			data, err := GetMapFromRst(c)
			if err != nil {
				RenderErrs(c, err)
				return
			}

			query := db.Table(info.TableName)

			// 处理主键条件
			for _, pk := range info.PrimaryKeys {
				if val, exists := data[pk]; exists {
					query = query.Where(gom.CndEq(pk, val))
					delete(data, pk) // 从更新数据中移除主键
				}
			}

			// 执行更新
			result, err := query.Update(data)
			if err != nil {
				RenderErrs(c, err)
				return
			}

			SetContextEntity(result)(c)
		},
		RenderJSON,
	)
}

// GetTableDeleteHandler 生成删除处理器
func GetTableDeleteHandler(db *gom.DB, info TableRouteInfo) RouteHandler {
	return GetRouteHandler(string(PathDelete), http.MethodDelete,
		SetContextDatabase(db),
		func(c *gin.Context) {
			query := db.Table(info.TableName)

			// 处理主键条件
			for _, pk := range info.PrimaryKeys {
				if val := c.Query(pk); val != "" {
					query = query.Where(gom.CndEq(pk, val))
				}
			}

			// 执行删除
			result, err := query.Delete(nil)
			if err != nil {
				RenderErrs(c, err)
				return
			}

			SetContextEntity(result)(c)
		},
		RenderJSON,
	)
}

// 工具函数:检查slice中是否包含指定值
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func UnZipSlice(val interface{}) []interface{} {
	switch v := val.(type) {
	case []interface{}:
		return v
	case string:
		return []interface{}{v}
	default:
		return []interface{}{val}
	}
}
