package util

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// IPInfo represents the response from IP-API
type IPInfo struct {
	Status     string `json:"status"`
	Country    string `json:"country"`
	RegionName string `json:"regionName"`
	City       string `json:"city"`
	ISP        string `json:"isp"`
	Query      string `json:"query"`
	Org        string `json:"org"`
}

// GetIPInfo retrieves location and ISP information for an IP address
func GetIPInfo(ip string) (*IPInfo, error) {
	if ip == "" {
		return nil, fmt.Errorf("empty IP address")
	}

	resp, err := http.Get(fmt.Sprintf("http://ip-api.com/json/%s", ip))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var info IPInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, err
	}

	if info.Status != "success" {
		return nil, fmt.Errorf("IP lookup failed: %s", info.Status)
	}

	return &info, nil
}
