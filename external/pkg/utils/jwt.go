package utils

import (
	"errors"
	"fmt"
	"github.com/golang-jwt/jwt/v5"
	"time"
)

type JwtUtil struct {
	signKey     []byte
	expireTime  time.Duration
	refreshTime time.Duration
}

type JwtOption func(*JwtUtil)

func WithSignKey(key []byte) JwtOption {
	return func(j *JwtUtil) {
		j.signKey = key
	}
}

func WithExpireTime(d time.Duration) JwtOption {
	return func(j *JwtUtil) {
		j.expireTime = d
	}
}

func WithRefreshTime(d time.Duration) JwtOption {
	return func(j *JwtUtil) {
		j.refreshTime = d
	}
}

func NewJwtUtil(opts ...JwtOption) *JwtUtil {
	ju := &JwtUtil{
		signKey:     []byte("default_key"),
		expireTime:  24 * time.Hour,
		refreshTime: 7 * 24 * time.Hour,
	}

	for _, opt := range opts {
		opt(ju)
	}

	return ju
}

func (j *JwtUtil) Generate(claims jwt.Claims) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(j.signKey)
}

func (j *JwtUtil) Parse(tokenStr string) (jwt.Claims, error) {
	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return j.signKey, nil
	})

	if err != nil {
		return nil, fmt.Errorf("解析token失败: %v", err)
	}

	if !token.Valid {
		return nil, errors.New("无效的token")
	}

	return token.Claims, nil
}

func (j *JwtUtil) NeedRefresh(claims jwt.Claims) bool {
	if standardClaims, ok := claims.(jwt.RegisteredClaims); ok {
		if exp := standardClaims.ExpiresAt; exp != nil {
			remaining := time.Until(exp.Time)
			return remaining < j.refreshTime
		}
	}
	return false
}
