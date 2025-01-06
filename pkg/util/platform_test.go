package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetPlatformInfo(t *testing.T) {
	tests := []struct {
		name      string
		userAgent string
		ip        string
		cidrs     []string
		want      string
	}{
		{
			name:      "Unknown platform",
			userAgent: "unknown",
			ip:        "1.1.1.1",
			cidrs:     []string{},
			want:      "未知服务,User-Agent:unknown",
		},
		{
			name:      "OpenAI platform",
			userAgent: "curl/7.64.1",
			ip:        "23.102.140.120",
			cidrs:     []string{"23.102.140.112/28"},
			want:      "OpenAI服务",
		},
		{
			name:      "OpenAI platform",
			userAgent: "OpenAI,image download	",
			ip:        "1.102.140.120",
			cidrs:     []string{"23.102.140.112/28"},
			want:      "可能是OpenAI服务",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetPlatformInfo(tt.userAgent, tt.ip, tt.cidrs)
			assert.Equal(t, tt.want, got)
		})
	}
}
