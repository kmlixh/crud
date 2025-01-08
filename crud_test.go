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
	_ "github.com/kmlixh/gom/v4/factory/mysql"
	"github.com/stretchr/testify/assert"
)

// TestUser 测试用户结构体
type TestUser struct {
	ID        int64     `json:"id" gom:"id,primary_key,auto_increment"`
	Username  string    `json:"username" gom:"username"`
	Email     string    `json:"email" gom:"email"`
	Age       int       `json:"age" gom:"age"`
	Status    string    `json:"status" gom:"status"`
	CreatedAt time.Time `json:"created_at" gom:"created_at"`
}

// TableName 返回表名
func (u *TestUser) TableName() string {
	return "test_users"
}

// GetTableName 返回表名
func (u *TestUser) GetTableName() string {
	return "test_users"
}

// GetModel 返回模型实例
func (u *TestUser) GetModel() interface{} {
	return &TestUser{}
}

func init() {
	Debug = true
}

// setupTestDB 设置测试数据库
func setupTestDB() *gom.DB {
	debugf("Setting up test database")
	// 连接数据库
	db, err := gom.Open("mysql", "root:123456@tcp(192.168.110.249:3306)/test?charset=utf8mb4&parseTime=True&loc=Local", &define.DBOptions{
		Debug: false,
	})
	if err != nil {
		panic(fmt.Sprintf("Failed to connect to database: %v", err))
	}

	// 创建测试表
	result := db.Chain().RawExecute(`DROP TABLE IF EXISTS test_users`)
	if result.Error != nil {
		panic(fmt.Sprintf("Failed to drop table: %v", result.Error))
	}

	result = db.Chain().RawExecute(`
		CREATE TABLE test_users (
			id BIGINT PRIMARY KEY AUTO_INCREMENT,
			username VARCHAR(255),
			email VARCHAR(255),
			age INT,
			status VARCHAR(50),
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if result.Error != nil {
		panic(fmt.Sprintf("Failed to create table: %v", result.Error))
	}

	debugf("Test database setup completed")
	return db
}

// setupTestRouter 设置测试路由
func setupTestRouter(db *gom.DB) (*gin.Engine, *Crud) {
	debugf("Setting up test router")
	gin.SetMode(gin.DebugMode)
	r := gin.Default()

	// 创建 CRUD 实例
	crud := New2(db, &TestUser{})
	debugf("Table name: %s", crud.tableName)
	crud.SetDescription("测试用户管理模块")
	RegisterCrud(crud) // 注册 CRUD 实例

	// 设置处理器
	if listHandler, ok := crud.GetHandler(LIST); ok {
		listHandler.AllowedFields = []string{"id", "username", "email", "age", "status"}
		listHandler.SetDescription("获取用户列表")
		crud.AddHandler(LIST, listHandler.Method, listHandler)
		debugf("List handler registered: %s %s", listHandler.Method, listHandler.Path)
	}

	// 注册路由
	api := r.Group("/api/users")
	crud.RegisterRoutes(api, "")

	// 注册 API 文档
	RegisterApi(r, "/api-info")
	RegisterApiDoc(r, "/api-doc")

	debugf("Test router setup completed")
	return r, crud
}

// insertTestData 插入测试数据
func insertTestData(t *testing.T, db *gom.DB) {
	testUsers := []TestUser{
		{Username: "user1", Email: "user1@test.com", Age: 25, Status: "active"},
		{Username: "user2", Email: "user2@test.com", Age: 30, Status: "active"},
		{Username: "user3", Email: "user3@test.com", Age: 35, Status: "inactive"},
		{Username: "user4", Email: "user4@test.com", Age: 40, Status: "active"},
		{Username: "user5", Email: "user5@test.com", Age: 45, Status: "inactive"},
	}

	for _, user := range testUsers {
		result := db.Chain().Table("test_users").Values(map[string]interface{}{
			"username": user.Username,
			"email":    user.Email,
			"age":      user.Age,
			"status":   user.Status,
		}).Save()
		if result.Error != nil {
			t.Logf("Error inserting test data: %v", result.Error)
		} else {
			t.Logf("Inserted test data: %v", result)
		}
	}
}

// TestCRUDOperations 测试 CRUD 操作
func TestCRUDOperations(t *testing.T) {
	db := setupTestDB()
	r, _ := setupTestRouter(db)

	t.Run("Create", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, err := http.NewRequest("POST", "/api/users/save", strings.NewReader(`{
			"username": "testuser",
			"email": "test@example.com",
			"age": 30,
			"status": "active"
		}`))
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		t.Logf("Create Response: %s", w.Body.String())
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), `"username":"testuser"`)
		assert.Contains(t, w.Body.String(), `"email":"test@example.com"`)
		assert.Contains(t, w.Body.String(), `"status":"active"`)
	})

	t.Run("Read", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, err := http.NewRequest("GET", "/api/users/detail/1", nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		r.ServeHTTP(w, req)

		t.Logf("Read Response: %s", w.Body.String())
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), `"username":"testuser"`)
		assert.Contains(t, w.Body.String(), `"email":"test@example.com"`)
		assert.Contains(t, w.Body.String(), `"status":"active"`)
	})

	t.Run("Update", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, err := http.NewRequest("PUT", "/api/users/update/1", strings.NewReader(`{
			"status": "inactive"
		}`))
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		t.Logf("Update Response: %s", w.Body.String())
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), `"status":"inactive"`)
	})

	t.Run("Delete", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, err := http.NewRequest("DELETE", "/api/users/delete/1", nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		r.ServeHTTP(w, req)

		t.Logf("Delete Response: %s", w.Body.String())
		assert.Equal(t, http.StatusOK, w.Code)

		// 验证删除成功
		w = httptest.NewRecorder()
		req, err = http.NewRequest("GET", "/api/users/detail/1", nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		r.ServeHTTP(w, req)

		t.Logf("Verify Delete Response: %s", w.Body.String())
		assert.Equal(t, http.StatusNotFound, w.Code)
		var response map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, float64(CodeNotFound), response["code"])
	})
}

// TestPagination 测试分页功能
func TestPagination(t *testing.T) {
	db := setupTestDB()
	r, _ := setupTestRouter(db)
	insertTestData(t, db)

	t.Run("Default Pagination", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, err := http.NewRequest("GET", "/api/users/page", nil)
		assert.NoError(t, err)
		r.ServeHTTP(w, req)

		t.Logf("Response: %s", w.Body.String())
		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)

		data := response["data"].(map[string]interface{})
		assert.Equal(t, float64(1), data["page"])
		assert.Equal(t, float64(10), data["size"])
	})

	t.Run("Custom Pagination", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, err := http.NewRequest("GET", "/api/users/page?page=2&size=2", nil)
		assert.NoError(t, err)
		r.ServeHTTP(w, req)

		t.Logf("Response: %s", w.Body.String())
		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)

		data := response["data"].(map[string]interface{})
		assert.Equal(t, float64(2), data["page"])
		assert.Equal(t, float64(2), data["size"])
	})
}

// TestQueryConditions 测试查询条件
func TestQueryConditions(t *testing.T) {
	db := setupTestDB()
	r, _ := setupTestRouter(db)

	insertTestData(t, db)

	t.Run("Filter by Status", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, err := http.NewRequest("GET", "/api/users/page?status=active", nil)
		assert.NoError(t, err)
		r.ServeHTTP(w, req)

		t.Logf("Response: %s", w.Body.String())
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), `"status":"active"`)
		assert.NotContains(t, w.Body.String(), `"status":"inactive"`)
	})

	t.Run("Filter by Age Range", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, err := http.NewRequest("GET", "/api/users/page?age_gte=30&age_lte=40", nil)
		assert.NoError(t, err)
		r.ServeHTTP(w, req)

		t.Logf("Response: %s", w.Body.String())
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), `"age":30`)
		assert.Contains(t, w.Body.String(), `"age":35`)
		assert.Contains(t, w.Body.String(), `"age":40`)
	})
}

// TestFieldFiltering 测试字段过滤
func TestFieldFiltering(t *testing.T) {
	db := setupTestDB()
	r, crud := setupTestRouter(db)
	insertTestData(t, db)

	// 设置允许的字段
	if handler, ok := crud.GetHandler(PAGE); ok {
		handler.AllowedFields = []string{"username", "email"}
	}

	t.Run("Select Specific Fields", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, err := http.NewRequest("GET", "/api/users/page?fields=username,email", nil)
		assert.NoError(t, err)
		r.ServeHTTP(w, req)

		t.Logf("Response: %s", w.Body.String())
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), `"username":"user1"`)
		assert.Contains(t, w.Body.String(), `"email":"user1@test.com"`)
		assert.NotContains(t, w.Body.String(), `"age"`)
		assert.NotContains(t, w.Body.String(), `"status"`)
	})
}

// TestSorting 测试排序功能
func TestSorting(t *testing.T) {
	db := setupTestDB()
	r, _ := setupTestRouter(db)
	insertTestData(t, db) // 先插入测试数据

	t.Run("Sort by Age Ascending", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, err := http.NewRequest("GET", "/api/users/page?sort=age", nil)
		assert.NoError(t, err)
		r.ServeHTTP(w, req)

		t.Logf("Response: %s", w.Body.String())
		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)

		data := response["data"].(map[string]interface{})
		list := data["list"].([]interface{})

		// 验证年龄升序
		var lastAge float64 = 0
		for _, item := range list {
			user := item.(map[string]interface{})
			age := user["age"].(float64)
			assert.True(t, age >= lastAge, "Ages should be in ascending order")
			lastAge = age
		}
	})

	t.Run("Sort by Age Descending", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, err := http.NewRequest("GET", "/api/users/page?sort=-age", nil)
		assert.NoError(t, err)
		r.ServeHTTP(w, req)

		t.Logf("Response: %s", w.Body.String())
		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)

		data := response["data"].(map[string]interface{})
		list := data["list"].([]interface{})

		// 验证年龄降序
		var lastAge float64 = 999
		for _, item := range list {
			user := item.(map[string]interface{})
			age := user["age"].(float64)
			assert.True(t, age <= lastAge, "Ages should be in descending order")
			lastAge = age
		}
	})
}

// TestApiDocumentation 测试 API 文档生成
func TestApiDocumentation(t *testing.T) {
	db := setupTestDB()
	r, crud := setupTestRouter(db)

	// 设置模型描述
	crud.SetDescription("测试用户管理模块")

	t.Run("JSON API Info", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, err := http.NewRequest("GET", "/api-info", nil)
		assert.NoError(t, err)
		r.ServeHTTP(w, req)

		t.Logf("Response: %s", w.Body.String())
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "TestUser")
		assert.Contains(t, w.Body.String(), "测试用户管理模块")
	})

	t.Run("HTML API Doc", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, err := http.NewRequest("GET", "/api-doc", nil)
		assert.NoError(t, err)
		r.ServeHTTP(w, req)

		t.Logf("Response: %s", w.Body.String())
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "TestUser")
		assert.Contains(t, w.Body.String(), "测试用户管理模块")
	})
}
