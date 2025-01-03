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

func NewJwtUtil(opts ...JwtOption) (*JwtUtil, error) {
	ju := &JwtUtil{
		signKey:     []byte("default_key"),
		expireTime:  7 * 24 * time.Hour,
		refreshTime: 24 * time.Hour,
	}

	for _, opt := range opts {
		opt(ju)
	}

	// 检查 refreshTime 是否小于 expireTime
	if ju.refreshTime >= ju.expireTime {
		return nil, errors.New("refreshTime must be less than expireTime")
	}

	// 检查 signKey 是否为空
	if len(ju.signKey) == 0 {
		return nil, errors.New("signKey cannot be empty")
	}

	return ju, nil
}

func (j *JwtUtil) Generate(info any) (string, error) {
	claims := jwt.MapClaims{"exp": float64(time.Now().Add(j.expireTime).Unix()), "info": info}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(j.signKey)
}

func (j *JwtUtil) Parse(tokenStr string) (info any, err error) {
	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return j.signKey, nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %v", err)
	}
	if !token.Valid {
		err = errors.New("invalid token")
		return
	}
	if mapClaim, ok := token.Claims.(jwt.MapClaims); ok {
		return mapClaim["info"], nil
	}
	err = errors.New("invalid token")
	return
}
