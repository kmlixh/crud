package crud

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"strings"
	"time"
)

type TokenDetail struct {
	Token    string
	UserId   string
	UserType string
	Expire   time.Duration
}

func (t *TokenDetail) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, t)
}

func (t TokenDetail) MarshalBinary() (data []byte, err error) {
	return json.Marshal(t)
}

type TokenStore interface {
	SaveToken(detail TokenDetail) error
	GenerateToken(userId string, userType string) string
	GetTokenDetail(token string) (TokenDetail, error)
	DeleteToken(token string) error
	GetTokensOfUser(userId string, userType string) []string
}
type RedisTokenStore struct {
	RedisClient *redis.Client
}

func (r RedisTokenStore) SaveToken(detail TokenDetail) error {
	cmd := r.RedisClient.Set(context.Background(), "TOKEN_USER_"+detail.UserType+"_"+detail.UserId+"_"+detail.Token, detail, detail.Expire)
	if cmd.Err() != nil {
		return cmd.Err()
	}
	cmds := r.RedisClient.Set(context.Background(), "TOKEN_"+detail.Token, detail, detail.Expire)
	return cmds.Err()
}

func (r RedisTokenStore) GenerateToken(userId string, userType string) string {
	uuid, er := uuid.NewUUID()
	if er != nil {
		return ""
	}
	return uuid.String()
}

func (r RedisTokenStore) GetTokenDetail(token string) (TokenDetail, error) {
	cmd := r.RedisClient.Get(context.Background(), "TOKEN_"+token)
	if cmd.Err() != nil {
		return TokenDetail{}, cmd.Err()
	}
	var detail TokenDetail
	er := cmd.Scan(&detail)
	if er != nil || detail.Token == "" {
		return TokenDetail{}, errors.New("token not found")
	}
	return detail, er
}

func (r RedisTokenStore) GetTokensOfUser(userId string, userType string) []string {
	var results []string
	prefix := "TOKEN_USER_" + userId + "_" + userType + "_"
	ctx := context.Background()
	it := r.RedisClient.Scan(ctx, 0, prefix+"*", 1).Iterator()
	for it.Next(ctx) {
		results = append(results, strings.TrimLeft(it.Val(), prefix))
	}
	return results
}

func (r RedisTokenStore) DeleteToken(token string) error {
	detail, er := r.GetTokenDetail(token)
	if er != nil {
		return er
	}
	cmd := r.RedisClient.Del(context.Background(), "TOKEN_USER_"+detail.UserType+"_"+detail.UserId+"_"+detail.Token, "TOKEN_"+detail.Token)
	return cmd.Err()
}

func NewRedisStore(client *redis.Client) RedisTokenStore {
	return RedisTokenStore{client}
}

var store TokenStore

func SetStore(tokenStore TokenStore) {
	store = tokenStore
}

func GenTokenForUser(userId string, userType string, expire time.Duration) (string, error) {
	token := store.GenerateToken(userId, userType)
	detail := TokenDetail{Token: token, UserId: userId, UserType: userType, Expire: expire}
	er := store.SaveToken(detail)
	return token, er
}
func CheckToken(token string) bool {
	_, er := store.GetTokenDetail(token)
	return er == nil
}
func CheckTokenGin(c *gin.Context) {
	token := c.GetHeader("token")
	if token == "" {
		RenderJson(c, Err2(403, "unauthorized!"))
		c.Abort()
	} else {
		if CheckToken(token) {
			detail, er := store.GetTokenDetail(token)
			if er == nil && detail.UserId != "" {
				c.Set("userId", detail.UserId)
			}
			c.Next()
		} else {
			RenderJson(c, Err2(403, "unauthorized!"))
			c.Abort()
		}
	}
}
func GetTokensOfUser(userId string, userType string) []string {
	return store.GetTokensOfUser(userId, userType)
}
