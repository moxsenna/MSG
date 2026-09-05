package audit_test

import (
	"testing"

	"github.com/gosom/google-maps-scraper/internal/msg/audit"
)

func TestValidateURL_BlocksPrivate(t *testing.T) {
	cases := []struct {
		url     string
		blocked bool
	}{
		{"https://example.com", false},
		{"http://example.com/page", false},
		{"https://google.com", false},
		{"http://localhost/test", true},
		{"https://127.0.0.1/admin", true},
		{"http://10.0.0.1", true},
		{"http://192.168.1.50", true},
		{"http://172.16.5.4", true},
		{"http://[::1]/", true},
		{"http://0.0.0.0", true},
		{"ftp://example.com", true},
		{"", true},
		{"not-a-url", true},
	}
	for _, c := range cases {
		err := audit.ValidateURL(c.url)
		if c.blocked && err == nil {
			t.Errorf("expected blocked %q", c.url)
		}
		if !c.blocked && err != nil {
			t.Errorf("expected allowed %q got %v", c.url, err)
		}
	}
}

func TestValidateURL_BlocksPrivateIPLiteral(t *testing.T) {
	for _, u := range []string{"http://192.168.0.10", "http://10.10.10.10", "http://172.20.10.5"} {
		if err := audit.ValidateURL(u); err == nil {
			t.Errorf("expected block %s", u)
		}
	}
}
