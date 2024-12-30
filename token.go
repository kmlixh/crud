package crud

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// TokenDetail Token详情
type TokenDetail struct {
	Token     string        `json:"token"`
	UserID    string        `json:"user_id"`
	UserType  string        `json:"user_type"`
	ExpiresIn time.Duration `json:"expires_in"`
}

var (
	// defaultTokenStore 默认的Token存储
	defaultTokenStore TokenStore
)

// SetTokenStore 设置全局Token存储
func SetTokenStore(store TokenStore) {
	defaultTokenStore = store
}

// GetTokenStore 获取全局Token存储
func GetTokenStore() TokenStore {
	return defaultTokenStore
}

// GlobalCheckToken 全局检查token是否合法
func GlobalCheckToken(token string) bool {
	if defaultTokenStore == nil {
		return false
	}
	return defaultTokenStore.CheckToken(token)
}

// GlobalCheckTokenGin 全局检查请求头中的token是否合法
func GlobalCheckTokenGin(c *gin.Context) bool {
	if defaultTokenStore == nil {
		return false
	}
	if store, ok := defaultTokenStore.(*RedisTokenStore); ok {
		return store.CheckTokenGin(c)
	}
	return false
}

// GlobalTokenAuthMiddleware 全局Token验证中间件
func GlobalTokenAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !GlobalCheckTokenGin(c) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code":    CodeError,
				"message": "unauthorized",
			})
			return
		}
		c.Next()
	}
}

// TokenStore Token存储接口
type TokenStore interface {
	SaveToken(detail TokenDetail) error
	GetToken(token string) (*TokenDetail, error)
	DeleteToken(token string) error
	GetUserTokens(userID, userType string) ([]string, error)
	CheckToken(token string) bool
}

// RedisTokenStore Redis实现的Token存储
type RedisTokenStore struct {
	client *redis.Client
	prefix string
}

// NewRedisTokenStore 创建Redis Token存储
func NewRedisTokenStore(client *redis.Client, prefix string) *RedisTokenStore {
	return &RedisTokenStore{
		client: client,
		prefix: prefix,
	}
}

// SaveToken 保存Token
func (s *RedisTokenStore) SaveToken(detail TokenDetail) error {
	data, err := json.Marshal(detail)
	if err != nil {
		return err
	}

	ctx := context.Background()
	key := fmt.Sprintf("%s:token:%s", s.prefix, detail.Token)
	userKey := fmt.Sprintf("%s:user:%s:%s", s.prefix, detail.UserType, detail.UserID)

	pipe := s.client.Pipeline()
	pipe.Set(ctx, key, data, detail.ExpiresIn)
	pipe.SAdd(ctx, userKey, detail.Token)
	pipe.Expire(ctx, userKey, detail.ExpiresIn)

	_, err = pipe.Exec(ctx)
	return err
}

// GetToken 获取Token信息
func (s *RedisTokenStore) GetToken(token string) (*TokenDetail, error) {
	ctx := context.Background()
	key := fmt.Sprintf("%s:token:%s", s.prefix, token)

	data, err := s.client.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, err
	}

	var detail TokenDetail
	if err := json.Unmarshal(data, &detail); err != nil {
		return nil, err
	}

	return &detail, nil
}

// DeleteToken 删除Token
func (s *RedisTokenStore) DeleteToken(token string) error {
	detail, err := s.GetToken(token)
	if err != nil {
		return err
	}
	if detail == nil {
		return nil
	}

	ctx := context.Background()
	key := fmt.Sprintf("%s:token:%s", s.prefix, token)
	userKey := fmt.Sprintf("%s:user:%s:%s", s.prefix, detail.UserType, detail.UserID)

	pipe := s.client.Pipeline()
	pipe.Del(ctx, key)
	pipe.SRem(ctx, userKey, token)
	_, err = pipe.Exec(ctx)
	return err
}

// GetUserTokens 获取用户的所有Token
func (s *RedisTokenStore) GetUserTokens(userID, userType string) ([]string, error) {
	ctx := context.Background()
	key := fmt.Sprintf("%s:user:%s:%s", s.prefix, userType, userID)
	return s.client.SMembers(ctx, key).Result()
}

// GenerateToken 生成新的Token
func GenerateToken(userID, userType string, expiresIn time.Duration) TokenDetail {
	return TokenDetail{
		Token:     uuid.New().String(),
		UserID:    userID,
		UserType:  userType,
		ExpiresIn: expiresIn,
	}
}

// CheckToken 检查指定token是否合法
func (s *RedisTokenStore) CheckToken(token string) bool {
	detail, err := s.GetToken(token)
	if err != nil {
		return false
	}
	return detail != nil
}

// CheckTokenGin 检查请求头中是否包含token，并校验token是否合法
func (s *RedisTokenStore) CheckTokenGin(c *gin.Context) bool {
	// 从请求头中获取token
	token := c.GetHeader("Authorization")
	if token == "" {
		// 如果请求头中没有token，尝试从查询参数中获取
		token = c.Query("token")
		if token == "" {
			return false
		}
	}

	// 如果token以"Bearer "开头，去掉这个前缀
	if len(token) > 7 && token[:7] == "Bearer " {
		token = token[7:]
	}

	// 检查token是否合法
	detail, err := s.GetToken(token)
	if err != nil {
		return false
	}

	// 如果token合法，将用户信息存储到上下文中
	if detail != nil {
		c.Set("user_id", detail.UserID)
		c.Set("user_type", detail.UserType)
		c.Set("token", detail.Token)
		return true
	}

	return false
}

// TokenAuthMiddleware Gin中间件，用于验证token
func TokenAuthMiddleware(store *RedisTokenStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !store.CheckTokenGin(c) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code":    CodeError,
				"message": "unauthorized",
			})
			return
		}
		c.Next()
	}
}

// GetCurrentUser 从上下文中获取当前用户信息
func GetCurrentUser(c *gin.Context) (userID, userType string, ok bool) {
	userIDVal, ok1 := c.Get("user_id")
	userTypeVal, ok2 := c.Get("user_type")
	if !ok1 || !ok2 {
		return "", "", false
	}
	userID, ok1 = userIDVal.(string)
	userType, ok2 = userTypeVal.(string)
	if !ok1 || !ok2 {
		return "", "", false
	}
	return userID, userType, true
}

// GetCurrentToken 从上下文中获取当前token
func GetCurrentToken(c *gin.Context) (token string, ok bool) {
	tokenVal, exists := c.Get("token")
	if !exists {
		return "", false
	}
	token, ok = tokenVal.(string)
	if !ok {
		return "", false
	}
	return token, true
}

// GlobalSaveToken 全局保存Token
func GlobalSaveToken(detail TokenDetail) error {
	if defaultTokenStore == nil {
		return fmt.Errorf("token store not initialized")
	}
	return defaultTokenStore.SaveToken(detail)
}

// GlobalGetToken 全局获取Token信息
func GlobalGetToken(token string) (*TokenDetail, error) {
	if defaultTokenStore == nil {
		return nil, fmt.Errorf("token store not initialized")
	}
	return defaultTokenStore.GetToken(token)
}

// GlobalDeleteToken 全局删除Token
func GlobalDeleteToken(token string) error {
	if defaultTokenStore == nil {
		return fmt.Errorf("token store not initialized")
	}
	return defaultTokenStore.DeleteToken(token)
}

// GlobalGetUserTokens 全局获取用户的所有Token
func GlobalGetUserTokens(userID, userType string) ([]string, error) {
	if defaultTokenStore == nil {
		return nil, fmt.Errorf("token store not initialized")
	}
	return defaultTokenStore.GetUserTokens(userID, userType)
}

// GlobalGenerateToken 全局生成新的Token
func GlobalGenerateToken(userID, userType string, expiresIn time.Duration) (TokenDetail, error) {
	token := GenerateToken(userID, userType, expiresIn)
	if err := GlobalSaveToken(token); err != nil {
		return TokenDetail{}, fmt.Errorf("failed to save token: %v", err)
	}
	return token, nil
}

// GlobalGenerateAndSaveToken 全局生成并保存Token，如果已存在则先删除
func GlobalGenerateAndSaveToken(userID, userType string, expiresIn time.Duration) (TokenDetail, error) {
	if defaultTokenStore == nil {
		return TokenDetail{}, fmt.Errorf("token store not initialized")
	}

	// 获取用户现有的所有token
	existingTokens, err := GlobalGetUserTokens(userID, userType)
	if err != nil {
		return TokenDetail{}, fmt.Errorf("failed to get user tokens: %v", err)
	}

	// 删除现有的token
	for _, t := range existingTokens {
		if err := GlobalDeleteToken(t); err != nil {
			return TokenDetail{}, fmt.Errorf("failed to delete existing token: %v", err)
		}
	}

	// 生成并保存新token
	return GlobalGenerateToken(userID, userType, expiresIn)
}

// GlobalValidateAndRefreshToken 全局验证Token并刷新过期时间
func GlobalValidateAndRefreshToken(token string, expiresIn time.Duration) error {
	if defaultTokenStore == nil {
		return fmt.Errorf("token store not initialized")
	}

	// 获取token信息
	detail, err := GlobalGetToken(token)
	if err != nil {
		return fmt.Errorf("failed to get token: %v", err)
	}
	if detail == nil {
		return fmt.Errorf("token not found")
	}

	// 更新过期时间
	detail.ExpiresIn = expiresIn
	if err := GlobalSaveToken(*detail); err != nil {
		return fmt.Errorf("failed to refresh token: %v", err)
	}

	return nil
}

// GlobalGetCurrentUserDetail 全局获取当前用户的完整Token信息
func GlobalGetCurrentUserDetail(c *gin.Context) (*TokenDetail, error) {
	token, ok := GetCurrentToken(c)
	if !ok {
		return nil, fmt.Errorf("token not found in context")
	}
	return GlobalGetToken(token)
}

// GlobalCheckTokenWithType 全局检查token是否合法并验证用户类型
func GlobalCheckTokenWithType(token, requiredType string) bool {
	if defaultTokenStore == nil {
		return false
	}
	detail, err := GlobalGetToken(token)
	if err != nil || detail == nil {
		return false
	}
	return detail.UserType == requiredType
}

// GlobalTokenAuthMiddlewareWithType 全局Token验证中间件（带用户类型验证）
func GlobalTokenAuthMiddlewareWithType(requiredType string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !GlobalCheckTokenGin(c) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code":    CodeError,
				"message": "unauthorized",
			})
			return
		}

		userType, _, ok := GetCurrentUser(c)
		if !ok || userType != requiredType {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"code":    CodeError,
				"message": "forbidden",
			})
			return
		}

		c.Next()
	}
}
