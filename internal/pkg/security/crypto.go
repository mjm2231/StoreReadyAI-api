package security

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"

	"golang.org/x/crypto/bcrypt"
)

// HashPassword 使用 bcrypt 对密码加密。
// cost 建议：10~12（开发 10，生产可 12）。
func HashPassword(password string, cost int) (string, error) {
	if password == "" {
		return "", errors.New("password 不能为空")
	}
	if cost <= 0 {
		cost = bcrypt.DefaultCost
	}
	b, err := bcrypt.GenerateFromPassword([]byte(password), cost)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// ComparePassword 校验密码。
func ComparePassword(hashed, password string) bool {
	if hashed == "" || password == "" {
		return false
	}
	return bcrypt.CompareHashAndPassword([]byte(hashed), []byte(password)) == nil
}

// RandString 生成安全随机字符串（URL 安全）。
//
// length 表示输出长度（字符数），内部使用 base64 URL 编码。
func RandString(length int) (string, error) {
	if length <= 0 {
		length = 32
	}
	// base64 每 3 字节 -> 4 字符，预估需要的字节数
	need := (length*3)/4 + 1
	buf := make([]byte, need)
	if _, err := io.ReadFull(rand.Reader, buf); err != nil {
		return "", err
	}
	s := base64.RawURLEncoding.EncodeToString(buf)
	if len(s) > length {
		s = s[:length]
	}
	return s, nil
}

// MustRandString 启动期生成随机串，失败则 panic（仅建议启动期用）。
func MustRandString(length int) string {
	s, err := RandString(length)
	if err != nil {
		panic(fmt.Sprintf("rand string failed: %v", err))
	}
	return s
}

// HMACSHA256 计算 HMAC-SHA256（返回 base64url）。
func HMACSHA256(key []byte, msg []byte) string {
	h := hmac.New(sha256.New, key)
	_, _ = h.Write(msg)
	sum := h.Sum(nil)
	return base64.RawURLEncoding.EncodeToString(sum)
}

// ConstantTimeEqual 常量时间比较（避免时序攻击）。
func ConstantTimeEqual(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	return hmac.Equal([]byte(a), []byte(b))
}
