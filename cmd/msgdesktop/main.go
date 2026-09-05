package main

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/runtime"

	"github.com/gosom/google-maps-scraper/desktop/frontend"
	"github.com/gosom/google-maps-scraper/desktop/securestore"
	"github.com/gosom/google-maps-scraper/desktop/storage"
	"github.com/gosom/google-maps-scraper/desktop/storage/sqlite"
	"github.com/gosom/google-maps-scraper/internal/msg/acquisition"
	"github.com/gosom/google-maps-scraper/internal/msg/ai"
	"github.com/gosom/google-maps-scraper/internal/msg/audit"
	"github.com/gosom/google-maps-scraper/internal/msg/domain"
	"github.com/gosom/google-maps-scraper/internal/msg/ingest"
	"github.com/gosom/google-maps-scraper/internal/msg/leads"
	"github.com/gosom/google-maps-scraper/internal/msg/outreach"
	"github.com/gosom/google-maps-scraper/internal/msg/pipeline"
	"github.com/gosom/google-maps-scraper/internal/msg/scoring"
	"github.com/gosom/google-maps-scraper/internal/msg/update"
)

// App is the Wails-bound application struct. Every exported method becomes a JS binding.
type App struct {
	ctx        context.Context
	cancelMu   sync.Mutex
	cancelFn   context.CancelFunc
	wailsReady bool
	store      securestore.SecureStore
	db         *sql.DB
	leads      *leads.Service
	ingest     *ingest.Service
	pipeline   *pipeline.Service
	adapter    *acquisition.Adapter
	scoring    *scoring.Service
	outreach   *outreach.Service
	aiSlots    *ai.SlotStore
	aiChain    *ai.Chain
}

// NewApp creates a new App.
func NewApp() *App { return &App{} }

// Startup is called by Wails runtime.
func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx
	a.wailsReady = true
}

// emit sends a Wails runtime event only when running inside the Wails
// lifecycle. Outside it (unit tests, plain go run) the runtime call would
// kill the process, so events are skipped there.
func (a *App) emit(name string, data ...interface{}) {
	if !a.wailsReady || a.ctx == nil {
		return
	}
	runtime.EventsEmit(a.ctx, name, data...)
}

var appVersion = "0.1.0-dev"

// GetVersion returns the app version (injected via -ldflags -X main.appVersion).
func (a *App) GetVersion() string { return appVersion }

func (a *App) GetSetting(key string) (string, error) {
	if a.db == nil {
		return "", errNotReady("storage not initialized")
	}
	var v string
	if err := a.db.QueryRow(`SELECT value FROM settings WHERE key=?`, key).Scan(&v); err != nil {
		return "", nil
	}
	return v, nil
}

func (a *App) SetSetting(key, value string) error {
	if a.db == nil {
		return errNotReady("storage not initialized")
	}
	_, err := a.db.Exec(`INSERT INTO settings(key, value) VALUES(?,?) ON CONFLICT(key) DO UPDATE SET value=excluded.value, updated_at=strftime('%Y-%m-%dT%H:%M:%SZ','now')`, key, value)
	return err
}

func (a *App) updateSource() string {
	if a.db == nil {
		return ""
	}
	var v string
	if err := a.db.QueryRow(`SELECT value FROM settings WHERE key='update_source'`).Scan(&v); err != nil {
		return ""
	}
	return v
}

func (a *App) GetUpdateSource() string {
	if s := a.updateSource(); s != "" {
		return s
	}
	return "github:" + update.FeedRepo
}

func (a *App) SetUpdateSource(source string) error {
	source = strings.TrimSpace(source)
	if source == "" || strings.EqualFold(source, "github") {
		source = "github:" + update.FeedRepo
	}
	if !strings.HasPrefix(strings.ToLower(source), "github:") && !strings.HasPrefix(strings.ToLower(source), "folder:") {
		return fmt.Errorf("sumber harus github atau folder:<path>")
	}
	if strings.HasPrefix(strings.ToLower(source), "folder:") {
		dir := strings.TrimSpace(source[len("folder:"):])
		if st, err := os.Stat(dir); err != nil || !st.IsDir() {
			return fmt.Errorf("folder tidak ditemukan: %s", dir)
		}
	}
	return a.SetSetting("update_source", source)
}

func (a *App) CheckForUpdates() (update.UpdateInfo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	return update.CheckFromSource(ctx, a.updateSource(), appVersion)
}

func (a *App) DownloadAndInstallUpdate() (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	info, err := update.CheckFromSource(ctx, a.updateSource(), appVersion)
	if err != nil {
		return "", err
	}
	if !info.Available {
		return "Sudah versi terbaru (" + appVersion + ").", nil
	}
	a.emit("update:progress", map[string]any{"stage": "downloading", "done": 0, "total": -1})
	handle, err := update.Download(ctx, info.ExeURL, func(done, total int64) {
		a.emit("update:progress", map[string]any{"stage": "downloading", "done": done, "total": total})
	})
	if err != nil {
		return "", err
	}
	expected := ""
	if info.Sha256URL != "" {
		a.emit("update:progress", map[string]any{"stage": "verifying"})
		expected, err = update.ExpectedSHA256(ctx, info.Sha256URL)
		if err != nil {
			return "", fmt.Errorf("verify failed: %w", err)
		}
	}
	a.emit("update:progress", map[string]any{"stage": "applying"})
	if err := update.VerifyAndApply(handle, expected); err != nil {
		return "", err
	}
	a.emit("update:progress", map[string]any{"stage": "done"})
	return "Update " + info.Latest + " terpasang. Tutup dan buka ulang aplikasi untuk memakai versi baru.", nil
}

// GetAppDataDir returns the OS app data directory.
func (a *App) GetAppDataDir() (string, error) { return storage.AppDataDir() }

// Search runs a discovery search and ingests RawPlaceV1 into SQLite.
func (a *App) Search(req acquisition.SearchRequest) (string, error) {
	if a.db == nil || a.ingest == nil || a.adapter == nil {
		return "", errNotReady("storage not initialized")
	}
	runID := uuid.NewString()
	_, err := a.db.Exec(`INSERT INTO search_runs(id, query, location_text, goal, status, started_at) VALUES(?,?,?,?, 'running', datetime('now'))`,
		runID, req.Query, req.LocationText, req.Goal)
	if err != nil {
		return "", err
	}
	sink := &ingestSink{svc: a.ingest, scoring: a.scoring, runID: runID}
	ctx := a.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, cancel := context.WithCancel(ctx)
	a.cancelMu.Lock()
	a.cancelFn = cancel
	a.cancelMu.Unlock()
	defer func() {
		a.cancelMu.Lock()
		a.cancelFn = nil
		a.cancelMu.Unlock()
	}()
	progress := &emitProgress{app: a, runID: runID, sink: sink}
	a.emit("search:started", map[string]any{"runId": runID, "query": req.Query})
	err = a.adapter.Search(ctx, req, sink, progress)
	if err == nil {
		a.emit("search:completed", map[string]any{"runId": runID, "found": sink.count})
	} else {
		a.emit("search:failed", map[string]any{"runId": runID, "error": err.Error()})
	}
	status := "completed"
	if err != nil {
		status = "failed"
	}
	_, _ = a.db.Exec(`UPDATE search_runs SET status=?, finished_at=datetime('now'), result_count=? WHERE id=?`, status, sink.count, runID)
	if err != nil {
		return runID, err
	}
	return runID, nil
}

// LeadListResult is a single-object envelope so the Wails JS binding
// always resolves to an object (never a null multi-return tuple).
type LeadListResult struct {
	Items []leads.LeadListItem `json:"items"`
	Total int                  `json:"total"`
}

func (a *App) CancelSearch() {
	a.cancelMu.Lock()
	defer a.cancelMu.Unlock()
	if a.cancelFn != nil {
		a.cancelFn()
	}
}

func (a *App) BackupNow() (string, error) {
	dbPath, err := storage.DBPath()
	if err != nil {
		return "", err
	}
	return storage.Backup(dbPath)
}

// ListLeads returns paginated leads server-side (50k-capable).
func (a *App) ListLeads(q leads.Query) (LeadListResult, error) {
	if a.leads == nil {
		return LeadListResult{Items: []leads.LeadListItem{}, Total: 0}, errNotReady("storage not initialized")
	}
	items, total, err := a.leads.List(q)
	if err != nil {
		return LeadListResult{Items: []leads.LeadListItem{}, Total: 0}, err
	}
	if items == nil {
		items = []leads.LeadListItem{}
	}
	return LeadListResult{Items: items, Total: total}, nil
}

// GetLead returns a single lead detail with CRM state.
func (a *App) GetLead(id string) (*leads.Detail, error) {
	if a.leads == nil {
		return nil, errNotReady("storage not initialized")
	}
	return a.leads.Get(id)
}

func (a *App) SetDealOutcome(id, stage string, value float64, reason string) error {
	if a.leads == nil {
		return errNotReady("storage not initialized")
	}
	return a.leads.SetDealOutcome(id, stage, value, reason)
}

// DeleteLead removes a business and all its CRM data (cascades via FK).
func (a *App) DeleteLead(id string) error {
	if a.db == nil {
		return errNotReady("storage not initialized")
	}
	res, err := a.db.Exec(`DELETE FROM businesses WHERE id=?`, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("lead %q not found", id)
	}
	return nil
}

// UpdateLeadStage moves a lead to a new pipeline stage.
func (a *App) UpdateLeadStage(id, stage string) error {
	if a.leads == nil {
		return errNotReady("storage not initialized")
	}
	return a.leads.UpdateStage(id, stage)
}

// SetLeadDNC toggles Do Not Contact.
func (a *App) SetLeadDNC(id string, dnc bool) error {
	if a.leads == nil {
		return errNotReady("storage not initialized")
	}
	return a.leads.SetDNC(id, dnc)
}

// AddLeadNote appends a note.
func (a *App) AddLeadNote(id, body string) error {
	if a.leads == nil {
		return errNotReady("storage not initialized")
	}
	return a.leads.AddNote(id, body)
}

// AddLeadTag attaches a tag.
func (a *App) AddLeadTag(id, tag string) error {
	if a.leads == nil {
		return errNotReady("storage not initialized")
	}
	return a.leads.AddTag(id, tag)
}

// GetKanbanBoard returns the pipeline board grouped by stage.
func (a *App) GetKanbanBoard() (map[string][]pipeline.StageCard, error) {
	if a.pipeline == nil {
		return nil, errNotReady("storage not initialized")
	}
	return a.pipeline.GetKanbanBoard()
}

// MovePipelineCard moves a card to the next stage transactionally.
func (a *App) MovePipelineCard(businessID, newStage string) error {
	if a.pipeline == nil {
		return errNotReady("storage not initialized")
	}
	return a.pipeline.MoveStage(businessID, newStage)
}

// ListFollowUps returns overdue and upcoming follow-ups.
func (a *App) ListFollowUps() ([]pipeline.FollowUpItem, error) {
	if a.pipeline == nil {
		return nil, errNotReady("storage not initialized")
	}
	items, err := a.pipeline.ListFollowUps()
	if err != nil {
		return nil, err
	}
	if items == nil {
		return []pipeline.FollowUpItem{}, nil
	}
	return items, nil
}

// SetLeadFollowUp sets the next follow-up timestamp for a lead.
func (a *App) SetLeadFollowUp(id string, followUpAt string) error {
	if a.leads == nil {
		return errNotReady("storage not initialized")
	}
	if followUpAt == "" {
		return a.leads.SetFollowUp(id, nil)
	}
	for _, layout := range []string{time.RFC3339, "2006-01-02T15:04", "2006-01-02 15:04", "2006-01-02"} {
		if t, err := time.Parse(layout, followUpAt); err == nil {
			return a.leads.SetFollowUp(id, &t)
		}
	}
	return fmt.Errorf("invalid follow-up time %q, use YYYY-MM-DD or RFC3339", followUpAt)
}

// ActivityItem is a real audit timeline row from SQLite.
type ActivityItem struct {
	ID         string `json:"id"`
	BusinessID string `json:"business_id"`
	Title      string `json:"title"`
	Type       string `json:"type"`
	Detail     string `json:"detail"`
	At         string `json:"at"`
}

// ListActivities returns the real audit timeline (discoveries, stage changes, notes).
func (a *App) ListActivities(limit int) ([]ActivityItem, error) {
	if a.db == nil {
		return nil, errNotReady("storage not initialized")
	}
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	rows, err := a.db.Query(`
		SELECT ac.id, COALESCE(ac.business_id,''), COALESCE(b.title,''), ac.type, COALESCE(ac.payload_json,'{}'), ac.created_at
		FROM activities ac LEFT JOIN businesses b ON b.id=ac.business_id
		ORDER BY ac.created_at DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []ActivityItem{}
	for rows.Next() {
		var it ActivityItem
		var payload string
		if err := rows.Scan(&it.ID, &it.BusinessID, &it.Title, &it.Type, &payload, &it.At); err != nil {
			return nil, err
		}
		it.Detail = activityDetail(it.Type, it.Title, payload)
		if it.Title == "" {
			it.Title = "Pencarian"
		}
		items = append(items, it)
	}
	return items, rows.Err()
}

func activityDetail(typ, title, payload string) string {
	switch typ {
	case "discovered":
		if title != "" {
			return "Prospek baru diimpor ke SQLite"
		}
		return "Run pencarian selesai"
	case "rescraped":
		return "Data sumber diperbarui, CRM dipertahankan"
	case "stage_changed":
		return "Tahap CRM dipindahkan"
	case "note_added":
		return "Catatan ditambahkan"
	default:
		if title != "" {
			return title
		}
		return typ
	}
}

type errNotReady string

func (e errNotReady) Error() string { return string(e) }

type ingestSink struct {
	svc     *ingest.Service
	scoring *scoring.Service
	runID   string
	count   int
}

func (s *ingestSink) OnPlace(ctx context.Context, place domain.RawPlaceV1) error {
	businessID, _, err := s.svc.Ingest(ctx, s.runID, place)
	if err != nil {
		return err
	}
	s.count++
	if s.scoring != nil && businessID != "" {
		res := scoring.CalculateV2(scoring.InputsFromPlace(
			place.Title, place.Category, place.Website, place.Phone,
			place.Status, place.ReviewRating, place.ReviewCount))
		_, _ = s.scoring.Persist(res, businessID)
	}
	return nil
}

type emitProgress struct {
	app   *App
	runID string
	sink  *ingestSink
}

func (p *emitProgress) OnProgress(stage string, found, processed int) {
	count := 0
	if p.sink != nil {
		count = p.sink.count
	}
	p.app.emit("search:progress", map[string]any{"runId": p.runID, "stage": stage, "found": found, "processed": processed, "count": count})
}

func (p *emitProgress) OnError(code, message string) {
	p.app.emit("search:error", map[string]any{"runId": p.runID, "code": code, "message": message})
}

func (a *App) aiKey(slotID string) (string, error) {
	if a.store == nil {
		return "", errNotReady("secure store not initialized")
	}
	return a.store.Get("ai/key/" + slotID)
}

func (a *App) ListAIKeys() ([]ai.KeySlot, error) {
	if a.aiSlots == nil {
		return []ai.KeySlot{}, errNotReady("storage not initialized")
	}
	slots, err := a.aiSlots.List()
	if err != nil {
		return []ai.KeySlot{}, err
	}
	if slots == nil {
		return []ai.KeySlot{}, nil
	}
	return slots, nil
}

type AddAIKeyRequest struct {
	Provider string `json:"provider"`
	Label    string `json:"label"`
	Key      string `json:"key"`
	BaseURL  string `json:"base_url"`
	Model    string `json:"model"`
}

func (a *App) AddAIKey(req AddAIKeyRequest) (ai.KeySlot, error) {
	if a.aiSlots == nil {
		return ai.KeySlot{}, errNotReady("storage not initialized")
	}
	if len(req.Key) < 8 {
		return ai.KeySlot{}, fmt.Errorf("key too short (min 8 chars)")
	}
	slot, err := a.aiSlots.Create(req.Provider, req.Label, ai.KeyHint(req.Key), req.BaseURL, req.Model)
	if err != nil {
		return ai.KeySlot{}, err
	}
	if a.store == nil {
		return ai.KeySlot{}, errNotReady("secure store not initialized")
	}
	if err := a.store.Set("ai/key/"+slot.ID, req.Key); err != nil {
		_ = a.aiSlots.Delete(slot.ID)
		return ai.KeySlot{}, err
	}
	return slot, nil
}

func (a *App) UpdateAIKey(id, label string, enabled bool, baseURL, model string) error {
	if a.aiSlots == nil {
		return errNotReady("storage not initialized")
	}
	return a.aiSlots.Update(id, label, &enabled, baseURL, model)
}

func (a *App) RotateAIKey(id, newKey string) error {
	if a.aiSlots == nil {
		return errNotReady("storage not initialized")
	}
	if len(newKey) < 8 {
		return fmt.Errorf("key too short (min 8 chars)")
	}
	slots, err := a.aiSlots.List()
	if err != nil {
		return err
	}
	label := ""
	for _, sl := range slots {
		if sl.ID == id {
			label = sl.Label
			break
		}
	}
	if label == "" {
		return fmt.Errorf("slot %q not found", id)
	}
	if a.store == nil {
		return errNotReady("secure store not initialized")
	}
	if err := a.store.Set("ai/key/"+id, newKey); err != nil {
		return err
	}
	return a.aiSlots.Update(id, label, nil, "", "")
}

func (a *App) DeleteAIKey(id string) error {
	if a.aiSlots == nil {
		return errNotReady("storage not initialized")
	}
	if err := a.aiSlots.Delete(id); err != nil {
		return err
	}
	if a.store != nil {
		_ = a.store.Delete("ai/key/" + id)
	}
	return nil
}

func (a *App) MoveAIKey(id string, direction int) error {
	if a.aiSlots == nil {
		return errNotReady("storage not initialized")
	}
	if direction != -1 && direction != 1 {
		return fmt.Errorf("direction must be -1 or 1")
	}
	return a.aiSlots.Move(id, direction)
}

func (a *App) TestAIKey(id string) (string, error) {
	if a.aiSlots == nil || a.aiChain == nil {
		return "", errNotReady("storage not initialized")
	}
	slots, err := a.aiSlots.List()
	if err != nil {
		return "", err
	}
	var target *ai.KeySlot
	for i, sl := range slots {
		if sl.ID == id {
			target = &slots[i]
			break
		}
	}
	if target == nil {
		return "", fmt.Errorf("slot %q not found", id)
	}
	key, err := a.aiKey(id)
	if err != nil {
		return "", fmt.Errorf("cannot read key: %w", err)
	}
	p, err := ai.BuildProvider(*target, key)
	if err != nil {
		return "", err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
	defer cancel()
	text, err := p.DraftOutreach(ctx, "Toko Contoh", "retail", "paket website", "whatsapp", "friendly")
	if err != nil {
		return "", err
	}
	if len(text) > 200 {
		text = text[:200] + "..."
	}
	return fmt.Sprintf("OK via %s (%s): %s", target.Label, target.Provider, text), nil
}

type AnalyzeLeadResult struct {
	Summary        string   `json:"summary"`
	OpportunityType string  `json:"opportunity_type"`
	SalesAngle     string   `json:"sales_angle"`
	SuggestedOffer string   `json:"suggested_offer"`
	Reasons        []string `json:"reasons"`
	Confidence     float64  `json:"confidence"`
	UsedLabel      string   `json:"used_label"`
	UsedProvider   string   `json:"used_provider"`
}

func (a *App) AnalyzeLead(id string) (AnalyzeLeadResult, error) {
	if a.leads == nil || a.aiSlots == nil || a.aiChain == nil {
		return AnalyzeLeadResult{}, errNotReady("storage not initialized")
	}
	detail, err := a.leads.Get(id)
	if err != nil {
		return AnalyzeLeadResult{}, err
	}
	slots, err := a.aiSlots.List()
	if err != nil {
		return AnalyzeLeadResult{}, err
	}
	websiteInfo := detail.Website
	if detail.ReviewCount > 0 {
		websiteInfo += fmt.Sprintf(" | rating %.1f (%d ulasan)", detail.ReviewRating, detail.ReviewCount)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	analysis, usedID, err := a.aiChain.AnalyzeOpportunity(ctx, slots, detail.Title, detail.Category, detail.City, websiteInfo)
	if err != nil {
		return AnalyzeLeadResult{}, err
	}
	var usedLabel, usedProvider, usedModel string
	for _, sl := range slots {
		if sl.ID == usedID {
			usedLabel, usedProvider, usedModel = sl.Label, sl.Provider, sl.Model
			break
		}
	}
	sum := sha256.Sum256([]byte(detail.Title + "|" + detail.Category + "|" + detail.City + "|" + websiteInfo))
	inputHash := fmt.Sprintf("%x", sum[:8])
	reasonsJSON, _ := json.Marshal(analysis.Reasons)
	if reasonsJSON == nil {
		reasonsJSON = []byte("[]")
	}
	_, err = a.db.Exec(`INSERT INTO ai_analyses(id, business_id, provider, model, prompt_version, input_hash, summary, opportunity_type, sales_angle, suggested_offer, reasons_json, confidence) VALUES(?,?,?,?,?,?,?,?,?,?,?,?)`,
		uuid.NewString(), id, usedProvider, usedModel, "v1", inputHash,
		analysis.Summary, analysis.OpportunityType, analysis.SalesAngle, analysis.SuggestedOffer,
		string(reasonsJSON), analysis.Confidence)
	if err != nil {
		return AnalyzeLeadResult{}, err
	}
	return AnalyzeLeadResult{
		Summary: analysis.Summary, OpportunityType: analysis.OpportunityType,
		SalesAngle: analysis.SalesAngle, SuggestedOffer: analysis.SuggestedOffer,
		Reasons: analysis.Reasons, Confidence: analysis.Confidence,
		UsedLabel: usedLabel, UsedProvider: usedProvider,
	}, nil
}

type DraftOutreachResult struct {
	DraftID      string `json:"draft_id"`
	Body         string `json:"body"`
	UsedLabel    string `json:"used_label"`
	UsedProvider string `json:"used_provider"`
}

func (a *App) DraftOutreachAI(businessID, channel, tone, offer string) (DraftOutreachResult, error) {
	if a.leads == nil || a.aiSlots == nil || a.aiChain == nil || a.outreach == nil {
		return DraftOutreachResult{}, errNotReady("storage not initialized")
	}
	if channel != "whatsapp" && channel != "email" {
		return DraftOutreachResult{}, fmt.Errorf("invalid channel %q (use whatsapp|email)", channel)
	}
	if tone == "" {
		tone = "friendly"
	}
	detail, err := a.leads.Get(businessID)
	if err != nil {
		return DraftOutreachResult{}, err
	}
	if detail.DNC {
		return DraftOutreachResult{}, outreach.ErrDNCBlocked
	}
	if offer == "" {
		offer = "paket website profesional + setup WhatsApp order"
	}
	slots, err := a.aiSlots.List()
	if err != nil {
		return DraftOutreachResult{}, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	text, usedID, err := a.aiChain.DraftOutreach(ctx, slots, detail.Title, detail.Category, offer, channel, tone)
	if err != nil {
		return DraftOutreachResult{}, err
	}
	draft, err := a.outreach.CreateDraft(businessID, channel, tone, text)
	if err != nil {
		return DraftOutreachResult{}, err
	}
	var usedLabel, usedProvider string
	for _, sl := range slots {
		if sl.ID == usedID {
			usedLabel, usedProvider = sl.Label, sl.Provider
			break
		}
	}
	return DraftOutreachResult{DraftID: draft.ID, Body: draft.Body, UsedLabel: usedLabel, UsedProvider: usedProvider}, nil
}

func (a *App) ListOutreachDrafts(businessID string) ([]outreach.Draft, error) {
	if a.outreach == nil {
		return []outreach.Draft{}, errNotReady("storage not initialized")
	}
	drafts, err := a.outreach.ListDrafts(businessID)
	if err != nil {
		return []outreach.Draft{}, err
	}
	if drafts == nil {
		return []outreach.Draft{}, nil
	}
	return drafts, nil
}

func (a *App) OpenWhatsAppURL(businessID, draftID string) (string, error) {
	if a.outreach == nil {
		return "", errNotReady("storage not initialized")
	}
	return a.outreach.OpenWhatsApp(businessID, draftID)
}

func (a *App) CreateOutreachTemplate(name, channel, tone, body string) (outreach.Template, error) {
	if a.outreach == nil {
		return outreach.Template{}, errNotReady("storage not initialized")
	}
	t, err := a.outreach.CreateTemplate(name, channel, tone, body)
	if err != nil {
		return outreach.Template{}, err
	}
	return *t, nil
}

func (a *App) ListOutreachTemplates() ([]outreach.Template, error) {
	if a.outreach == nil {
		return []outreach.Template{}, errNotReady("storage not initialized")
	}
	tpls, err := a.outreach.ListTemplates()
	if err != nil {
		return []outreach.Template{}, err
	}
	if tpls == nil {
		return []outreach.Template{}, nil
	}
	return tpls, nil
}

func (a *App) DeleteOutreachTemplate(id string) error {
	if a.outreach == nil {
		return errNotReady("storage not initialized")
	}
	return a.outreach.DeleteTemplate(id)
}

func (a *App) OpenEmailURL(businessID, draftID string) (string, error) {
	if a.outreach == nil {
		return "", errNotReady("storage not initialized")
	}
	return a.outreach.OpenEmail(businessID, draftID)
}

func (a *App) MarkOutreachSent(businessID, draftID, channel string) error {
	if a.outreach == nil {
		return errNotReady("storage not initialized")
	}
	if channel == "" {
		channel = "whatsapp"
	}
	return a.outreach.MarkContacted(businessID, draftID, channel)
}

type WebsiteAuditResult struct {
	Reachable    bool     `json:"reachable"`
	HTTPS        bool     `json:"https"`
	HasContact   bool     `json:"has_contact"`
	HasWhatsApp  bool     `json:"has_whatsapp"`
	HasForm      bool     `json:"has_form"`
	Title        string   `json:"title"`
	FinalURL     string   `json:"final_url"`
	ErrorCode    string   `json:"error_code"`
	NewScore     int      `json:"new_score"`
	SocialLinks  []string `json:"social_links"`
}

func auditRun(ctx context.Context, db *sql.DB, businessID, website string) (*audit.Result, error) {
	return audit.New(db).Run(ctx, businessID, website)
}

func (a *App) RunWebsiteAudit(businessID string) (WebsiteAuditResult, error) {
	if a.leads == nil {
		return WebsiteAuditResult{}, errNotReady("storage not initialized")
	}
	detail, err := a.leads.Get(businessID)
	if err != nil {
		return WebsiteAuditResult{}, err
	}
	if strings.TrimSpace(detail.Website) == "" {
		return WebsiteAuditResult{}, fmt.Errorf("no website to audit (ini justru peluang utama: tawarkan pembuatan website)")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	res, err := auditRun(ctx, a.db, businessID, detail.Website)
	if err != nil {
		return WebsiteAuditResult{}, err
	}
	newScore := 0
	if a.scoring != nil {
		in := scoring.InputsFromPlace(detail.Title, detail.Category, detail.Website, detail.Phone, "", detail.ReviewRating, detail.ReviewCount)
		in.IsHTTPS = res.Evidence.HTTPS
		hasMeta := res.Evidence.MetaDesc != ""
		in.HasMeta = &hasMeta
		in.HasContact = res.Evidence.HasContactInfo
		in.HasBookingSignal = res.Evidence.HasBooking
		if r, err := a.scoring.Persist(scoring.CalculateV2(in), businessID); err == nil {
			newScore = r.OverallScore
		}
	}
	deref := func(b *bool) bool { return b != nil && *b }
	out := WebsiteAuditResult{
		Reachable: deref(res.Evidence.Reachable), HTTPS: deref(res.Evidence.HTTPS),
		HasContact: deref(res.Evidence.HasContactInfo),
		HasWhatsApp: deref(res.Evidence.HasWhatsApp), HasForm: deref(res.Evidence.HasForm),
		Title: res.Evidence.Title, FinalURL: res.Evidence.FinalURL,
		ErrorCode: res.Evidence.ErrorCode, NewScore: newScore, SocialLinks: res.Evidence.SocialLinks,
	}
	return out, nil
}

func (a *App) DownloadInstallerAndMigrate() (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	info, err := update.CheckFromSource(ctx, a.updateSource(), appVersion)
	if err != nil {
		return "", err
	}
	if info.InstallerURL == "" {
		return "", fmt.Errorf("rilis %s tidak menyertakan installer", info.Latest)
	}
	a.emit("update:progress", map[string]any{"stage": "downloading-installer", "done": 0, "total": -1})
	handle, err := update.Download(ctx, info.InstallerURL, func(done, total int64) {
		a.emit("update:progress", map[string]any{"stage": "downloading-installer", "done": done, "total": total})
	})
	if err != nil {
		return "", err
	}
	parts := strings.SplitN(handle, "|", 2)
	tmpInstaller := parts[0]
	name := info.InstallerName
	if name == "" {
		name = "MSGDesktop-installer.exe"
	}
	dst := filepath.Join(os.TempDir(), name)
	_ = os.Remove(dst)
	if err := os.Rename(tmpInstaller, dst); err != nil {
		return "", err
	}
	if err := exec.Command(dst).Start(); err != nil {
		return "", fmt.Errorf("installer terunduh di %s tapi gagal dijalankan: %w", dst, err)
	}
	if a.wailsReady && a.ctx != nil {
		go func() {
			time.Sleep(2 * time.Second)
			runtime.Quit(a.ctx)
		}()
	}
	return "Installer " + info.Latest + " dijalankan. Aplikasi akan tertutup — lanjutkan instalasi (pilih lokasi default per-user), data SQLite aman.", nil
}

type SavedSearch struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Query        string `json:"query"`
	LocationText string `json:"location_text"`
	Config       string `json:"config"`
	CreatedAt    string `json:"created_at"`
}

func (a *App) SaveSearch(name, query, location, config string) (SavedSearch, error) {
	if a.db == nil {
		return SavedSearch{}, errNotReady("storage not initialized")
	}
	if strings.TrimSpace(name) == "" || strings.TrimSpace(query) == "" {
		return SavedSearch{}, fmt.Errorf("nama dan query wajib diisi")
	}
	id := uuid.NewString()
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := a.db.Exec(`INSERT INTO saved_searches(id, name, query, location_text, config_json, created_at, updated_at) VALUES(?,?,?,?,?,?,?)`,
		id, strings.TrimSpace(name), strings.TrimSpace(query), strings.TrimSpace(location), config, now, now)
	if err != nil {
		return SavedSearch{}, err
	}
	return SavedSearch{ID: id, Name: strings.TrimSpace(name), Query: strings.TrimSpace(query), LocationText: strings.TrimSpace(location), Config: config, CreatedAt: now}, nil
}

func (a *App) ListSavedSearches() ([]SavedSearch, error) {
	if a.db == nil {
		return []SavedSearch{}, errNotReady("storage not initialized")
	}
	rows, err := a.db.Query(`SELECT id, name, query, location_text, config_json, created_at FROM saved_searches ORDER BY created_at DESC`)
	if err != nil {
		return []SavedSearch{}, err
	}
	defer rows.Close()
	out := []SavedSearch{}
	for rows.Next() {
		var s SavedSearch
		if err := rows.Scan(&s.ID, &s.Name, &s.Query, &s.LocationText, &s.Config, &s.CreatedAt); err != nil {
			return []SavedSearch{}, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func (a *App) DeleteSavedSearch(id string) error {
	if a.db == nil {
		return errNotReady("storage not initialized")
	}
	res, err := a.db.Exec(`DELETE FROM saved_searches WHERE id=?`, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("saved search not found")
	}
	return nil
}

type ScoreEvidenceItem struct {
	Label  string `json:"label"`
	Points int    `json:"points"`
	Detail string `json:"detail"`
}

type ScoreDetailResult struct {
	Overall    int                 `json:"overall"`
	Website    int                 `json:"website"`
	App        int                 `json:"app"`
	Priority   string              `json:"priority"`
	Confidence float64             `json:"confidence"`
	Evidence   []ScoreEvidenceItem `json:"evidence"`
}

func (a *App) GetScoreDetail(businessID string) (ScoreDetailResult, error) {
	if a.scoring == nil {
		return ScoreDetailResult{}, errNotReady("storage not initialized")
	}
	r, err := a.scoring.Latest(businessID)
	if err != nil {
		return ScoreDetailResult{}, err
	}
	out := ScoreDetailResult{
		Overall: r.OverallScore, Website: r.WebsiteScore, App: r.AppScore,
		Priority: r.Priority, Confidence: r.Confidence, Evidence: []ScoreEvidenceItem{},
	}
	for _, e := range r.Evidence {
		out.Evidence = append(out.Evidence, ScoreEvidenceItem{Label: e.Label, Points: e.Points, Detail: e.Detail})
	}
	return out, nil
}

type WeeklyReport struct {
	NewLeads      int     `json:"new_leads"`
	Contacted     int     `json:"contacted"`
	Replied       int     `json:"replied"`
	Won           int     `json:"won"`
	WonValue      float64 `json:"won_value"`
	Lost          int     `json:"lost"`
	Overdue       int     `json:"overdue"`
	SearchRuns    int     `json:"search_runs"`
	Activities    int     `json:"activities"`
}

func (a *App) WeeklyReport() (WeeklyReport, error) {
	if a.db == nil {
		return WeeklyReport{}, errNotReady("storage not initialized")
	}
	var r WeeklyReport
	week := `datetime('now','-7 days')`
	_ = a.db.QueryRow(`SELECT COUNT(*) FROM businesses WHERE created_at>=`+week).Scan(&r.NewLeads)
	_ = a.db.QueryRow(`SELECT COUNT(*) FROM lead_states WHERE stage='contacted' AND updated_at>=`+week).Scan(&r.Contacted)
	_ = a.db.QueryRow(`SELECT COUNT(*) FROM lead_states WHERE stage='replied' AND updated_at>=`+week).Scan(&r.Replied)
	_ = a.db.QueryRow(`SELECT COUNT(*) FROM lead_states WHERE stage='won' AND updated_at>=`+week).Scan(&r.Won)
	_ = a.db.QueryRow(`SELECT COALESCE(SUM(won_value),0) FROM lead_states WHERE stage='won' AND updated_at>=`+week).Scan(&r.WonValue)
	_ = a.db.QueryRow(`SELECT COUNT(*) FROM lead_states WHERE stage='lost' AND updated_at>=`+week).Scan(&r.Lost)
	_ = a.db.QueryRow(`SELECT COUNT(*) FROM lead_states WHERE next_follow_up_at IS NOT NULL AND next_follow_up_at!='' AND next_follow_up_at<strftime('%Y-%m-%dT%H:%M:%SZ','now')`).Scan(&r.Overdue)
	_ = a.db.QueryRow(`SELECT COUNT(*) FROM search_runs WHERE started_at>=`+week).Scan(&r.SearchRuns)
	_ = a.db.QueryRow(`SELECT COUNT(*) FROM activities WHERE created_at>=`+week).Scan(&r.Activities)
	return r, nil
}

func main() {
	dir, err := storage.AppDataDir()
	if err != nil {
		log.Fatalf("appdata: %v", err)
	}
	dbPath, err := storage.DBPath()
	if err != nil {
		log.Fatalf("dbpath: %v", err)
	}
	db, err := sqlite.Open(dbPath)
	if err != nil {
		log.Fatalf("open db %s: %v", dbPath, err)
	}
	if err := sqlite.Migrate(db); err != nil {
		log.Fatalf("migrate: %v", err)
	}
	store, _ := securestore.New()
	app := NewApp()
	app.db = db
	app.store = store
	app.leads = leads.New(db)
	app.ingest = ingest.New(db)
	app.pipeline = pipeline.New(db)
	app.adapter = acquisition.NewAdapter()
	app.scoring = scoring.New(db)
	app.outreach = outreach.New(db)
	if n, err := app.scoring.ScoreMissing(); err == nil && n > 0 {
		log.Printf("scored %d businesses missing scores", n)
	}
	app.aiSlots = ai.NewSlotStore(db)
	app.aiChain = ai.NewChain(func(slot ai.KeySlot) (ai.Provider, error) {
		key, err := app.aiKey(slot.ID)
		if err != nil {
			return nil, err
		}
		return ai.BuildProvider(slot, key)
	})

	err = wails.Run(&options.App{
		Title:     "MSG Desktop",
		Width:     1280,
		Height:    800,
		MinWidth:  1024,
		MinHeight: 600,
		AssetServer: &assetserver.Options{
			Assets: frontend.Assets,
		},
		BackgroundColour: &options.RGBA{R: 255, G: 255, B: 255, A: 1},
		OnStartup:        app.Startup,
		Bind:             []interface{}{app},
	})
	if err != nil {
		log.Fatalf("wails: %v", err)
	}
	_ = dir
}
