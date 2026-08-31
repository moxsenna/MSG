package gmaps

import (
	"net/http"
	"strconv"
	"time"
)

const (
	baseBackoff = 500 * time.Millisecond
	maxBackoff  = 30 * time.Second
)

func RetryDelay(attempt int, headers http.Header) time.Duration {
	if d := retryAfterDelay(headers); d > 0 {
		if d > maxBackoff {
			return maxBackoff
		}
		return d
	}

	if attempt <= 0 {
		attempt = 1
	}

	d := baseBackoff * time.Duration(1<<uint(attempt-1))
	if d > maxBackoff {
		d = maxBackoff
	}

	jitter := time.Duration(attempt*50) * time.Millisecond
	if d+jitter > maxBackoff {
		return maxBackoff
	}

	return d + jitter
}

func retryAfterDelay(headers http.Header) time.Duration {
	if headers == nil {
		return 0
	}

	val := headers.Get("Retry-After")
	if val == "" {
		val = headers.Get("retry-after")
	}
	if val == "" {
		return 0
	}

	if secs, err := strconv.Atoi(val); err == nil {
		return time.Duration(secs) * time.Second
	}

	if t, err := http.ParseTime(val); err == nil {
		d := time.Until(t)
		if d > 0 {
			return d
		}
	}

	return 0
}
