package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/kmlixh/crud"
	"github.com/kmlixh/gom/v4"
	"github.com/kmlixh/gom/v4/define"
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

func (u User) TableName() string {
	return "example_users"
}

// 预定义用户状态
const (
	UserStatusActive   = "active"
	UserStatusInactive = "inactive"
	UserStatusBlocked  = "blocked"
)

func main() {
	// 连接数据库
	log.Println("Connecting to database...")
	db, err := gom.Open("mysql", "remote:123456@tcp(192.168.110.249:3306)/test?parseTime=true", &define.DBOptions{
		Debug: false,
	})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	log.Println("Testing database connection...")
	result := db.Chain().Table("example_users").List()
	// 处理时间戳
	if len(result.Data) > 0 {
		processTimeFields(result.Data)
	}
	// 打印第一个用户的用户名
	if len(result.Data) > 0 {
		if username, ok := result.Data[0]["username"]; ok {
			switch v := username.(type) {
			case []uint8:
				fmt.Println("================", string(v))
			case string:
				fmt.Println("================", v)
			}
		}
	}
	if err := result.Error; err != nil {
		log.Fatalf("Failed to test database connection: %v", err)
	}
	log.Println("Successfully connected to database")

	// 清空用户表
	log.Println("Clearing users table...")
	dr := db.Chain().Table("example_users").Delete()
	if dr.Error != nil {
		log.Fatalf("Failed to clear users table: %v", dr.Error)
	}
	log.Println("Successfully cleared users table")

	// 插入测试数据
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

	for _, user := range testData {
		values := map[string]interface{}{
			"username":   user.Username,
			"email":      user.Email,
			"age":        user.Age,
			"status":     user.Status,
			"created_at": time.Now(),
		}
		dr := db.Chain().Table("example_users").Values(values).Save()
		if dr.Error != nil {
			log.Printf("Failed to insert user %s: %v", user.Username, dr.Error)
		}
	}

	// 创建 Gin 路由
	r := gin.Default()
	r.Use(crud.Cors())

	// 创建用户 CRUD 实例
	userCrud := crud.New2(db, &User{}).
		SetDescription("用户管理模块，提供用户的增删改查功能")

	// 修改默认处理器的字段控制
	if listHandler, ok := userCrud.GetHandler(crud.LIST); ok {
		listHandler.AllowedFields = []string{"id", "username", "email", "status", "created_at"}
		listHandler.SetDescription("获取用户列表，支持分页和排序")
		userCrud.AddHandler(crud.LIST, http.MethodGet, listHandler)
	}

	// 创建分页处理器
	pageHandler, ok := userCrud.GetHandler(crud.PAGE)
	if !ok {
		log.Fatalf("Failed to get page handler")
	}
	pageHandler.SetDescription("分页获取用户列表")
	pageHandler.AllowedFields = []string{"id", "username", "email", "status", "created_at"}

	// 添加预处理器
	pageHandler.AddProcessor(crud.PreProcess, crud.OnPhase, func(ctx *crud.ProcessContext) error {
		if ctx.Chain == nil {
			ctx.Chain = db.Chain()
		}
		ctx.Chain = ctx.Chain.Table("example_users")
		return nil
	})

	// 添加查询构建器
	pageHandler.AddProcessor(crud.BuildQuery, crud.OnPhase, func(ctx *crud.ProcessContext) error {
		if ctx.Chain == nil {
			return fmt.Errorf("database chain is nil")
		}

		// 处理查询条件
		if err := crud.ParseQueryConditions(ctx.GinContext, ctx.Chain); err != nil {
			return err
		}

		// 处理排序
		if sort := ctx.GinContext.Query("sort"); sort != "" {
			if sort[0] == '-' {
				ctx.Chain = ctx.Chain.OrderByDesc(sort[1:])
			} else {
				ctx.Chain = ctx.Chain.OrderBy(sort)
			}
		}

		// 处理分页
		page := 1
		size := 10
		if p := ctx.GinContext.Query("page"); p != "" {
			if v, err := strconv.Atoi(p); err == nil && v > 0 {
				page = v
			}
		}
		if s := ctx.GinContext.Query("size"); s != "" {
			if v, err := strconv.Atoi(s); err == nil && v > 0 {
				size = v
			}
		}

		offset := (page - 1) * size
		ctx.Chain = ctx.Chain.Offset(offset).Limit(size)

		// 保存分页参数
		if ctx.Data == nil {
			ctx.Data = make(map[string]interface{})
		}
		ctx.Data["page"] = page
		ctx.Data["size"] = size

		return nil
	})

	// 添加执行处理器
	pageHandler.AddProcessor(crud.Execute, crud.OnPhase, func(ctx *crud.ProcessContext) error {
		if ctx.Chain == nil {
			return fmt.Errorf("database chain is nil")
		}

		// 获取总数
		countChain := db.Chain().Table("example_users")
		if err := crud.ParseQueryConditions(ctx.GinContext, countChain); err != nil {
			return err
		}
		total, err := countChain.Count()
		if err != nil {
			return err
		}

		// 获取分页数据
		result := ctx.Chain.List()
		if err := result.Error; err != nil {
			return err
		}

		// 构建响应数据
		page := ctx.Data["page"].(int)
		size := ctx.Data["size"].(int)

		if ctx.Data == nil {
			ctx.Data = make(map[string]interface{})
		}
		ctx.Data["result"] = map[string]interface{}{
			"list":  result.Data,
			"page":  page,
			"size":  size,
			"total": total,
			"pages": int(float64(total)/float64(size) + 0.9999),
		}
		return nil
	})

	// 添加后处理器
	pageHandler.AddProcessor(crud.PostProcess, crud.OnPhase, func(ctx *crud.ProcessContext) error {
		if ctx.Data == nil {
			ctx.Data = make(map[string]interface{})
		}
		if result, ok := ctx.Data["result"].(map[string]interface{}); ok {
			if list, ok := result["list"].([]map[string]interface{}); ok {
				processTimeFields(list)
				decodeUserFields(list)
			}
			crud.JsonOk(ctx.GinContext, result)
			return nil
		}
		return fmt.Errorf("invalid result format")
	})

	// 添加分页处理器
	userCrud.AddHandler(crud.PAGE, http.MethodGet, pageHandler)

	if detailHandler, ok := userCrud.GetHandler(crud.SINGLE); ok {
		detailHandler.AllowedFields = []string{"id", "username", "email", "age", "status", "created_at"}
		detailHandler.SetDescription("获取用户详细信息")
		userCrud.AddHandler(crud.SINGLE, http.MethodGet, detailHandler)
	}

	// 创建自定义处理器
	activeUsersHandler := crud.NewHandler("/active", http.MethodGet).
		SetDescription("获取所有活跃状态的用户列表")

	activeUsersHandler.AddProcessor(crud.PreProcess, crud.OnPhase, func(ctx *crud.ProcessContext) error {
		if ctx.Chain == nil {
			ctx.Chain = db.Chain()
		}
		ctx.Chain = ctx.Chain.Table("example_users")
		return nil
	})

	activeUsersHandler.AddProcessor(crud.BuildQuery, crud.OnPhase, func(ctx *crud.ProcessContext) error {
		if ctx.Chain == nil {
			ctx.Chain = db.Chain().Table("example_users")
		}
		ctx.Chain = ctx.Chain.Eq("status", UserStatusActive)
		return nil
	})

	activeUsersHandler.AddProcessor(crud.Execute, crud.OnPhase, func(ctx *crud.ProcessContext) error {
		if ctx.Chain == nil {
			return fmt.Errorf("database chain is nil")
		}
		result := ctx.Chain.List()
		if err := result.Error; err != nil {
			return err
		}
		if ctx.Data == nil {
			ctx.Data = make(map[string]interface{})
		}
		ctx.Data["result"] = result.Data
		return nil
	})

	activeUsersHandler.AddProcessor(crud.PostProcess, crud.OnPhase, func(ctx *crud.ProcessContext) error {
		if ctx.Data == nil {
			ctx.Data = make(map[string]interface{})
		}
		if result, ok := ctx.Data["result"].([]map[string]interface{}); ok {
			processTimeFields(result)
			decodeUserFields(result)
		}
		crud.JsonOk(ctx.GinContext, ctx.Data["result"])
		return nil
	})

	// 创建年龄统计处理器
	ageStatsHandler := crud.NewHandler("/age-stats", http.MethodGet).
		SetDescription("获取用户年龄统计信息")

	ageStatsHandler.AddProcessor(crud.PreProcess, crud.OnPhase, func(ctx *crud.ProcessContext) error {
		if ctx.Chain == nil {
			ctx.Chain = db.Chain()
		}
		ctx.Chain = ctx.Chain.Table("example_users")
		return nil
	})

	ageStatsHandler.AddProcessor(crud.Execute, crud.OnPhase, func(ctx *crud.ProcessContext) error {
		if ctx.Chain == nil {
			return fmt.Errorf("database chain is nil")
		}
		result := ctx.Chain.List()
		if err := result.Error; err != nil {
			return err
		}

		processTimeFields(result.Data)
		stats := make(map[string]int)
		for _, item := range result.Data {
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
			stats[group]++
		}

		if ctx.Data == nil {
			ctx.Data = make(map[string]interface{})
		}
		ctx.Data["result"] = stats
		return nil
	})

	ageStatsHandler.AddProcessor(crud.PostProcess, crud.OnPhase, func(ctx *crud.ProcessContext) error {
		if ctx.Data == nil {
			ctx.Data = make(map[string]interface{})
		}
		crud.JsonOk(ctx.GinContext, ctx.Data["result"])
		return nil
	})

	// 注册 API 路由
	apiGroup := r.Group("/api")
	userGroup := apiGroup.Group("/users")

	// 注册默认处理器
	userCrud.RegisterRoutes(userGroup, "")

	// 注册自定义处理器
	userGroup.GET("/active", activeUsersHandler.HandleRequest)
	userGroup.GET("/age-stats", ageStatsHandler.HandleRequest)

	// 注册 API 文档
	crud.RegisterCrud(userCrud)
	crud.RegisterApi(r, "/api-info")
	crud.RegisterApiDoc(r, "/api-doc")

	// 在新的 goroutine 中运行测试
	go func() {
		// 等待服务器启动
		time.Sleep(2 * time.Second)

		log.Println("Starting API tests...")
		client := &http.Client{Timeout: 10 * time.Second}

		// 运行所有测试
		testAPIInfo(client)
		testUserList(client)
		testActiveUsers(client)
		testAgeStats(client)

		log.Println("API tests completed")
		os.Exit(0)
	}()

	// 启动服务器
	log.Println("Starting server on :8080...")
	if err := r.Run(":8080"); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

// 测试 API 信息
func testAPIInfo(client *http.Client) {
	fmt.Println("\n测试 API 信息...")
	resp, err := client.Get("http://localhost:8080/api-info")
	if err != nil {
		fmt.Printf("请求失败: %v\n", err)
		return
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		fmt.Printf("解析响应失败: %v\n", err)
		return
	}

	fmt.Printf("API 信息: %+v\n", result)
}

// 测试用户列表
func testUserList(client *http.Client) {
	fmt.Println("\n测试用户列表...")
	resp, err := client.Get("http://localhost:8080/api/users/page")
	if err != nil {
		fmt.Printf("请求失败: %v\n", err)
		return
	}
	defer resp.Body.Close()

	fmt.Printf("状态码: %d\n", resp.StatusCode)
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("读取响应失败: %v\n", err)
		return
	}

	// 打印原始响应
	fmt.Println("\n原始响应:")
	fmt.Println(string(body))

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		fmt.Printf("解析响应失败: %v\n", err)
		return
	}

	fmt.Println("\n解析后的结果:")
	prettyPrint(result)
}

// 测试活跃用户列表
func testActiveUsers(client *http.Client) {
	fmt.Println("\n测试活跃用户列表...")
	resp, err := client.Get("http://localhost:8080/api/users/active")
	if err != nil {
		fmt.Printf("请求失败: %v\n", err)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("读取响应失败: %v\n", err)
		return
	}

	fmt.Println("\n原始响应:")
	fmt.Println(string(body))

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		fmt.Printf("解析响应失败: %v\n", err)
		return
	}

	fmt.Println("\n解析后的结果:")
	prettyPrint(result)
}

// 测试年龄统计
func testAgeStats(client *http.Client) {
	fmt.Println("\n测试年龄统计...")
	resp, err := client.Get("http://localhost:8080/api/users/age-stats")
	if err != nil {
		fmt.Printf("请求失败: %v\n", err)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("读取响应失败: %v\n", err)
		return
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		fmt.Printf("解析响应失败: %v\n", err)
		return
	}

	fmt.Println("\n解析后的结果:")
	prettyPrint(result)
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
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		fmt.Printf("格式化 JSON 失败: %v\n", err)
		return
	}
	fmt.Println(string(b))
}

// 添加 base64 解码函数
func decodeBase64(s string) string {
	decoded, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return s
	}
	return string(decoded)
}

// decodeUserFields 解码用户字段
func decodeUserFields(users []map[string]interface{}) {
	for _, user := range users {
		// 处理 username
		if username, ok := user["username"]; ok {
			switch v := username.(type) {
			case []uint8:
				user["username"] = string(v)
			case string:
				// 已经是字符串，不需要处理
			}
		}

		// 处理 email
		if email, ok := user["email"]; ok {
			switch v := email.(type) {
			case []uint8:
				user["email"] = string(v)
			case string:
				// 已经是字符串，不需要处理
			}
		}

		// 处理 status
		if status, ok := user["status"]; ok {
			switch v := status.(type) {
			case []uint8:
				user["status"] = string(v)
			case string:
				// 已经是字符串，不需要处理
			}
		}

		// 处理 created_at
		if createdAt, ok := user["created_at"]; ok {
			switch v := createdAt.(type) {
			case []uint8:
				user["created_at"] = string(v)
			case string:
				// 已经是字符串，不需要处理
			case time.Time:
				// 已经是时间类型，不需要处理
			}
		}
	}
}

// 处理时间戳字段
func processTimeFields(data []map[string]interface{}) {
	for _, item := range data {
		if createdAt, ok := item["created_at"]; ok {
			switch v := createdAt.(type) {
			case []uint8:
				if t, err := time.Parse("2006-01-02 15:04:05", string(v)); err == nil {
					item["created_at"] = t
				}
			}
		}
	}
}
