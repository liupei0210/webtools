package utils

import (
	"testing"
	"time"
)

func TestGetJwtUtil(t *testing.T) {
	key := []byte("k9606")
	instance, err := NewJwtUtil(WithSignKey(key))
	if err != nil {
		t.Error(err)
		return
	}
	tokenStr, err := instance.Generate("张三")
	if err != nil {
		t.Error(err)
		return
	}
	println(tokenStr)
	time.Sleep(3 * time.Second)
	info, err := instance.Parse(tokenStr)
	if err != nil {
		t.Error(err)
		return
	}
	println(info.(string))
}
