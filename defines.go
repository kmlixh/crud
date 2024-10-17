package crud

import (
	"github.com/gin-gonic/gin"
	"github.com/kmlixh/gom/v3/define"
)

type DefaultRoutePath string

const (
	PathList        DefaultRoutePath = "list"
	PathAdd         DefaultRoutePath = "add"
	PathDetail      DefaultRoutePath = "detail"
	PathUpdate      DefaultRoutePath = "update"
	PathDelete      DefaultRoutePath = "delete"
	PathTableStruct DefaultRoutePath = "tableStruct"
)

func DoNothingFunc(c *gin.Context) {

}

type NameMethods int

func (n NameMethods) Original() int {
	return int(n)
}

const (
	Original NameMethods = iota
	CamelCase
	SnakeCase
)

type PageInfo struct {
	PageNum    int64 `json:"pageNum"`
	PageSize   int64 `json:"pageSize"`
	TotalSize  int64 `json:"totalSize"`
	TotalPages int64 `json:"totalPages"`
	Data       any   `json:"data"`
}
type HandlerAppendType int

const (
	Before HandlerAppendType = iota - 1
	Replace
	After
)

type IHandlerRegister interface {
	Register(routes gin.IRoutes) error
	AppendHandler(name string, handler gin.HandlerFunc, appendType HandlerAppendType, position HandlerPosition) error
	AddHandler(routeHandler RouteHandler) error
}
type RouteHandler struct {
	Path       string
	HttpMethod string
	Handlers   []gin.HandlerFunc
}
type ConditionParam struct {
	QueryName string
	ColName   string
	Operation define.Operation
}
type HandlerRegister struct {
	Name     string
	Handlers []RouteHandler
	IdxMap   map[string]int
}
type HandlerPosition int

const (
	Db HandlerPosition = iota
	Entity
	UnMarsh
	Cnd
	Columns
	Page
	OrderBys
	FinalOpera
	Renders
)

type TableInfo struct {
	Name    string
	Title   string
	Columns []ColumnInfo
}
type ColumnInfo struct {
	Name          string
	Title         string
	DateType      string
	Constraint    Constraint
	Options       []Option
	ColumnVisible bool
	Searchable    bool
	Editable      bool
}
type Constraint struct {
	InputType    string
	StepValue    string
	DefaultValue string
	MinValue     string
	MaxValue     string
}
type Option struct {
	Title string
	Key   string
	Value any
}
