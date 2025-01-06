package util

import (
	"fmt"
	"strings"
)

// MaskKey masks a key string by showing only the first and last n characters
func MaskKey(key string, firstN, lastN int) string {
	if key == "" {
		return ""
	}

	keyLen := len(key)
	if keyLen <= firstN+lastN {
		return key
	}

	firstPart := key[:firstN]
	lastPart := key[keyLen-lastN:]
	maskedPart := strings.Repeat("*", 3)

	return fmt.Sprintf("%s%s%s", firstPart, maskedPart, lastPart)
}
