package ipinfo

import (
	"github.com/go-coders/check-trace/pkg/util"
)

// Provider defines the interface for getting IP information
type Provider interface {
	GetIPInfo(ip string) (*Info, error)
}

// Info represents IP location information
type Info struct {
	Country    string
	City       string
	RegionName string
	ISP        string
}

// DefaultProvider implements Provider using the util package
type DefaultProvider struct{}

// NewProvider creates a new default IP info provider
func NewProvider() Provider {
	return &DefaultProvider{}
}

func (p *DefaultProvider) GetIPInfo(ip string) (*Info, error) {
	info, err := util.GetIPInfo(ip)
	if err != nil {
		return nil, err
	}
	return &Info{
		Country:    info.Country,
		City:       info.City,
		RegionName: info.RegionName,
		ISP:        info.ISP,
	}, nil
}
