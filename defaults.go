package AutoCrudGo

import (
	"github.com/gin-gonic/gin"
	"github.com/kmlixh/gom/v2"
	"reflect"
)

func DefaultQueryList(i interface{}, db *gom.DB, columns ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		results := reflect.New(reflect.SliceOf(reflect.TypeOf(i))).Interface()

		db.Select(results, columns...)

	}
}
