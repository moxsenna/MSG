package update

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/minio/selfupdate"
)

const FeedRepo = "moxsenna/msg-desktop"
const ExeAssetName = "MSGDesktop.exe"

var httpClient = &http.Client{Timeout: 30 * time.Second}

// dlClient has no overall timeout: a 30MB binary on a slow link takes
// longer than any fixed client timeout. Cancellation still flows from ctx
// (10-minute deadline set by the caller), and stalls surface as read errors.
var dlClient = &http.Client{Timeout: 0}

type UpdateInfo struct {
	Available     bool   `json:"available"`
	Current       string `json:"current"`
	Latest        string `json:"latest"`
	Notes         string `json:"notes"`
	ExeURL        string `json:"exe_url"`
	Sha256URL     string `json:"sha256_url"`
	InstallerURL  string `json:"installer_url"`
	InstallerName string `json:"installer_name"`
}

type releaseAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

type releaseResp struct {
	TagName string         `json:"tag_name"`
	Body    string         `json:"body"`
	Assets  []releaseAsset `json:"assets"`
}

func feedURL(repo string) string {
	return "https://api.github.com/repos/" + repo + "/releases/latest"
}

func normVersion(v string) string {
	return strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(v), "v"))
}

func compareVersion(a, b string) int {
	pa := strings.Split(normVersion(a), ".")
	pb := strings.Split(normVersion(b), ".")
	for i := 0; i < len(pa) && i < len(pb); i++ {
		na, ea := strconv.Atoi(pa[i])
		nb, eb := strconv.Atoi(pb[i])
		if ea != nil || eb != nil {
			if pa[i] != pb[i] {
				if pa[i] < pb[i] {
					return -1
				}
				return 1
			}
			continue
		}
		if na != nb {
			if na < nb {
				return -1
			}
			return 1
		}
	}
	if len(pa) != len(pb) {
		if len(pa) < len(pb) {
			return -1
		}
		return 1
	}
	return 0
}

func Check(ctx context.Context, repo, current string) (UpdateInfo, error) {
	return checkFromURL(ctx, feedURL(repo), current)
}

func CheckFromSource(ctx context.Context, source, current string) (UpdateInfo, error) {
	if dir, ok := folderSourceDir(source); ok {
		return checkFromFolder(dir, current)
	}
	repo := strings.TrimSpace(source)
	if repo == "" || repo == "github" {
		repo = FeedRepo
	}
	return Check(ctx, repo, current)
}

func folderSourceDir(source string) (string, bool) {
	s := strings.TrimSpace(source)
	if strings.HasPrefix(strings.ToLower(s), "folder:") {
		dir := strings.TrimSpace(s[len("folder:"):])
		if dir == "" {
			return "", false
		}
		return dir, true
	}
	return "", false
}

type folderFeed struct {
	Version   string `json:"version"`
	Notes     string `json:"notes"`
	Exe       string `json:"exe"`
	Installer string `json:"installer"`
}

func checkFromFolder(dir, current string) (UpdateInfo, error) {
	info := UpdateInfo{Current: current}
	raw, err := os.ReadFile(filepath.Join(dir, "latest.json"))
	if err != nil {
		return info, fmt.Errorf("folder rilis belum ada update (latest.json tidak ditemukan di %s)", dir)
	}
	var feed folderFeed
	if err := json.Unmarshal(raw, &feed); err != nil {
		return info, fmt.Errorf("latest.json rusak: %w", err)
	}
	info.Latest = feed.Version
	info.Notes = feed.Notes
	exeName := feed.Exe
	if exeName == "" {
		exeName = ExeAssetName
	}
	info.ExeURL = "file:" + filepath.Join(dir, exeName)
	info.Sha256URL = "file:" + filepath.Join(dir, exeName+".sha256")
	instName := feed.Installer
	if instName == "" {
		if name, err := installerInFolder(dir); err == nil {
			instName = name
		}
	}
	if instName != "" {
		info.InstallerURL = "file:" + filepath.Join(dir, instName)
		info.InstallerName = instName
	}
	if normVersion(current) == "" {
		info.Available = true
		return info, nil
	}
	info.Available = compareVersion(current, feed.Version) < 0
	return info, nil
}

func installerInFolder(dir string) (string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", err
	}
	for _, e := range entries {
		lower := strings.ToLower(e.Name())
		if strings.Contains(lower, "installer") && strings.HasSuffix(lower, ".exe") {
			return e.Name(), nil
		}
	}
	return "", fmt.Errorf("no installer in folder")
}

func checkFromURL(ctx context.Context, url, current string) (UpdateInfo, error) {
	info := UpdateInfo{Current: current}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return info, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	resp, err := httpClient.Do(req)
	if err != nil {
		return info, fmt.Errorf("update check failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == 404 {
		return info, fmt.Errorf("no releases published yet")
	}
	if resp.StatusCode != 200 {
		return info, fmt.Errorf("update check http %d", resp.StatusCode)
	}
	var rel releaseResp
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return info, fmt.Errorf("update feed decode: %w", err)
	}
	info.Latest = rel.TagName
	info.Notes = rel.Body
	for _, a := range rel.Assets {
		switch a.Name {
		case ExeAssetName:
			info.ExeURL = a.BrowserDownloadURL
		case ExeAssetName + ".sha256":
			info.Sha256URL = a.BrowserDownloadURL
		default:
			lower := strings.ToLower(a.Name)
			if strings.Contains(lower, "installer") && strings.HasSuffix(lower, ".exe") {
				info.InstallerURL = a.BrowserDownloadURL
				info.InstallerName = a.Name
			}
		}
	}
	if normVersion(current) == "" {
		info.Available = info.ExeURL != ""
		return info, nil
	}
	info.Available = info.ExeURL != "" && compareVersion(current, rel.TagName) < 0
	return info, nil
}

func ExpectedSHA256(ctx context.Context, shaURL string) (string, error) {
	if path, ok := fileURLPath(shaURL); ok {
		b, err := os.ReadFile(path)
		if err != nil {
			return "", err
		}
		fields := strings.Fields(string(b))
		if len(fields) == 0 {
			return "", fmt.Errorf("empty sha256 file")
		}
		return strings.ToLower(fields[0]), nil
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, shaURL, nil)
	if err != nil {
		return "", err
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("sha256 http %d", resp.StatusCode)
	}
	b, err := io.ReadAll(io.LimitReader(resp.Body, 256))
	if err != nil {
		return "", err
	}
	fields := strings.Fields(string(b))
	if len(fields) == 0 {
		return "", fmt.Errorf("empty sha256 file")
	}
	return strings.ToLower(fields[0]), nil
}

func fileURLPath(url string) (string, bool) {
	if strings.HasPrefix(strings.ToLower(url), "file:") {
		return strings.TrimSpace(url[len("file:"):]), true
	}
	return "", false
}

func Download(ctx context.Context, url string, onProgress func(downloaded, total int64)) (string, error) {
	if path, ok := fileURLPath(url); ok {
		return copyLocal(path, onProgress)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	resp, err := dlClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("download failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("download http %d", resp.StatusCode)
	}
	tmp, err := os.CreateTemp("", "msg-update-*.exe")
	if err != nil {
		return "", err
	}
	defer func() {
		if err != nil {
			os.Remove(tmp.Name())
		}
	}()
	h := sha256.New()
	var done int64
	total := resp.ContentLength
	buf := make([]byte, 128*1024)
	for {
		n, rerr := resp.Body.Read(buf)
		if n > 0 {
			done += int64(n)
			h.Write(buf[:n])
			if _, werr := tmp.Write(buf[:n]); werr != nil {
				tmp.Close()
				err = werr
				return "", err
			}
			if onProgress != nil {
				onProgress(done, total)
			}
		}
		if rerr != nil {
			if rerr == io.EOF {
				break
			}
			tmp.Close()
			err = rerr
			return "", err
		}
	}
	if err := tmp.Close(); err != nil {
		return "", err
	}
	got := hex.EncodeToString(h.Sum(nil))
	return tmp.Name() + "|" + got, nil
}

func copyLocal(src string, onProgress func(downloaded, total int64)) (string, error) {
	in, err := os.Open(src)
	if err != nil {
		return "", fmt.Errorf("buka file update lokal: %w", err)
	}
	defer in.Close()
	st, err := in.Stat()
	if err != nil {
		return "", err
	}
	total := st.Size()
	tmp, err := os.CreateTemp("", "msg-update-*.exe")
	if err != nil {
		return "", err
	}
	defer func() {
		if err != nil {
			os.Remove(tmp.Name())
		}
	}()
	h := sha256.New()
	var done int64
	buf := make([]byte, 128*1024)
	for {
		n, rerr := in.Read(buf)
		if n > 0 {
			done += int64(n)
			h.Write(buf[:n])
			if _, werr := tmp.Write(buf[:n]); werr != nil {
				tmp.Close()
				err = werr
				return "", err
			}
			if onProgress != nil {
				onProgress(done, total)
			}
		}
		if rerr != nil {
			if rerr == io.EOF {
				break
			}
			tmp.Close()
			err = rerr
			return "", err
		}
	}
	if err := tmp.Close(); err != nil {
		return "", err
	}
	return tmp.Name() + "|" + hex.EncodeToString(h.Sum(nil)), nil
}

func VerifyAndApply(fileAndHash, expected string) error {
	parts := strings.SplitN(fileAndHash, "|", 2)
	if len(parts) != 2 {
		return fmt.Errorf("internal: bad download handle")
	}
	path, got := parts[0], parts[1]
	defer os.Remove(path)
	if strings.ToLower(expected) != "" && got != strings.ToLower(expected) {
		return fmt.Errorf("checksum mismatch (possible corrupt/tampered download)")
	}
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	var applyErr error
	for attempt := 0; attempt < 3; attempt++ {
		if _, serr := f.Seek(0, io.SeekStart); serr != nil {
			return serr
		}
		if applyErr = selfupdate.Apply(f, selfupdate.Options{}); applyErr == nil {
			return nil
		}
		if !isPermissionError(applyErr) {
			break
		}
		time.Sleep(2 * time.Second)
	}
	return fmt.Errorf("apply update: %w (aplikasi terinstall di folder yang butuh Admin — pakai Migrasi Installer di bawah, sekali saja)", applyErr)
}

func isPermissionError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "access is denied") || strings.Contains(msg, "permission denied")
}
