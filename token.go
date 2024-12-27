package crud

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

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
