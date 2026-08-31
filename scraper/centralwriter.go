package scraper

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/gosom/google-maps-scraper/gmaps"
	"github.com/gosom/google-maps-scraper/internal/jsonbsanitize"
	"github.com/gosom/google-maps-scraper/log"
	"github.com/gosom/scrapemate"
	"github.com/jackc/pgx/v5/pgxpool"
)

// FlushResult is sent to the River worker when results are flushed to DB.
type FlushResult struct {
	ResultCount int
	Err         error
}

type trackedJob struct {
	jobID      string
	entries    []*gmaps.Entry
	completion chan FlushResult
	riverJobID int64
	keyword    string
	startedAt  time.Time
}

// SaveFunc persists results. The default writes to PostgreSQL;
// tests can supply a lightweight substitute via NewCentralWriter.
type SaveFunc func(ctx context.Context, riverJobID int64, keyword string, entries []*gmaps.Entry) error

var _ scrapemate.ResultWriter = (*CentralWriter)(nil)

// CentralWriter tracks in-flight River jobs, receives ScrapeMate
// results, and flushes them to the database when the exit monitor signals done.
type CentralWriter struct {
	mu     sync.Mutex
	jobs   map[string]*trackedJob
	save   SaveFunc
	OnResultsSaved func(count int)
}

// NewCentralWriter creates a new CentralWriter.
// Pass nil for saveFn to use the default PostgreSQL saver built from db.
func NewCentralWriter(db *pgxpool.Pool, saveFn SaveFunc) *CentralWriter {
	if saveFn == nil {
		saveFn = pgSave(db)
	}

	return &CentralWriter{
		jobs: make(map[string]*trackedJob),
		save: saveFn,
	}
}

func (cw *CentralWriter) RegisterJob(jobID string, riverJobID int64, keyword string) <-chan FlushResult {
	cw.mu.Lock()
	defer cw.mu.Unlock()

	ch := make(chan FlushResult, 1)
	if cw.jobs == nil {
		cw.jobs = make(map[string]*trackedJob)
	}
	cw.jobs[jobID] = &trackedJob{
		jobID:      jobID,
		completion: ch,
		riverJobID: riverJobID,
		keyword:    keyword,
		startedAt:  time.Now(),
	}

	log.Debug("registered scrape job", "job_id", jobID, "river_job_id", riverJobID)

	return ch
}

func (cw *CentralWriter) AddResult(jobID string, entry *gmaps.Entry) {
	cw.mu.Lock()
	defer cw.mu.Unlock()

	j, ok := cw.jobs[jobID]
	if !ok {
		return
	}

	j.entries = append(j.entries, entry)
}

// MarkDone is called by the exit monitor when a job is complete.
func (cw *CentralWriter) MarkDone(jobID string) {
	cw.Flush(jobID)
}

// ForceFlush immediately flushes results for a job (used on timeout/shutdown).
func (cw *CentralWriter) ForceFlush(jobID string) {
	cw.Flush(jobID)
}

func (cw *CentralWriter) Discard(jobID string) {
	cw.mu.Lock()
	defer cw.mu.Unlock()

	delete(cw.jobs, jobID)
}

func (cw *CentralWriter) TrackedJobs() int {
	cw.mu.Lock()
	defer cw.mu.Unlock()

	return len(cw.jobs)
}

func (cw *CentralWriter) FlushQueueDepth() int {
	return 0
}

func (cw *CentralWriter) Flush(jobID string) {
	cw.mu.Lock()
	j, ok := cw.jobs[jobID]
	if !ok {
		cw.mu.Unlock()
		return
	}

	delete(cw.jobs, jobID)
	cw.mu.Unlock()

	for _, entry := range j.entries {
		jsonbsanitize.StripNULFromEntry(entry)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	err := cw.save(ctx, j.riverJobID, j.keyword, j.entries)

	cancel()

	if err != nil {
		log.Error("failed to save results",
			"job_id", j.jobID,
			"river_job_id", j.riverJobID,
			"error", err,
		)
	} else if cw.OnResultsSaved != nil && len(j.entries) > 0 {
		cw.OnResultsSaved(len(j.entries))
	}

	j.completion <- FlushResult{ResultCount: len(j.entries), Err: err}

	log.Debug("flushed scrape job",
		"job_id", j.jobID,
		"river_job_id", j.riverJobID,
		"result_count", len(j.entries),
		"duration_ms", time.Since(j.startedAt).Milliseconds(),
		"save_error", err,
	)
}

// Run processes results from the ScrapeMate scraper and buffers them for the
// currently registered River job.
func (cw *CentralWriter) Run(ctx context.Context, in <-chan scrapemate.Result) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case result, ok := <-in:
			if !ok {
				return nil
			}

			cw.processResult(result)
		}
	}
}

func (cw *CentralWriter) processResult(result scrapemate.Result) {
	jobID := ""
	if result.Job != nil {
		jobID = result.Job.GetID()
	}

	if entry, ok := result.Data.(*gmaps.Entry); ok {
		entryJobID := entry.ID
		if entryJobID == "" {
			entryJobID = jobID
		}

		cw.AddResult(entryJobID, entry)
		cw.markCompletedFromResult(result, 1)

		return
	}

	if entries, ok := result.Data.([]*gmaps.Entry); ok {
		for _, entry := range entries {
			entry.ID = jobID
			cw.AddResult(jobID, entry)
		}

		cw.markCompletedFromResult(result, len(entries))
	}
}

func (cw *CentralWriter) markCompletedFromResult(result scrapemate.Result, count int) {
	if count <= 0 || result.Job == nil {
		return
	}

	switch job := result.Job.(type) {
	case *gmaps.PlaceJob:
		if job.WriterManagedCompletion && job.ExitMonitor != nil {
			job.ExitMonitor.IncrPlacesCompleted(count)
		}
	case *gmaps.EmailExtractJob:
		if job.WriterManagedCompletion && job.ExitMonitor != nil {
			job.ExitMonitor.IncrPlacesCompleted(count)
		}
	case *gmaps.SearchJob:
		if job.WriterManagedCompletion && job.ExitMonitor != nil {
			job.ExitMonitor.IncrPlacesCompleted(count)
		}
	}
}

// pgSave returns a SaveFunc that writes to the scrape_results table.
func pgSave(db *pgxpool.Pool) SaveFunc {
	return func(ctx context.Context, riverJobID int64, keyword string, entries []*gmaps.Entry) error {
		resultsJSON, err := json.Marshal(entries)
		if err != nil {
			return err
		}

		q := `INSERT INTO scrape_results (job_id, keyword, results, result_count)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (job_id) DO UPDATE SET
				results = $3,
				result_count = $4`

		_, err = db.Exec(ctx, q, riverJobID, keyword, resultsJSON, len(entries))

		return err
	}
}
