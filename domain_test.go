package crud

import (
	"bytes"
	"encoding/json"
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

type Domain struct {
	ID           uint      `json:"id" gom:"id,@,auto"`
	Name         string    `json:"name" gom:"name,notnull,unique"`
	DomainName   string    `json:"domainName" gom:"identifier,notnull,unique"`
	Description  string    `json:"description" gom:"description"`
	ServiceCount int       `json:"serviceCount" gom:"service_count,default"`
	Status       int       `json:"status" gom:"status,notnull,default"`
	CreatedAt    time.Time `json:"createdAt" gom:"created_at,notnull,default"`
	UpdatedAt    time.Time `json:"updatedAt" gom:"updated_at,notnull,default"`
}

func (d *Domain) TableName() string {
	return "domains"
}

func initTestDB() *gom.DB {
	opts := &define.DBOptions{
		MaxOpenConns: 10,
		MaxIdleConns: 5,
		Debug:        true,
	}
	// 使用 MySQL 测试数据库
	db, err := gom.Open("mysql", "root:123456@tcp(192.168.110.249:3306)/test?parseTime=true", opts)
	if err != nil {
		panic(err)
	}

	// 先删除已存在的表
	db.Chain().RawExecute("DROP TABLE IF EXISTS domain_services")
	db.Chain().RawExecute("DROP TABLE IF EXISTS services")
	db.Chain().RawExecute("DROP TABLE IF EXISTS domains")

	// Create tables using raw SQL
	result := db.Chain().RawExecute(`
	CREATE TABLE IF NOT EXISTS domains (
		id BIGINT PRIMARY KEY AUTO_INCREMENT,
		name VARCHAR(255) NOT NULL UNIQUE,
		identifier VARCHAR(255) NOT NULL UNIQUE,
		description TEXT,
		service_count INT DEFAULT 0,
		status INT NOT NULL DEFAULT 1,
		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
	)`)
	if result.Error != nil {
		panic(result.Error)
	}

	result = db.Chain().RawExecute(`
	CREATE TABLE IF NOT EXISTS services (
		id BIGINT PRIMARY KEY AUTO_INCREMENT,
		name VARCHAR(255) NOT NULL,
		description TEXT,
		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
	)`)
	if result.Error != nil {
		panic(result.Error)
	}

	result = db.Chain().RawExecute(`
	CREATE TABLE IF NOT EXISTS domain_services (
		domain_id BIGINT,
		service_id BIGINT,
		PRIMARY KEY (domain_id, service_id),
		FOREIGN KEY (domain_id) REFERENCES domains(id) ON DELETE CASCADE,
		FOREIGN KEY (service_id) REFERENCES services(id) ON DELETE CASCADE
	)`)
	if result.Error != nil {
		panic(result.Error)
	}

	return db
}

func cleanupTestDB(db *gom.DB) {
	if db != nil {
		db.Close()
	}
}

// CRUD represents the CRUD operations handler
type CRUD struct {
	db *gom.DB
}

// NewCRUD creates a new CRUD instance
func NewCRUD(db *gom.DB) *CRUD {
	return &CRUD{db: db}
}

func mapToDomain(m map[string]any) Domain {
	item := Domain{
		Name:         m["name"].(string),
		DomainName:   m["domainName"].(string),
		Description:  "",
		ServiceCount: 0,
		Status:       1,
	}

	if desc, ok := m["description"].(string); ok {
		item.Description = desc
	}

	if count, ok := m["serviceCount"].(float64); ok {
		item.ServiceCount = int(count)
	}

	if status, ok := m["status"].(float64); ok {
		item.Status = int(status)
	}

	if ct, ok := m["createdAt"].(time.Time); ok {
		item.CreatedAt = ct
	}
	if ut, ok := m["updatedAt"].(time.Time); ok {
		item.UpdatedAt = ut
	}
	return item
}

// Register registers the CRUD routes for a model
func (c *CRUD) Register(rg *gin.RouterGroup, model interface{}) {
	crud := New2(c.db, model)
	crud.RegisterRoutes(rg, "")
}

func TestDomainCRUD(t *testing.T) {
	// Setup
	db := initTestDB()
	defer cleanupTestDB(db)

	gin.SetMode(gin.TestMode)
	router := gin.Default()
	crud := New2(db, &Domain{})
	crud.SetDescription("域名管理模块")
	RegisterCrud(crud)

	api := router.Group("/api/domains")
	crud.RegisterRoutes(api, "")

	t.Run("Create Domain", func(t *testing.T) {
		payload := map[string]interface{}{
			"name":         "Test Domain",
			"domainName":   "test-domain",
			"description":  "Test Description",
			"serviceCount": 0,
			"status":       1,
		}
		jsonData, err := json.Marshal(payload)
		assert.NoError(t, err)

		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/api/domains/save", bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)

		data := response["data"].(map[string]interface{})
		assert.NotZero(t, data["id"])
		assert.Equal(t, payload["name"], data["name"])
		assert.Equal(t, payload["domainName"], data["domainName"])
		assert.Equal(t, payload["description"], data["description"])
		assert.Equal(t, float64(1), data["status"])
	})

	t.Run("Get Domain List", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/api/domains/list", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)

		data := response["data"].([]interface{})
		assert.NotEmpty(t, data)
		firstItem := data[0].(map[string]interface{})
		assert.Equal(t, "Test Domain", firstItem["name"])
	})

	t.Run("Get Domain Detail", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/api/domains/detail/1", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)

		data := response["data"].(map[string]interface{})
		assert.NotZero(t, data["id"])
		assert.Equal(t, "Test Domain", data["name"])
		assert.Equal(t, "test-domain", data["identifier"])
	})

	t.Run("Update Domain", func(t *testing.T) {
		payload := map[string]interface{}{
			"id":           1,
			"name":         "Updated Domain",
			"domainName":   "updated-domain",
			"description":  "Updated Description",
			"serviceCount": 0,
			"status":       2,
			"services":     []interface{}{},
		}
		jsonData, err := json.Marshal(payload)
		assert.NoError(t, err)

		w := httptest.NewRecorder()
		req := httptest.NewRequest("PUT", "/api/domains/update", bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		// Verify the update
		w = httptest.NewRecorder()
		req = httptest.NewRequest("GET", "/api/domains/detail/1", nil)
		router.ServeHTTP(w, req)

		var response map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)

		data := response["data"].(map[string]interface{})
		assert.Equal(t, "Updated Domain", data["name"])
		assert.Equal(t, "updated-domain", data["identifier"])
		assert.Equal(t, "Updated Description", data["description"])
		assert.Equal(t, float64(2), data["status"])
	})

	t.Run("Delete Domain", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("DELETE", "/api/domains/delete/1", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		// Verify the deletion
		w = httptest.NewRecorder()
		req = httptest.NewRequest("GET", "/api/domains/detail/1", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("Create Domain with Invalid Data", func(t *testing.T) {
		// First, create a domain
		payload := map[string]interface{}{
			"name":         "Test Domain",
			"identifier":   "test-domain",
			"description":  "Test Description",
			"serviceCount": 0,
			"status":       1,
			"services":     []interface{}{},
		}
		jsonData, err := json.Marshal(payload)
		assert.NoError(t, err)

		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/api/domains/save", bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		// Then try to create another domain with the same name and identifier
		w = httptest.NewRecorder()
		req = httptest.NewRequest("POST", "/api/domains/save", bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		// Should fail due to unique constraint
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}
