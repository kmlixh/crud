package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/kmlixh/crud"
	"github.com/kmlixh/gom/v4"
	_ "github.com/kmlixh/gom/v4/factory/mysql"
)

// User 用户模型
type User struct {
	ID        int       `json:"id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	Age       int       `json:"age"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

// 预定义用户状态
const (
	UserStatusActive   = "active"
	UserStatusInactive = "inactive"
	UserStatusBlocked  = "blocked"
)

func main() {
	log.Println("Starting application...")

	// 初始化数据库连接
	log.Println("Connecting to database...")
	db, err := gom.Open("mysql", "root:123456@tcp(192.168.110.249:3306)/test?charset=utf8mb4&parseTime=True&loc=Local", false)
	if err != nil {
		log.Fatal("Failed to open database:", err)
	}

	// 测试数据库连接
	log.Println("Testing database connection...")
	result := db.Chain().Table("users").List()
	if err := result.Error(); err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	log.Println("Successfully connected to database")

	// 清空用户表
	log.Println("Clearing users table...")
	if _, err := db.Chain().Table("users").Delete(); err != nil {
		log.Fatal("Failed to clear users table:", err)
	}
	log.Println("Successfully cleared users table")

	// 创建路由
	r := gin.Default()
	r.Use(crud.Cors()) // 添加跨域中间件

	// 创建用户CRUD处理器
	userCrud := crud.New2(db, &User{}).
		SetDescription("用户管理模块，提供用户的增删改查功能")

	// 修改默认处理器的字段控制
	if listHandler, ok := userCrud.GetHandler(crud.LIST); ok {
		listHandler.AllowedFields = []string{"id", "username", "email", "status", "created_at"}
		listHandler.SetDescription("获取用户列表，支持分页和排序")
		userCrud.AddHandler(crud.LIST, listHandler.Method, listHandler)
	}

	if detailHandler, ok := userCrud.GetHandler(crud.SINGLE); ok {
		detailHandler.AllowedFields = []string{"id", "username", "email", "age", "status", "created_at"}
		detailHandler.SetDescription("获取用户详细信息")
		userCrud.AddHandler(crud.SINGLE, detailHandler.Method, detailHandler)
	}

	// 添加自定义处理器 - 获取活跃用户
	activeUsersHandler := crud.NewHandler("/active", http.MethodGet).
		SetDescription("获取所有活跃状态的用户列表").
		PreProcess(func(ctx *crud.ProcessContext) error {
			return nil
		}).
		BuildQuery(func(ctx *crud.ProcessContext) error {
			ctx.Chain = db.Chain().Table("users").
				Eq("status", UserStatusActive)
			return nil
		}).
		ExecuteStep(func(ctx *crud.ProcessContext) error {
			result := ctx.Chain.List()
			if err := result.Error(); err != nil {
				return err
			}
			ctx.Data["result"] = result.Data
			return nil
		}).
		PostProcess(func(ctx *crud.ProcessContext) error {
			crud.CodeMsgFunc(ctx.GinContext, crud.CodeSuccess, "success", ctx.Data["result"])
			return nil
		})

	userCrud.AddHandler("active_users", activeUsersHandler.Method, activeUsersHandler)

	// 添加自定义处理器 - 按年龄段统计
	ageStatsHandler := crud.NewHandler("/age-stats", http.MethodGet).
		SetDescription("按年龄段统计用户数量").
		ExecuteStep(func(ctx *crud.ProcessContext) error {
			result := make(map[string]int)

			// 获取所有用户
			users := db.Chain().Table("users").List()
			if err := users.Error(); err != nil {
				return err
			}

			// 手动统计年龄段
			data := users.Data
			for _, item := range data {
				age, ok := item["age"].(int64)
				if !ok {
					continue
				}

				var group string
				switch {
				case age < 18:
					group = "under_18"
				case age >= 18 && age <= 30:
					group = "18_30"
				case age >= 31 && age <= 50:
					group = "31_50"
				default:
					group = "over_50"
				}
				result[group]++
			}

			ctx.Data["result"] = result
			return nil
		}).
		PostProcess(func(ctx *crud.ProcessContext) error {
			crud.JsonOk(ctx.GinContext, ctx.Data["result"])
			return nil
		})

	userCrud.AddHandler("age_stats", ageStatsHandler.Method, ageStatsHandler)

	// 注册用户相关路由
	userCrud.RegisterRoutes(r.Group("/api"), "/users")

	// 注册API信息路由
	crud.RegisterApi(r, "/api-info")

	// 启动HTTP服务器
	go func() {
		if err := r.Run(":8080"); err != nil {
			log.Fatal(err)
		}
	}()

	// 等待服务器启动
	time.Sleep(time.Second)

	// 测试数据
	testData := []User{
		{Username: "alice", Email: "alice@example.com", Age: 25, Status: UserStatusActive},
		{Username: "bob", Email: "bob@example.com", Age: 30, Status: UserStatusActive},
		{Username: "charlie", Email: "charlie@example.com", Age: 35, Status: UserStatusInactive},
		{Username: "david", Email: "david@example.com", Age: 40, Status: UserStatusBlocked},
		{Username: "eve", Email: "eve@example.com", Age: 22, Status: UserStatusActive},
		{Username: "frank", Email: "frank@example.com", Age: 28, Status: UserStatusActive},
		{Username: "grace", Email: "grace@example.com", Age: 45, Status: UserStatusInactive},
		{Username: "henry", Email: "henry@example.com", Age: 17, Status: UserStatusActive},
		{Username: "ivy", Email: "ivy@example.com", Age: 55, Status: UserStatusActive},
		{Username: "jack", Email: "jack@example.com", Age: 33, Status: UserStatusBlocked},
	}

	// 插入测试数据
	for _, user := range testData {
		values := map[string]interface{}{
			"username":   user.Username,
			"email":      user.Email,
			"age":        user.Age,
			"status":     user.Status,
			"created_at": time.Now(),
		}
		_, err := db.Chain().Table("users").Values(values).Save()
		if err != nil {
			log.Printf("Failed to insert user %s: %v", user.Username, err)
		}
	}

	// 等待服务器完全启动
	time.Sleep(time.Second * 2)

	// 运行接口测试
	fmt.Println("\n=== 开始接口测试 ===")
	runAPITests()
}

// 运行所有接口测试
func runAPITests() {
	baseURL := "http://localhost:8080"
	client := &http.Client{Timeout: 5 * time.Second}

	// 测试 API 信息
	testAPIInfo(client, baseURL+"/api-info")

	// 测试用户列表
	testUserList(client, baseURL+"/api/users")

	// 测试活跃用户列表
	testActiveUsers(client, baseURL+"/api/users/active")

	// 测试年龄统计
	testAgeStats(client, baseURL+"/api/users/age-stats")

	// 测试用户详情
	testUserDetail(client, baseURL+"/api/users/1")
}

// 测试 API 信息
func testAPIInfo(client *http.Client, url string) {
	fmt.Println("\n=== 测试 API 信息 ===")
	var result crud.CodeMsg
	if err := getJSON(client, url, &result); err != nil {
		fmt.Printf("获取 API 信息失败: %v\n", err)
		return
	}

	prettyPrint(result.Data)
}

// 测试用户列表
func testUserList(client *http.Client, url string) {
	fmt.Println("\n=== 测试用户列表 ===")

	// 测试基本列表
	fmt.Println("\n1. 基本列表：")
	var result crud.CodeMsg
	if err := getJSON(client, url, &result); err != nil {
		fmt.Printf("获取用户列表失败: %v\n", err)
		return
	}
	prettyPrint(result.Data)

	// 测试分页
	fmt.Println("\n2. 分页查询（第1页，每页5条）：")
	if err := getJSON(client, url+"?page=1&size=5", &result); err != nil {
		fmt.Printf("获取分页数据失败: %v\n", err)
		return
	}
	prettyPrint(result.Data)

	// 测试排序
	fmt.Println("\n3. 按年龄排序：")
	if err := getJSON(client, url+"?sort=age", &result); err != nil {
		fmt.Printf("获取排序数据失败: %v\n", err)
		return
	}
	prettyPrint(result.Data)
}

// 测试活跃用户列表
func testActiveUsers(client *http.Client, url string) {
	fmt.Println("\n=== 测试活跃用户列表 ===")
	var result crud.CodeMsg
	if err := getJSON(client, url, &result); err != nil {
		fmt.Printf("获取活跃用户失败: %v\n", err)
		return
	}

	prettyPrint(result.Data)
}

// 测试年龄统计
func testAgeStats(client *http.Client, url string) {
	fmt.Println("\n=== 测试年龄统计 ===")
	var result crud.CodeMsg
	if err := getJSON(client, url, &result); err != nil {
		fmt.Printf("获取年龄统计失败: %v\n", err)
		return
	}

	prettyPrint(result.Data)
}

// 测试用户详情
func testUserDetail(client *http.Client, url string) {
	fmt.Println("\n=== 测试用户详情 ===")
	var result crud.CodeMsg
	if err := getJSON(client, url, &result); err != nil {
		fmt.Printf("获取用户详情失败: %v\n", err)
		return
	}

	prettyPrint(result.Data)
}

// 辅助函数：发送 GET 请求并解析 JSON 响应
func getJSON(client *http.Client, url string, result interface{}) error {
	resp, err := client.Get(url)
	if err != nil {
		return fmt.Errorf("请求失败: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("读取响应失败: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("请求失败，状态码: %d，响应: %s", resp.StatusCode, string(body))
	}

	if err := json.Unmarshal(body, result); err != nil {
		return fmt.Errorf("解析 JSON 失败: %v", err)
	}

	return nil
}

// 辅助函数：格式化输出 JSON
func prettyPrint(v interface{}) {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		fmt.Printf("格式化 JSON 失败: %v\n", err)
		return
	}
	fmt.Println(string(data))
}
