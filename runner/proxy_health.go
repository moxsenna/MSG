package runner

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync/atomic"
	"time"

	"github.com/gosom/google-maps-scraper/log"
)

var proxyErrorTotal atomic.Int64

func ProxyErrorTotal() int64 {
	return proxyErrorTotal.Load()
}

func CheckProxies(ctx context.Context, proxies []string) []string {
	if len(proxies) == 0 {
		return proxies
	}

	healthy := make([]string, 0, len(proxies))
	for _, p := range proxies {
		trimmed := strings.TrimSpace(p)
		if trimmed == "" {
			continue
		}

		parsed, err := url.Parse(trimmed)
		if err != nil {
			proxyErrorTotal.Add(1)
			log.Error("proxy invalid URL, blacklisted", "proxy", trimmed, "error", err)
			continue
		}

		switch strings.ToLower(parsed.Scheme) {
		case "socks5", "socks5h":
			log.Info("proxy socks check skipped (no HEAD probe)", "proxy", trimmed)
			healthy = append(healthy, trimmed)
			continue
		case "http", "https":
		default:
			proxyErrorTotal.Add(1)
			log.Error("proxy unsupported scheme, blacklisted", "proxy", trimmed, "scheme", parsed.Scheme)
			continue
		}

		if err := probeProxy(ctx, trimmed); err != nil {
			proxyErrorTotal.Add(1)
			log.Error("proxy health check failed, blacklisted", "proxy", trimmed, "error", err)
			continue
		}

		log.Info("proxy healthy", "proxy", trimmed)
		healthy = append(healthy, trimmed)
	}

	if len(healthy) == 0 && len(proxies) > 0 {
		log.Error("all proxies failed health check, proceeding with empty proxy list — scraper may be blocked",
			"total", len(proxies))
	}

	if len(healthy) < len(proxies) {
		log.Info("proxy health summary",
			"healthy", len(healthy),
			"blacklisted", len(proxies)-len(healthy),
			"proxy_error_total", proxyErrorTotal.Load(),
		)
	}

	return healthy
}

func probeProxy(ctx context.Context, proxyURL string) error {
	parsed, err := url.Parse(proxyURL)
	if err != nil {
		return fmt.Errorf("parse: %w", err)
	}

	transport := &http.Transport{
		Proxy: http.ProxyURL(parsed),
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   5 * time.Second,
	}

	reqCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodHead, "https://www.google.com/generate_204", http.NoBody)
	if err != nil {
		return err
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 500 {
		return fmt.Errorf("unexpected status %d", resp.StatusCode)
	}

	return nil
}
