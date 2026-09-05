# MSG Desktop — Agent Onboarding Guide

> Baca file ini dulu sebelum menyentuh apapun. Ini repo hybrid: **upstream open-source scraper** + **produk proprietari MSG Desktop** di atasnya. Salah paham soal batas ini = merusak upstream.

## 1. Aplikasi apa ini?

**MSG Desktop** = aplikasi Windows local-first buat prospektor jasa website/aplikasi ke bisnis lokal (kasus utama: tukang jualan website keliling Cirebon).

Alur user: **Discover** (scrape Google Maps beneran) → **Leads** (skor, filter, bulk, hapus) → **Pipeline** (kanban drag-drop, nominal deal) → **Follow-ups** (jadwal, pengingat) → **Outreach** (draft pesan AI + WhatsApp) → **Activity** (audit timeline). Plus: **AI keys multi-akun + fallback**, **auto-update** (GitHub / folder lokal), **Settings** (backup/restore, telemetri opt-in).

Prinsip produk:
- **Tanpa Docker, tanpa server, tanpa Redis/Postgres.** Satu exe + SQLite + SecureStore OS.
- **Jujur ke user.** Tidak ada data mock/synthetic di production. Kalau data habis → bilang habis. Kalau gagal → error jujur + cara keluar. Nol `catch {}` diam, nol angka ngarang.
- **AI opsional.** Semua fitur inti jalan tanpa API key. AI (analisa, outreach) cuma enhancement via chain fallback.

## 2. Peta repo (yang penting)

```
cmd/msgdesktop/main.go      ← entrypoint Wails App + SEMUA bindings JS (satu file, ~1100 baris, ~55 method)
desktop/frontend/src/       ← React+TS+Vite+Fluent UI v9 (7 routes di App.tsx)
desktop/frontend/public/    ← favicon.svg (di-copy ke dist/)
desktop/storage/            ← AppDataDir, Backup/Restore, sqlite (db.go, migrate.go, migrations/)
desktop/securestore/        ← secret OS-backed (DPAPI/file). Secret TIDAK PERNAH di SQLite.
internal/msg/               ← domain produk. JANGAN taruh CRM di gmaps/ (lihat §3).
  acquisition/  Search + ScraperAdapter + directScrapeFallback (Playwright)
  domain/       RawPlaceV1 (kontrak v1, 26 field + Socials)
  ingest/       dedupe place_id>cid>data_id>link + preservasi CRM + skor saat ingest
  leads/        List (filter/sort/paging), Get, stage/DNC/note/tag/followup/deal, delete
  pipeline/     kanban (termasuk stage "new"), follow-ups, MoveStage
  scoring/      CalculateV2 rule-based + evidence + ScoreMissing + Latest
  outreach/     drafts, templates, wa.me/mailto, MarkContacted, DNC enforced
  ai/           KeySlot metadata, Chain fallback+cooldown, provider gemini/openai/custom
  audit/        audit website real via Playwright (SSRF-safe) + rescore
  update/       cek/download/verifikasi-apply dari GitHub ATAU folder lokal
scripts/publish-release.ps1 ← SATU perintah rilis: build → vet → exe+installer → checksum → gh release [+ local dir]
scripts/check-*.mjs         ← gate checks (lihat §7)
.unlazy/                    ← ledger gates per scope (jangan commit; sudah di .gitignore)
```

Yang **bukan** produk (jangan diubah kecuali paham): `gmaps/`, `runner/`, `scraper/`, `web/`, `cmd/gmapssaas/` = upstream. Perubahan Go di sana wajib dicatat di `UPSTREAM_TOUCHES_V2.md`.

## 3. Batas upstream (keras)

- `gmaps.Entry` **dilarang** membawa field CRM. Mapping `gmaps.Entry → domain.RawPlaceV1` hanya di `internal/msg/acquisition/mapper.go`.
- Guard test `internal/msg/guard_test.go` + `TestNoDockerInvocationInProductCode` melarang `docker run gosom/...` di kode produk.
- `alie-core` frozen untuk V0.1. No Redis/Postgres/BullMQ/Nest di runtime Desktop.

## 4. Data model (SQLite, WAL, `%LOCALAPPDATA%\MSG\msg.db`)

Migrasi berurutan di `desktop/storage/sqlite/migrations/` (runner `migrate.go`, idempoten via `schema_migrations`):
- `001_init.sql` — search_runs, source_snapshots, businesses (+source_link, address_text, city), business_contacts (phone/social/email), lead_states (stage, dnc, follow_up, won_value, lost_reason), notes, tags, activities, outreach_drafts/events/templates, opportunity_scores, website_audits, ai_key_slots, saved_searches, settings.
- `002_backfill_city.sql`, `003_ai_key_slots.sql`, `004_outreach_templates.sql`.

Aturan main: secret (API key) → SecureStore key `ai/key/<slotID>`, SQLite cuma metadata + `key_hint` (`****4digit`). Secret tidak boleh muncul di log/export/DB — ada test yang enforce.

## 5. Kontrak Wails bindings (dipakai frontend via `window.go.main.App.*`)

Satu struct `App` di `cmd/msgdesktop/main.go`. Aturan: return **satu object** (jangan multi-return tuple — pernah resolve `null` di JS → "object null is not iterable"). List selalu return envelope `{items, total}` dengan slice non-nil.

Daftar (ringkas): `Search`, `CancelSearch`, `ListLeads`, `GetLead`, `UpdateLeadStage`, `SetDealOutcome`, `SetLeadDNC`, `AddLeadNote`, `AddLeadTag`, `DeleteLead`, `SetLeadFollowUp`, `GetKanbanBoard`, `MovePipelineCard`, `ListFollowUps`, `ListActivities`, `GetScoreDetail`, `AnalyzeLead`, `DraftOutreachAI`, `ListOutreachDrafts`, `OpenWhatsAppURL`, `OpenEmailURL`, `MarkOutreachSent`, `CreateOutreachTemplate`, `ListOutreachTemplates`, `DeleteOutreachTemplate`, `RunWebsiteAudit`, `SaveSearch`, `ListSavedSearches`, `DeleteSavedSearch`, `ListAIKeys`, `AddAIKey`, `UpdateAIKey`, `RotateAIKey`, `DeleteAIKey`, `MoveAIKey`, `TestAIKey`, `GetVersion`, `GetSetting`, `SetSetting`, `SetTelemetryOptIn`, `BackupNow`, `RestoreBackup`, `WeeklyReport`, `CheckForUpdates`, `DownloadAndInstallUpdate`, `DownloadInstallerAndMigrate`, `GetUpdateSource`, `SetUpdateSource`, `GetAppDataDir`.

Catatan penting:
- `Search` bikin cancellable context per-run; `CancelSearch` batalkan. Jangan pernah blokir tanpa jalan keluar.
- `runtime.EventsEmit` **membunuh proses** kalau ctx bukan dari lifecycle Wails → semua emit lewat `a.emit()` yang guard `wailsReady`.
- `ListLeads` Query pakai `json` tags snake_case (frontend kirim lowercase).
- `ingestSink.OnPlace` = Ingest + skor + (opsional, default mati) auto-audit. Setting `auto_audit_websites`.
- `main()` menjalankan `ScoreMissing()` sekali tiap start (backfill skor rows lama) + `applyTelemetrySetting()` (default mati).

## 6. Build, rilis, update (baca sebelum build!)

- **JANGAN `wails build`** untuk produksi: wrapper-nya pernah nge-build `main.go` yang salah (server scraper :8080, bukan app). Build resmi = `scripts/publish-release.ps1 -Version vX.Y.Z [-LocalDir ...]` (frontend → vet → `go build -tags desktop,production` → makensis per-user → checksum → `gh release create`).
- Output exe di `build/out/` (BUKAN `build/bin/`).
- **JANGAN** output/hapus file bernama `MSGDesktop.exe` / `*installer.exe` di `build/bin/` dan `D:\Scrape Data\Rilis-MSG` — Bitdefender mengkarantina nama itu di path tersebut. Kanal lokal pakai nama netral (`msg-app.bin`, `msg-setup.bin`) + `latest.json` nunjuk nama file.
- Ikon exe: `build/windows/icons/*.png` + `icon.ico` → `cmd/msgdesktop/rsrc_windows_amd64.syso` via go-winres; `publish-release.ps1` refresh versi otomatis. NSIS installer: `build/windows/installer/project.nsi`, scope=user + execution level=user (tanpa Admin).
- Update channels: GitHub Releases `moxsenna/msg-desktop` (default) atau folder lokal (setting `update_source`, format `folder:<path>`). Feed folder = `latest.json {version, notes, exe?, installer?}` + file + `.sha256`. Checksum selalu diverifikasi sebelum apply; mismatch = tolak.
- Repo kode produk: `moxsenna/MSG` (remote `msg`). Upstream `origin` (gosom) **jangan di-push**.

## 7. Cara verifikasi kerja (wajib sebelum lapor selesai)

1. `go vet ./cmd/msgdesktop/... ./internal/msg/... ./desktop/...` harus bersih.
2. `go test` paket yang diubah (`-short` untuk skip live-browser; live test Playwright cuma bila perlu, tandai butuh network).
3. `node <skill-dir>/scripts/gate-check.mjs --status .unlazy/<scope>/GATES.md` — semua gate met.
4. Frontend: `npm run build` di `desktop/frontend` (tsc ketat — `Option` multi-children butuh prop `text`, dst).
5. Test perilaku beneran, bukan cuma unit: jalankan binding-nya (test di `cmd/msgdesktop/*_test.go` itu polanya), baca outputnya.
6. Lapor jujur: yang PASS sebutkan buktinya; yang belum ke-test sebutkan eksplisit (mis. proxy live, email E2E). Dilarang menghapus test gagal biar hijau. Dilarang `as any`/`@ts-ignore`.

## 8. Gotcha yang sudah memakan korban (baca!)

- `runtime.EventsEmit` di luar lifecycle Wails = proses mati. Selalu via `a.emit()`.
- Multi-return Go → tuple JS bisa resolve `null`. Selalu envelope object + slice non-nil.
- SQLite WAL: backup WAJIB `PRAGMA wal_checkpoint(TRUNCATE)` dulu (sudah di `storage.Backup`), kalau tidak data terbaru hilang dari zip.
- Kolom nullable (city/website/phone/address/link) WAJIB `sql.NullString` di scan — rows lama bisa NULL → crash `converting NULL to string`.
- `t.Setenv("LOCALAPPDATA", ...)` di test memindahkan JUGA path browser Playwright → test Playwright gagal `Executable doesn't exist`. Set–restore manual di sekitar `securestore.New()` saja.
- Fluent UI `Dropdown` + `Option` multi-children butuh prop `text`, kalau tidak tsc gagal.
- `page.WaitForTimeout` di playwright-go tidak return value — jangan assign.
- `Select-String` PowerShell tidak punya `-Recurse`; heredoc `@' '@` bermasalah kalau konten ada backtick — pakai file terpisah/`Write` tool.
- Jaringan ke `uploads.github.com`/`api.github.com` kadang drop (DNS) — publish script bisa gagal di tengah; cek `gh release view`, upload susulan pakai `--clobber`, lalu undraft.
- `wails.json` `"main"` harus `./cmd/msgdesktop`; `-skipbindings` bikin `wailsjs` basi tapi runtime binding tetap jalan.

## 9. Status terakhir (per v0.5.2, Sept 2026)

Semua loop utama real + ke-test. Sisa jujur: proxy/grid-bbox belum test live (butuh proxy beneran), email E2E nunggu data email asli, code signing butuh beli sertifikat, multi-device di luar desain local-first.
