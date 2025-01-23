package crud

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/kmlixh/gom/v4"
	"github.com/kmlixh/gom/v4/define"
	"github.com/stretchr/testify/assert"
)

// 测试用的数据模型
type TestModel struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

func (t TestModel) TableName() string {
	return "test_models"
}

func setupTestRouter() (*gin.Engine, *gom.DB) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	// 连接到本地测试数据库
	db, err := gom.Open("mysql", "root:123456@tcp(10.0.1.5:3306)/test?charset=utf8mb4&parseTime=True&loc=Local", &define.DBOptions{
		Debug: true,
	})
	if err != nil {
		panic(err)
	}

	return r, db
}

func TestCRUDOperations(t *testing.T) {
	// 设置测试环境
	r, db := setupTestRouter()
	defer db.Close()

	// 确保测试表存在
	result := db.Chain().RawExecute("CREATE TABLE IF NOT EXISTS test_models (id BIGINT AUTO_INCREMENT PRIMARY KEY, name VARCHAR(255))")
	assert.NoError(t, result.Error)

	// 清理测试数据
	defer func() {
		result := db.Chain().RawExecute("TRUNCATE TABLE test_models")
		assert.NoError(t, result.Error)
	}()

	model := &TestModel{}

	// 创建路由处理器
	crud, err := GetRouteHandler2(
		"test",
		model,
		db,
		[]string{"id", "name"},
		nil,
		[]string{"id", "name"},
		nil,
		[]string{"name"},
		[]string{"name"},
		nil,
		nil,
	)

	assert.NoError(t, err)
	assert.NotNil(t, crud)

	// 注册路由
	err = crud.Register(r.Group("/api"))
	assert.NoError(t, err)

	// 测试创建操作
	t.Run("Create", func(t *testing.T) {
		w := httptest.NewRecorder()
		body := `{"name":"test1"}`
		req := httptest.NewRequest("POST", "/api/test/add", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		data := response["data"].(map[string]interface{})
		assert.Equal(t, "test1", data["name"])
		assert.NotZero(t, data["id"])
	})

	// 测试查询列表
	t.Run("List", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/api/test/list?pageSize=10&pageNum=1", nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		data := response["data"].(map[string]interface{})
		assert.NotNil(t, data["data"])
	})

	// 测试查询单个
	t.Run("Detail", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/api/test/detail?id=1", nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		data := response["data"].(map[string]interface{})
		assert.NotNil(t, data)
		assert.Equal(t, float64(1), data["id"])
	})

	// 测试更新
	t.Run("Update", func(t *testing.T) {
		w := httptest.NewRecorder()
		body := `{"id":1,"name":"updated"}`
		req := httptest.NewRequest("POST", "/api/test/update", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		data := response["data"].(map[string]interface{})
		assert.Equal(t, "updated", data["name"])
	})

	// 测试删除
	t.Run("Delete", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("DELETE", "/api/test/delete?id=1", nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		// 验证记录已被删除
		w = httptest.NewRecorder()
		req = httptest.NewRequest("GET", "/api/test/detail?id=1", nil)
		r.ServeHTTP(w, req)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Nil(t, response["data"])
	})
}

func TestToCamelCaseWithRegex(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"hello_world", "helloWorld"},
		{"user_name", "userName"},
		{"id", "id"},
		{"", ""},
		{"multiple__underscores", "multipleUnderscores"},
	}

	for _, test := range tests {
		result := ToCamelCaseWithRegex(test.input)
		assert.Equal(t, test.expected, result)
	}
}

func TestToSnakeCase(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"helloWorld", "hello_world"},
		{"userName", "user_name"},
		{"ID", "i_d"},
		{"", ""},
		{"ABC", "a_b_c"},
	}

	for _, test := range tests {
		result := ToSnakeCase(test.input)
		assert.Equal(t, test.expected, result)
	}
}

func TestStructToMap(t *testing.T) {
	type TestStruct struct {
		ID   int
		Name string
	}

	test := TestStruct{
		ID:   1,
		Name: "test",
	}

	ok, result := StructToMap(test)
	assert.True(t, ok)
	assert.Equal(t, "int", result["ID"])
	assert.Equal(t, "string", result["Name"])

	// 测试非结构体输入
	ok, result = StructToMap(123)
	assert.False(t, ok)
	assert.Nil(t, result)
}
