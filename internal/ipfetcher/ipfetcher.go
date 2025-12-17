package ipfetcher

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"backup-to-oss/internal/logger"
)

const (
	maxRetries = 3
	timeout    = 10 * time.Second
)

// PublicIPFetcher 公网IP获取器
type PublicIPFetcher struct {
	maxRetries int
	timeout    time.Duration
}

// NewPublicIPFetcher 创建新的公网IP获取器
func NewPublicIPFetcher() *PublicIPFetcher {
	return &PublicIPFetcher{
		maxRetries: maxRetries,
		timeout:    timeout,
	}
}

// Fetch 按顺序尝试多个服务获取公网IP
func (f *PublicIPFetcher) Fetch() (string, error) {
	methods := []struct {
		name   string
		method func() (string, error)
	}{
		{"ipinfo.io", f.fromIPInfo},
		{"httpbin.org", f.fromHTTPBin},
		{"ip.sb", f.fromIPSB},
		{"ip-scan.adspower.net", f.fromIPScanAdspower},
	}

	logger.Debug("正在尝试获取公网IP")
	for _, m := range methods {
		ip, err := m.method()
		if err == nil && ip != "" {
			logger.Info("成功获取到公网IP", "service", m.name, "ip", ip)
			return ip, nil
		}
		logger.Debug("获取公网IP失败，继续尝试下一个服务", "service", m.name, "error", err)
	}

	return "", fmt.Errorf("所有服务都无法获取公网IP")
}

// fromIPInfo 从 ipinfo.io 获取公网IP
func (f *PublicIPFetcher) fromIPInfo() (string, error) {
	url := "https://ipinfo.io/ip"
	return f.fetchSimple(url)
}

// fromHTTPBin 从 httpbin.org 获取公网IP
func (f *PublicIPFetcher) fromHTTPBin() (string, error) {
	url := "https://httpbin.org/ip"
	client := &http.Client{Timeout: f.timeout}

	for attempt := 0; attempt < f.maxRetries; attempt++ {
		resp, err := client.Get(url)
		if err != nil {
			if attempt < f.maxRetries-1 {
				time.Sleep(f.sleepDuration(attempt))
				continue
			}
			return "", err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			if attempt < f.maxRetries-1 {
				time.Sleep(f.sleepDuration(attempt))
				continue
			}
			return "", fmt.Errorf("HTTP status: %d", resp.StatusCode)
		}

		var data struct {
			Origin string `json:"origin"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
			return "", err
		}

		if data.Origin != "" {
			return strings.TrimSpace(data.Origin), nil
		}
	}

	return "", fmt.Errorf("failed to get IP from httpbin")
}

// fromIPSB 从 ip.sb 获取公网IP
func (f *PublicIPFetcher) fromIPSB() (string, error) {
	url := "http://ip.sb"
	return f.fetchSimple(url)
}

// fromIPScanAdspower 从 ip-scan.adspower.net 获取公网IP
func (f *PublicIPFetcher) fromIPScanAdspower() (string, error) {
	url := "https://ip-scan.adspower.net/sys/config/ip/get-visitor-ip"
	client := &http.Client{Timeout: f.timeout}

	for attempt := 0; attempt < f.maxRetries; attempt++ {
		resp, err := client.Get(url)
		if err != nil {
			if attempt < f.maxRetries-1 {
				time.Sleep(f.sleepDuration(attempt))
				continue
			}
			return "", err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			if attempt < f.maxRetries-1 {
				time.Sleep(f.sleepDuration(attempt))
				continue
			}
			return "", fmt.Errorf("HTTP status: %d", resp.StatusCode)
		}

		var data struct {
			Data struct {
				IP string `json:"ip"`
			} `json:"data"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
			return "", err
		}

		if data.Data.IP != "" {
			return strings.TrimSpace(data.Data.IP), nil
		}
	}

	return "", fmt.Errorf("failed to get IP from ip-scan.adspower")
}

// fetchSimple 从简单文本API获取IP
func (f *PublicIPFetcher) fetchSimple(url string) (string, error) {
	client := &http.Client{Timeout: f.timeout}

	for attempt := 0; attempt < f.maxRetries; attempt++ {
		resp, err := client.Get(url)
		if err != nil {
			if attempt < f.maxRetries-1 {
				time.Sleep(f.sleepDuration(attempt))
				continue
			}
			return "", err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			if attempt < f.maxRetries-1 {
				time.Sleep(f.sleepDuration(attempt))
				continue
			}
			return "", fmt.Errorf("HTTP status: %d", resp.StatusCode)
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return "", err
		}

		ip := strings.TrimSpace(string(body))
		if ip != "" {
			return ip, nil
		}
	}

	return "", fmt.Errorf("failed to get IP")
}

// sleepDuration 计算重试等待时间（带抖动）
func (f *PublicIPFetcher) sleepDuration(attempt int) time.Duration {
	base := 2 * time.Second
	jitter := time.Duration(attempt) * 100 * time.Millisecond
	return base + jitter
}

