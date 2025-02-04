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
	"github.com/stretchr/testify/assert"
)

// ComplexTestModel 包含多种类型的测试模型
type ComplexTestModel struct {
	ID        int64     `json:"id" gom:"id,@"`
	Name      string    `json:"name" gom:"name"`
	Age       int       `json:"age" gom:"age"`
	Score     float64   `json:"score" gom:"score"`
	Active    bool      `json:"active" gom:"active"`
	Tags      []string  `json:"tags" gom:"tags"`
	CreatedAt time.Time `json:"created_at" gom:"created_at"`
	tableName string
}

func (t ComplexTestModel) TableName() string {
	if t.tableName != "" {
		return t.tableName
	}
	return "complex_test_models"
}

func (t ComplexTestModel) CreateSql() string {
	return fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
		id BIGINT AUTO_INCREMENT PRIMARY KEY,
		name VARCHAR(255),
		age INT,
		score DOUBLE,
		active BOOLEAN,
		tags JSON,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	)`, t.TableName())
}

// 修改响应结构定义
type ResponseWrapper struct {
	Code int             `json:"code"`
	Msg  string          `json:"msg"`
	Data json.RawMessage `json:"data"`
}

func TestComplexCRUDOperations(t *testing.T) {
	r, db := setupTestRouter()
	defer db.Close()

	tableName := fmt.Sprintf("complex_test_models_%d", time.Now().UnixNano())

	// 创建测试表
	createTableSQL := fmt.Sprintf(`
		CREATE TABLE %s (
			id BIGINT AUTO_INCREMENT PRIMARY KEY,
			name VARCHAR(255),
			age INT,
			score DOUBLE,
			active BOOLEAN,
			tags JSON,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`, tableName)

	result := db.Chain().RawExecute(fmt.Sprintf("DROP TABLE IF EXISTS %s", tableName))
	assert.NoError(t, result.Error)

	result = db.Chain().RawExecute(createTableSQL)
	assert.NoError(t, result.Error)

	defer func() {
		result := db.Chain().RawExecute(fmt.Sprintf("DROP TABLE IF EXISTS %s", tableName))
		assert.NoError(t, result.Error)
	}()

	model := &ComplexTestModel{tableName: tableName}
	crud, err := NewCrud(db, tableName, model)
	assert.NoError(t, err)
	assert.NotNil(t, crud)

	err = crud.Register(r.Group("/api"))
	assert.NoError(t, err)

	var testID int64

	// 测试创建
	t.Run("Create", func(t *testing.T) {
		now := time.Now()
		body := fmt.Sprintf(`{
			"name": "test1",
			"age": 25,
			"score": 95.5,
			"active": true,
			"tags": ["tag1", "tag2"],
			"created_at": "%s"
		}`, now.Format(time.RFC3339))

		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/api/"+tableName+"/add", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response ResponseWrapper
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, 0, response.Code)

		var data ComplexTestModel
		err = json.Unmarshal(response.Data, &data)
		assert.NoError(t, err)

		assert.NotZero(t, data.ID)
		assert.Equal(t, "test1", data.Name)
		assert.Equal(t, 25, data.Age)
		assert.Equal(t, 95.5, data.Score)
		assert.True(t, data.Active)
		assert.Equal(t, []string{"tag1", "tag2"}, data.Tags)

		testID = data.ID
	})

	// 测试复杂查询
	t.Run("ComplexQuery", func(t *testing.T) {
		// 测试多条件查询
		queries := []struct {
			name     string
			url      string
			expected int
		}{
			{
				name:     "基本等于查询",
				url:      fmt.Sprintf("/api/%s/list?nameEq=test1", tableName),
				expected: 1,
			},
			{
				name:     "数值范围查询",
				url:      fmt.Sprintf("/api/%s/list?ageGe=20&ageLe=30&scoreLt=100", tableName),
				expected: 1,
			},
			{
				name:     "布尔值查询",
				url:      fmt.Sprintf("/api/%s/list?activeEq=true", tableName),
				expected: 1,
			},
			{
				name:     "模糊查询",
				url:      fmt.Sprintf("/api/%s/list?nameLike=test", tableName),
				expected: 1,
			},
			{
				name:     "组合查询",
				url:      fmt.Sprintf("/api/%s/list?nameEq=test1&ageGe=20&ageLe=30&scoreLt=100&activeEq=true", tableName),
				expected: 1,
			},
		}

		for _, tc := range queries {
			t.Run(tc.name, func(t *testing.T) {
				w := httptest.NewRecorder()
				req := httptest.NewRequest("GET", tc.url, nil)
				r.ServeHTTP(w, req)

				assert.Equal(t, http.StatusOK, w.Code)

				var response ResponseWrapper
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Equal(t, 0, response.Code)

				var pageInfo PageInfo
				err = json.Unmarshal(response.Data, &pageInfo)
				assert.NoError(t, err)

				var list []ComplexTestModel
				listData, err := json.Marshal(pageInfo.Data)
				assert.NoError(t, err)
				err = json.Unmarshal(listData, &list)
				assert.NoError(t, err)

				assert.Len(t, list, tc.expected)
			})
		}
	})

	// 测试更新
	t.Run("Update", func(t *testing.T) {
		updates := []struct {
			name     string
			body     string
			validate func(t *testing.T, model ComplexTestModel)
		}{
			{
				name: "更新基本字段",
				body: fmt.Sprintf(`{
					"id": %d,
					"name": "updated",
					"age": 30
				}`, testID),
				validate: func(t *testing.T, model ComplexTestModel) {
					assert.Equal(t, "updated", model.Name)
					assert.Equal(t, 30, model.Age)
				},
			},
			{
				name: "更新数值字段",
				body: fmt.Sprintf(`{
					"id": %d,
					"score": 98.5
				}`, testID),
				validate: func(t *testing.T, model ComplexTestModel) {
					assert.Equal(t, 98.5, model.Score)
				},
			},
			{
				name: "更新布尔字段",
				body: fmt.Sprintf(`{
					"id": %d,
					"active": false
				}`, testID),
				validate: func(t *testing.T, model ComplexTestModel) {
					assert.False(t, model.Active)
				},
			},
			{
				name: "更新数组字段",
				body: fmt.Sprintf(`{
					"id": %d,
					"tags": ["tag3", "tag4"]
				}`, testID),
				validate: func(t *testing.T, model ComplexTestModel) {
					assert.Equal(t, []string{"tag3", "tag4"}, model.Tags)
				},
			},
		}

		for _, tc := range updates {
			t.Run(tc.name, func(t *testing.T) {
				w := httptest.NewRecorder()
				req := httptest.NewRequest("POST", "/api/"+tableName+"/update", strings.NewReader(tc.body))
				req.Header.Set("Content-Type", "application/json")
				r.ServeHTTP(w, req)

				assert.Equal(t, http.StatusOK, w.Code)

				var response ResponseWrapper
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Equal(t, 0, response.Code)

				var data ComplexTestModel
				err = json.Unmarshal(response.Data, &data)
				assert.NoError(t, err)

				tc.validate(t, data)
			})
		}
	})

	// 测试删除
	t.Run("Delete", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", fmt.Sprintf("/api/%s/delete?id=%d", tableName, testID), nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response ResponseWrapper
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, 0, response.Code)

		// 验证删除后无法查询
		w = httptest.NewRecorder()
		req = httptest.NewRequest("GET", fmt.Sprintf("/api/%s/detail?id=%d", tableName, testID), nil)
		r.ServeHTTP(w, req)

		err = json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, 0, response.Code)
		assert.Equal(t, "null", string(response.Data))
	})
}

// 测试错误处理
func TestErrorHandling(t *testing.T) {
	_, db := setupTestRouter()
	defer db.Close()

	t.Run("InvalidTableName", func(t *testing.T) {
		model := &ComplexTestModel{}
		_, err := NewCrud(db, "", model)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "table name cannot be empty")
	})

	t.Run("NilModel", func(t *testing.T) {
		_, err := NewCrud(db, "test", nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "model cannot be nil")
	})

	t.Run("InvalidModel", func(t *testing.T) {
		var invalidModel string
		_, err := NewCrud(db, "test", &invalidModel)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "model must be a struct")
	})

	t.Run("DuplicateHandler", func(t *testing.T) {
		crud, err := NewCrud(db, "test", &ComplexTestModel{})
		assert.NoError(t, err)

		err = crud.AddHandler(RouteHandler{
			Path:       "list",
			HttpMethod: "GET",
			Handlers:   []gin.HandlerFunc{func(c *gin.Context) {}},
		})
		assert.NoError(t, err)

		err = crud.AddHandler(RouteHandler{
			Path:       "list",
			HttpMethod: "GET",
			Handlers:   []gin.HandlerFunc{func(c *gin.Context) {}},
		})
		assert.NoError(t, err) // 应该允许覆盖
	})

	t.Run("DeleteNonExistentHandler", func(t *testing.T) {
		crud, err := NewCrud(db, "test", &ComplexTestModel{})
		assert.NoError(t, err)

		err = crud.DeleteHandler("nonexistent")
		assert.Error(t, err)
	})
}

// 测试性能
func BenchmarkCRUDOperations(b *testing.B) {
	r, db := setupTestRouter()
	defer db.Close()

	tableName := fmt.Sprintf("bench_test_models_%d", time.Now().UnixNano())

	createTableSQL := fmt.Sprintf(`
		CREATE TABLE %s (
			id BIGINT AUTO_INCREMENT PRIMARY KEY,
			name VARCHAR(255),
			age INT,
			score DOUBLE,
			active BOOLEAN,
			tags JSON,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`, tableName)

	result := db.Chain().RawExecute(fmt.Sprintf("DROP TABLE IF EXISTS %s", tableName))
	if result.Error != nil {
		b.Fatal(result.Error)
	}

	result = db.Chain().RawExecute(createTableSQL)
	if result.Error != nil {
		b.Fatal(result.Error)
	}

	defer func() {
		result := db.Chain().RawExecute(fmt.Sprintf("DROP TABLE IF EXISTS %s", tableName))
		if result.Error != nil {
			b.Fatal(result.Error)
		}
	}()

	model := &ComplexTestModel{}
	crud, err := NewCrud(db, tableName, model)
	if err != nil {
		b.Fatal(err)
	}

	err = crud.Register(r.Group("/api"))
	if err != nil {
		b.Fatal(err)
	}

	b.Run("Create", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			body := fmt.Sprintf(`{
				"name": "bench%d",
				"age": %d,
				"score": %f,
				"active": true,
				"tags": ["tag1", "tag2"]
			}`, i, 20+i%30, 90.0+float64(i%10))

			w := httptest.NewRecorder()
			req := httptest.NewRequest("POST", "/api/"+tableName+"/add", strings.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			r.ServeHTTP(w, req)
		}
	})

	b.Run("Query", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			w := httptest.NewRecorder()
			req := httptest.NewRequest("GET", "/api/"+tableName+"/list?pageSize=10&pageNum=1", nil)
			r.ServeHTTP(w, req)
		}
	})
}
