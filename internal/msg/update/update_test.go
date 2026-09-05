package update

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func sha256sum(b []byte) string {
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}

func TestCompareVersion(t *testing.T) {
	cases := []struct{ a, b string; want int }{
		{"v0.1.0", "v0.1.0", 0},
		{"0.1.0", "v0.2.0", -1},
		{"v0.2.0", "v0.1.9", 1},
		{"v0.1.10", "v0.1.9", 1},
		{"v1.0.0", "v0.9.9", 1},
	}
	for _, c := range cases {
		if got := compareVersion(c.a, c.b); got != c.want {
			t.Errorf("compare(%q,%q)=%d want %d", c.a, c.b, got, c.want)
		}
	}
}

func TestCheckNoReleasesHonest(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
	}))
	defer srv.Close()
	_ = srv
	info, err := Check(context.Background(), "nobody/nothing", "v0.1.0")
	if err == nil {
		t.Fatalf("expected error, got %+v", info)
	}
}

func TestDownloadVerifiesHash(t *testing.T) {
	body := "fake-exe-bytes-123"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", fmt.Sprint(len(body)))
		_, _ = w.Write([]byte(body))
	}))
	defer srv.Close()
	handle, err := Download(context.Background(), srv.URL, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := VerifyAndApply(handle, "deadbeef"); err == nil || !strings.Contains(err.Error(), "checksum mismatch") {
		t.Fatalf("expected checksum mismatch, got %v", err)
	}
}

func TestCheckCapturesInstallerAsset(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"tag_name":"v9.9.9","body":"notes","assets":[
			{"name":"MSGDesktop.exe","browser_download_url":"http://x/MSGDesktop.exe"},
			{"name":"MSGDesktop.exe.sha256","browser_download_url":"http://x/MSGDesktop.exe.sha256"},
			{"name":"MSG.Desktop-amd64-installer.exe","browser_download_url":"http://x/installer.exe"}]}`))
	}))
	defer srv.Close()
	info, err := checkFromURL(context.Background(), srv.URL, "v0.1.0")
	if err != nil {
		t.Fatal(err)
	}
	if !info.Available || info.Latest != "v9.9.9" {
		t.Errorf("expected available v9.9.9, got %+v", info)
	}
	if info.InstallerURL != "http://x/installer.exe" || info.InstallerName == "" {
		t.Errorf("installer asset not captured: %+v", info)
	}
	if info.ExeURL == "" || info.Sha256URL == "" {
		t.Errorf("exe/sha assets missing: %+v", info)
	}
}

func TestFolderSourceCheckAndCopy(t *testing.T) {
	dir := t.TempDir()
	exeBytes := []byte("fake-new-exe-bytes")
	if err := os.WriteFile(dir+"/MSGDesktop.exe", exeBytes, 0o644); err != nil {
		t.Fatal(err)
	}
	h := sha256sum(exeBytes)
	if err := os.WriteFile(dir+"/MSGDesktop.exe.sha256", []byte(h+"  MSGDesktop.exe\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dir+"/latest.json", []byte(`{"version":"v9.0.0","notes":"coba lokal"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	info, err := CheckFromSource(context.Background(), "folder:"+dir, "v0.3.1")
	if err != nil {
		t.Fatal(err)
	}
	if !info.Available || info.Latest != "v9.0.0" {
		t.Fatalf("expected available v9.0.0, got %+v", info)
	}
	handle, err := Download(context.Background(), info.ExeURL, nil)
	if err != nil {
		t.Fatal(err)
	}
	sha, err := ExpectedSHA256(context.Background(), info.Sha256URL)
	if err != nil || sha != h {
		t.Fatalf("sha got %q want %q err=%v", sha, h, err)
	}
	if err := VerifyAndApply(handle, "bogus"); err == nil {
		t.Fatal("expected mismatch rejection")
	}
	_, err = CheckFromSource(context.Background(), "folder:"+dir, "v9.0.0")
	if err != nil {
		t.Fatal(err)
	}
	dir2 := t.TempDir()
	if err := os.WriteFile(dir2+"/msg-app.bin", exeBytes, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dir2+"/msg-app.bin.sha256", []byte(h+"  msg-app.bin\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dir2+"/latest.json", []byte(`{"version":"v9.0.1","notes":"x","exe":"msg-app.bin"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	neutral, err := CheckFromSource(context.Background(), "folder:"+dir2, "v9.0.0")
	if err != nil || !neutral.Available || neutral.Latest != "v9.0.1" {
		t.Fatalf("neutral names: %+v err=%v", neutral, err)
	}
	if !strings.HasSuffix(neutral.ExeURL, "msg-app.bin") {
		t.Errorf("expected neutral exe url, got %q", neutral.ExeURL)
	}
	if _, err := CheckFromSource(context.Background(), "folder:"+dir+"/nope", "v0.1.0"); err == nil {
		t.Fatal("expected missing-folder error")
	}
}

func TestExpectedSHA256Parses(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("abc123  MSGDesktop.exe\n"))
	}))
	defer srv.Close()
	got, err := ExpectedSHA256(context.Background(), srv.URL)
	if err != nil || got != "abc123" {
		t.Fatalf("got %q err %v", got, err)
	}
	_ = os.Getenv
}
