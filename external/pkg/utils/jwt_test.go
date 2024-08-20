package utils

import (
	"fmt"
	"github.com/golang-jwt/jwt/v5"
	"testing"
)

func TestGetJwtUtil(t *testing.T) {
	key := []byte("asdfasdf")
	tokenStr, err := GetJwtUtil().Generate(jwt.MapClaims{"name": "张三"}, key)
	if err != nil {
		t.Error(err)
		return
	}
	claims, err := GetJwtUtil().Parse(tokenStr, key)
	if err != nil {
		t.Error(err)
		return
	}
	fmt.Println(claims)
}
