package crud

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
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

func (u *TestUser) TableName() string {
	return "test_users"
}

func init() {
	Debug = true
}

// setupTestDB 设置测试数据库
func setupTestDB() *gom.DB {
	debugf("Setting up test database")
	db, err := gom.Open("mysql", "remote:123456@tcp(192.168.110.249:3306)/test?charset=utf8mb4&parseTime=true", &define.DBOptions{
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
	gin.SetMode(gin.TestMode)
	r := gin.Default()

	crud := New2(db, &TestUser{})
	crud.SetDescription("测试用户管理模块")
	RegisterCrud(crud)

	api := r.Group("/api/users")
	crud.RegisterRoutes(api, "")

	RegisterApi(r, "/api-info")
	RegisterApiDoc(r, "/api-doc")

	debugf("Test router setup completed")
	return r, crud
}

// insertTestData 插入测试数据
func insertTestData(t *testing.T, db *gom.DB) {
	testUsers := []map[string]interface{}{
		{"username": "user1", "email": "user1@test.com", "age": 25, "status": "active"},
		{"username": "user2", "email": "user2@test.com", "age": 30, "status": "active"},
		{"username": "user3", "email": "user3@test.com", "age": 35, "status": "inactive"},
		{"username": "user4", "email": "user4@test.com", "age": 40, "status": "active"},
		{"username": "user5", "email": "user5@test.com", "age": 45, "status": "inactive"},
	}

	for _, user := range testUsers {
		result := db.Chain().Table("test_users").Values(user).Save()
		assert.NoError(t, result.Error)
	}
}

func TestCRUDOperations(t *testing.T) {
	db := setupTestDB()
	r, _ := setupTestRouter(db)

	t.Run("Create", func(t *testing.T) {
		payload := map[string]interface{}{
			"username": "testuser",
			"email":    "test@example.com",
			"age":      30,
			"status":   "active",
		}
		jsonData, _ := json.Marshal(payload)

		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/api/users/save", bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, float64(0), response["code"])

		data := response["data"].(map[string]interface{})
		assert.Equal(t, "testuser", data["username"])
		assert.Equal(t, "test@example.com", data["email"])
		assert.Equal(t, float64(30), data["age"])
		assert.Equal(t, "active", data["status"])
	})

	t.Run("Read", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/api/users/detail/1", nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, float64(0), response["code"])

		data := response["data"].(map[string]interface{})
		assert.Equal(t, "testuser", data["username"])
		assert.Equal(t, "test@example.com", data["email"])
	})

	t.Run("Update", func(t *testing.T) {
		payload := map[string]interface{}{
			"status": "inactive",
		}
		jsonData, _ := json.Marshal(payload)

		w := httptest.NewRecorder()
		req := httptest.NewRequest("PUT", "/api/users/update/1", bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, float64(0), response["code"])

		data := response["data"].(map[string]interface{})
		assert.Equal(t, "inactive", data["status"])
	})

	t.Run("Delete", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("DELETE", "/api/users/delete/1", nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		// 验证删除后无法查询到记录
		w = httptest.NewRecorder()
		req = httptest.NewRequest("GET", "/api/users/detail/1", nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestPagination(t *testing.T) {
	db := setupTestDB()
	r, _ := setupTestRouter(db)
	insertTestData(t, db)

	tests := []struct {
		name           string
		url            string
		expectedPage   int
		expectedSize   int
		expectedTotal  int
		expectedStatus int
	}{
		{
			name:           "Default Pagination",
			url:            "/api/users/page",
			expectedPage:   1,
			expectedSize:   10,
			expectedTotal:  5,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Custom Pagination",
			url:            "/api/users/page?page=2&size=2",
			expectedPage:   2,
			expectedSize:   2,
			expectedTotal:  5,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Invalid Page",
			url:            "/api/users/page?page=0",
			expectedPage:   1,
			expectedSize:   10,
			expectedTotal:  5,
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			req := httptest.NewRequest("GET", tt.url, nil)
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)

			data := response["data"].(map[string]interface{})
			assert.Equal(t, float64(tt.expectedPage), data["page"])
			assert.Equal(t, float64(tt.expectedSize), data["size"])
			assert.Equal(t, float64(tt.expectedTotal), data["total"])
		})
	}
}

func TestQueryConditions(t *testing.T) {
	db := setupTestDB()
	r, _ := setupTestRouter(db)
	insertTestData(t, db)

	tests := []struct {
		name           string
		url            string
		expectedCount  int
		expectedStatus int
	}{
		{
			name:           "Filter by Status",
			url:            "/api/users/page?status=active",
			expectedCount:  3,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Filter by Age Range",
			url:            "/api/users/page?age_gte=30&age_lte=40",
			expectedCount:  3,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Filter by Email Like",
			url:            "/api/users/page?email_like=test",
			expectedCount:  5,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Multiple Filters",
			url:            "/api/users/page?status=active&age_gt=30",
			expectedCount:  1,
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			req := httptest.NewRequest("GET", tt.url, nil)
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)

			data := response["data"].(map[string]interface{})
			assert.Equal(t, float64(tt.expectedCount), data["total"])
		})
	}
}

func TestSorting(t *testing.T) {
	db := setupTestDB()
	r, _ := setupTestRouter(db)
	insertTestData(t, db)

	tests := []struct {
		name           string
		url            string
		expectedOrder  []int
		expectedStatus int
	}{
		{
			name:           "Sort by Age Ascending",
			url:            "/api/users/page?sort=age",
			expectedOrder:  []int{25, 30, 35, 40, 45},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Sort by Age Descending",
			url:            "/api/users/page?sort=-age",
			expectedOrder:  []int{45, 40, 35, 30, 25},
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			req := httptest.NewRequest("GET", tt.url, nil)
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)

			data := response["data"].(map[string]interface{})
			list := data["list"].([]interface{})

			ages := make([]int, len(list))
			for i, item := range list {
				user := item.(map[string]interface{})
				ages[i] = int(user["age"].(float64))
			}

			assert.Equal(t, tt.expectedOrder, ages)
		})
	}
}

func TestErrorHandling(t *testing.T) {
	db := setupTestDB()
	r, _ := setupTestRouter(db)

	tests := []struct {
		name           string
		method         string
		url            string
		payload        interface{}
		expectedStatus int
		expectedCode   float64
	}{
		{
			name:           "Invalid JSON",
			method:         "POST",
			url:            "/api/users/save",
			payload:        "invalid json",
			expectedStatus: http.StatusBadRequest,
			expectedCode:   float64(CodeInvalid),
		},
		{
			name:           "Record Not Found",
			method:         "GET",
			url:            "/api/users/detail/999",
			expectedStatus: http.StatusNotFound,
			expectedCode:   float64(CodeNotFound),
		},
		{
			name:   "Invalid Update Data",
			method: "PUT",
			url:    "/api/users/update/1",
			payload: map[string]interface{}{
				"age": "invalid",
			},
			expectedStatus: http.StatusBadRequest,
			expectedCode:   float64(CodeInvalid),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req *http.Request
			if tt.payload != nil {
				jsonData, _ := json.Marshal(tt.payload)
				req = httptest.NewRequest(tt.method, tt.url, bytes.NewBuffer(jsonData))
				req.Header.Set("Content-Type", "application/json")
			} else {
				req = httptest.NewRequest(tt.method, tt.url, nil)
			}

			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedCode, response["code"])
		})
	}
}

func TestFieldFiltering(t *testing.T) {
	db := setupTestDB()
	r, crud := setupTestRouter(db)
	insertTestData(t, db)

	if handler, ok := crud.GetHandler(PAGE); ok {
		handler.AllowedFields = []string{"username", "email"}
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/users/page?fields=username,email", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	data := response["data"].(map[string]interface{})
	list := data["list"].([]interface{})
	for _, item := range list {
		user := item.(map[string]interface{})
		assert.Contains(t, user, "username")
		assert.Contains(t, user, "email")
		assert.NotContains(t, user, "age")
		assert.NotContains(t, user, "status")
	}
}
