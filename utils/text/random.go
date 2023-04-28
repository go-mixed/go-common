package textUtils

import (
	cryptoRand "crypto/rand"
	mathRand "math/rand"
	"time"
)

// GenerateRandomBytes 生成指定长度的随机字节
func GenerateRandomBytes(length int) []byte {
	salt := make([]byte, length)
	_, err := cryptoRand.Read(salt)
	if err != nil {
		salt = []byte(GenerateRandomString(length))
	}
	return salt
}

// GenerateRandomString 生成指定长度的随机可视字符串
func GenerateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	r := mathRand.New(mathRand.NewSource(time.Now().UnixNano()))
	result := make([]byte, length)
	for i := range result {
		result[i] = charset[r.Intn(len(charset))]
	}

	return string(result)
}
