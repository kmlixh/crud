package AutoCrudGo

import (
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/kmlixh/gom/v2"
	"reflect"
	"time"
)

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
)

type QueryParameter struct {
	QueryName string
	ColName   string
	Required  bool
	QueryOpera
}

func DefaultQueryList(i interface{}, db *gom.DB, queryParam []QueryParameter, columns []string) gin.HandlerFunc {
	return func(c *gin.Context) {
		results := reflect.New(reflect.SliceOf(reflect.TypeOf(i))).Interface()
		cnd, err := MapToParamCondition(c, queryParam)
		if cnd != nil && err == nil {
			db.Where(cnd)
		}
		_, err = db.Select(results, columns...)
		if err != nil {
			RenderErr2(c, 500, "服务器内部错误："+err.Error())
			return
		}

	}
}

func MapToParamCondition(c *gin.Context, queryParam []QueryParameter) (gom.Condition, error) {
	maps, err := GetConditionMapFromRst(c)
	if err != nil {
		return nil, err
	}
	if len(maps) > 0 && len(queryParam) > 0 {
		var cnd = gom.CndEmpty()
		for _, param := range queryParam {
			if param.Required && maps[param.QueryName] == nil {
				return nil, errors.New(fmt.Sprintf("%s is requied!", param.QueryName))
			}
			val := maps[param.QueryName]
			if val != nil {
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
		return cnd, nil
	} else {
		return nil, nil
	}
}
