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
