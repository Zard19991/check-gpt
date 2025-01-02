package util

import (
	"math/rand"
	"time"
)

func init() {
	// Initialize random seed
	rand.Seed(time.Now().UnixNano())
}

// GenerateRandomDigits generates a string of random digits with the specified length
func GenerateRandomDigits(length int) string {
	digits := make([]byte, length)
	for i := 0; i < length; i++ {
		digits[i] = byte('0' + rand.Intn(10))
	}
	return string(digits)
}

// GenerateRandomString generates a random string with the specified length
func GenerateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	result := make([]byte, length)
	for i := range result {
		result[i] = charset[rand.Intn(len(charset))]
	}
	return string(result)
}
