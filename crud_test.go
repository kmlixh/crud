package crud

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/kmlixh/gom/v4"
	"github.com/kmlixh/gom/v4/define"
	"github.com/stretchr/testify/assert"
)

// 测试用的数据模型
type TestModel struct {
	ID   int64  `json:"id" gom:"id,@"`
	Name string `json:"name" gom:"name"`
}

func (t TestModel) TableName() string {
	return "test_models"
}
func (t TestModel) CreateSql() string {
	return ""
}

func setupTestRouter() (*gin.Engine, *gom.DB) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	// 连接数据库
	db, err := gom.Open("mysql", DefaultMySQLConfig().DSN(), &define.DBOptions{
		Debug: true,
	})
	if err != nil {
		panic(err)
	}

	return r, db
}

// 定义测试模型

func TestCRUDOperations(t *testing.T) {
	// 设置测试环境
	r, db := setupTestRouter()
	defer db.Close()

	// 为每个测试用例生成唯一的表名
	tableName := fmt.Sprintf("test_models_%d", time.Now().UnixNano())

	// 确保测试表存在并清空数据
	result := db.Chain().RawExecute(fmt.Sprintf("DROP TABLE IF EXISTS %s", tableName))
	assert.NoError(t, result.Error)

	result = db.Chain().RawExecute(fmt.Sprintf("CREATE TABLE %s (id BIGINT AUTO_INCREMENT PRIMARY KEY, name VARCHAR(255))", tableName))
	assert.NoError(t, result.Error)

	// 清理测试数据
	defer func() {
		result := db.Chain().RawExecute(fmt.Sprintf("DROP TABLE IF EXISTS %s", tableName))
		assert.NoError(t, result.Error)
	}()

	// 设置表名
	model := &TestModel{}
	// 定义测试ID变量
	var testID int64

	// 创建路由处理器
	crud, err := NewCrud2(
		"test",
		model,
		db,
		[]string{"id", "name"},
		[]ConditionParam{{QueryName: "id", Operation: define.OpEq}, {QueryName: "name", Operation: define.OpEq}},
		[]string{"id", "name"},
		[]ConditionParam{{QueryName: "id", Operation: define.OpEq}},
		[]string{"name"},
		[]string{"name"},
		[]ConditionParam{{QueryName: "name", Operation: define.OpEq}},
		[]ConditionParam{{QueryName: "id", Operation: define.OpEq}},
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

		// 打印响应内容以便调试
		t.Logf("Response Body: %s", w.Body.String())

		// 获取响应内容
		responseBody := w.Body.String()
		// 解析JSON响应
		var response struct {
			Code int                    `json:"code"`
			Msg  string                 `json:"msg"`
			Data map[string]interface{} `json:"data"`
		}
		err := json.Unmarshal([]byte(responseBody), &response)
		assert.NoError(t, err, "Failed to unmarshal response: %v", err)

		// 检查响应结构
		assert.Equal(t, 200, response.Code, "Response code should be 200")
		assert.NotEmpty(t, response.Data, "Response data should not be empty")

		// 验证数据
		assert.NotZero(t, response.Data["id"], "ID should not be zero")

		// 保存ID用于后续测试
		testID = int64(response.Data["id"].(float64))
	})

	// 测试查询列表
	t.Run("List", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/api/test/list?pageSize=10&pageNum=1", nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response struct {
			Code int    `json:"code"`
			Msg  string `json:"msg"`
			Data struct {
				List       interface{} `json:"list"`
				Total      int64       `json:"total"`
				PageSize   int         `json:"pageSize"`
				PageNumber int         `json:"pageNum"`
			} `json:"data"`
		}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, 200, response.Code)
		assert.NotNil(t, response.Data.List)
		assert.Equal(t, 10, response.Data.PageSize)
		assert.Equal(t, 1, response.Data.PageNumber)
	})

	// 测试查询单个
	t.Run("Detail", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", fmt.Sprintf("/api/test/detail?idEq=%d", 4), nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response struct {
			Code int                    `json:"code"`
			Msg  string                 `json:"msg"`
			Data map[string]interface{} `json:"data"`
		}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, 200, response.Code)
		assert.NotNil(t, response.Data)
		assert.Positive(t, response.Data["id"])
		assert.Equal(t, "test1", response.Data["name"])
	})

	// 测试更新
	t.Run("Update", func(t *testing.T) {
		w := httptest.NewRecorder()
		newName := "updated" + time.Now().String()
		body := fmt.Sprintf(`{"id":%d,"name":"%s"}`, 4, newName)
		req := httptest.NewRequest("POST", "/api/test/update", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response struct {
			Code int                    `json:"code"`
			Msg  string                 `json:"msg"`
			Data map[string]interface{} `json:"data"`
		}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, 200, response.Code)
		assert.NotNil(t, response.Data)
	})

	// 测试删除
	t.Run("Delete", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", fmt.Sprintf("/api/test/delete?id=%d", testID), nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var deleteResponse struct {
			Code int    `json:"code"`
			Msg  string `json:"msg"`
		}
		err := json.Unmarshal(w.Body.Bytes(), &deleteResponse)
		assert.NoError(t, err)
		assert.Equal(t, 200, deleteResponse.Code)

		// 验证记录已被删除
		w = httptest.NewRecorder()
		req = httptest.NewRequest("GET", fmt.Sprintf("/api/test/detail?id=%d", testID), nil)
		r.ServeHTTP(w, req)

		var response struct {
			Code int         `json:"code"`
			Msg  string      `json:"msg"`
			Data interface{} `json:"data"`
		}
		err = json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, 200, response.Code)
		assert.Nil(t, response.Data)
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
