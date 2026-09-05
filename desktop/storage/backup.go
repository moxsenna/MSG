package storage

import (
	"archive/zip"
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

// Backup creates a timestamped zip archive in AppDataDir/backups containing msg.db and screenshots.
func Backup(dbPath string) (string, error) {
	appDir, err := AppDataDir()
	if err != nil {
		return "", err
	}
	backupDir := filepath.Join(appDir, "backups")
	if err := os.MkdirAll(backupDir, 0o755); err != nil {
		return "", err
	}

	ts := time.Now().UTC().Format("20060102-150405")
	zipPath := filepath.Join(backupDir, fmt.Sprintf("msg-backup-%s.zip", ts))

	zf, err := os.Create(zipPath)
	if err != nil {
		return "", fmt.Errorf("create zip: %w", err)
	}
	defer zf.Close()

	zw := zip.NewWriter(zf)
	defer zw.Close()

	// 1. Add DB file if exists
	if _, err := os.Stat(dbPath); err == nil {
		if err := addFileToZip(zw, dbPath, "msg.db"); err != nil {
			return "", fmt.Errorf("zip db: %w", err)
		}
	}

	// 2. Add screenshots folder
	screenDir := filepath.Join(appDir, "screenshots")
	_ = filepath.Walk(screenDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		rel, _ := filepath.Rel(appDir, path)
		_ = addFileToZip(zw, path, rel)
		return nil
	})

	return zipPath, nil
}

func addFileToZip(zw *zip.Writer, srcPath, zipName string) error {
	src, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer src.Close()

	w, err := zw.Create(zipName)
	if err != nil {
		return err
	}
	_, err = io.Copy(w, src)
	return err
}

// ExportCSV exports leads query results to a CSV file.
func ExportCSV(db *sql.DB, outPath string) error {
	f, err := os.Create(outPath)
	if err != nil {
		return err
	}
	defer f.Close()

	w := csv.NewWriter(f)
	defer w.Flush()

	// Header
	_ = w.Write([]string{"ID", "Title", "Category", "City", "Website", "Phone", "Rating", "ReviewCount", "Stage", "DNC"})

	rows, err := db.Query(`SELECT b.id, b.title, COALESCE(b.primary_category,''), COALESCE(b.city,''), COALESCE(b.website,''), COALESCE(b.phone,''), b.review_rating, b.review_count, ls.stage, ls.do_not_contact FROM businesses b JOIN lead_states ls ON ls.business_id=b.id ORDER BY b.created_at DESC`)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var id, title, cat, city, web, phone, stage string
		var rating float64
		var count, dnc int
		if err := rows.Scan(&id, &title, &cat, &city, &web, &phone, &rating, &count, &stage, &dnc); err != nil {
			return err
		}
		_ = w.Write([]string{
			id, title, cat, city, web, phone,
			fmt.Sprintf("%.1f", rating),
			fmt.Sprintf("%d", count),
			stage,
			fmt.Sprintf("%t", dnc == 1),
		})
	}
	return rows.Err()
}

// ExportJSON exports leads query results to a JSON file.
func ExportJSON(db *sql.DB, outPath string) error {
	rows, err := db.Query(`SELECT b.id, b.title, COALESCE(b.primary_category,''), COALESCE(b.city,''), COALESCE(b.website,''), COALESCE(b.phone,''), b.review_rating, b.review_count, ls.stage, ls.do_not_contact FROM businesses b JOIN lead_states ls ON ls.business_id=b.id ORDER BY b.created_at DESC`)
	if err != nil {
		return err
	}
	defer rows.Close()

	type ExportItem struct {
		ID          string  `json:"id"`
		Title       string  `json:"title"`
		Category    string  `json:"category"`
		City        string  `json:"city"`
		Website     string  `json:"website"`
		Phone       string  `json:"phone"`
		Rating      float64 `json:"review_rating"`
		ReviewCount int     `json:"review_count"`
		Stage       string  `json:"stage"`
		DNC         bool    `json:"dnc"`
	}

	var list []ExportItem
	for rows.Next() {
		var it ExportItem
		var dnc int
		if err := rows.Scan(&it.ID, &it.Title, &it.Category, &it.City, &it.Website, &it.Phone, &it.Rating, &it.ReviewCount, &it.Stage, &dnc); err != nil {
			return err
		}
		it.DNC = dnc == 1
		list = append(list, it)
	}

	b, err := json.MarshalIndent(list, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(outPath, b, 0o644)
}
