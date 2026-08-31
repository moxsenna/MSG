package scraper

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gosom/google-maps-scraper/gmaps"
	"github.com/gosom/google-maps-scraper/log"
	"github.com/gosom/scrapemate"
	"github.com/gosom/scrapemate/scrapemateapp"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ScraperManager manages the scraper lifecycle, restarting it after a configured
// number of jobs to prevent memory leaks.
//
//nolint:revive // ScraperManager is intentionally named with its package context
type ScraperManager struct {
	dbPool      *pgxpool.Pool
	concurrency int
	fastMode    bool
	debug       bool
	maxJobs     int64
	proxies     []string

	mu       sync.RWMutex
	provider *Provider

	centralWriter *CentralWriter

	jobCount atomic.Int64

	// Channel to signal restart needed
	restartChan chan struct{}

	// OnJobComplete is called after each job finishes (for stats tracking).
	OnJobComplete func()
}

// NewScraperManager creates a new ScraperManager.
func NewScraperManager(dbPool *pgxpool.Pool, concurrency int, fastMode, debug bool, maxJobs int64, proxies []string) *ScraperManager {
	if concurrency <= 0 {
		concurrency = 1
	}

	if maxJobs <= 0 {
		maxJobs = math.MaxInt64
	}

	return &ScraperManager{
		dbPool:        dbPool,
		concurrency:   concurrency,
		fastMode:      fastMode,
		debug:         debug,
		maxJobs:       maxJobs,
		proxies:       proxies,
		restartChan:   make(chan struct{}, 1),
		centralWriter: NewCentralWriter(dbPool, nil),
	}
}

// CentralWriter returns the CentralWriter instance.
func (m *ScraperManager) CentralWriter() *CentralWriter {
	return m.centralWriter
}

// getProvider returns the current provider.
func (m *ScraperManager) getProvider() *Provider {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.provider
}

// JobDone is called after each River job completes.
func (m *ScraperManager) JobDone() {
	count := m.jobCount.Add(1)

	if m.OnJobComplete != nil {
		m.OnJobComplete()
	}

	if count >= m.maxJobs {
		// Signal restart needed (non-blocking)
		select {
		case m.restartChan <- struct{}{}:
		default:
		}
	}
}

// ActiveJobs returns the number of currently active scrape jobs.
func (m *ScraperManager) ActiveJobs() int64 {
	return int64(m.centralWriter.TrackedJobs())
}

// SubmitJob submits a job to the current provider.
func (m *ScraperManager) SubmitJob(ctx context.Context, job scrapemate.IJob) error {
	switch j := job.(type) {
	case *gmaps.GmapJob:
		if !j.WriterManagedCompletion {
			return fmt.Errorf("failed to submit job: GmapJob requires WriterManagedCompletion")
		}
	case *gmaps.SearchJob:
		if !j.WriterManagedCompletion {
			return fmt.Errorf("failed to submit job: SearchJob requires WriterManagedCompletion")
		}
	}

	ticker := time.NewTicker(25 * time.Millisecond)
	defer ticker.Stop()

	for {
		p := m.getProvider()
		if p != nil {
			return p.Submit(ctx, job)
		}

		select {
		case <-ctx.Done():
			return fmt.Errorf("failed to submit job: scraper provider not ready: %w", ctx.Err())
		case <-ticker.C:
		}
	}
}

// RegisterJob delegates to CentralWriter.
func (m *ScraperManager) RegisterJob(jobID string, riverJobID int64, keyword string) <-chan FlushResult {
	return m.centralWriter.RegisterJob(jobID, riverJobID, keyword)
}

// MarkDone delegates to CentralWriter.
func (m *ScraperManager) MarkDone(jobID string) {
	m.centralWriter.MarkDone(jobID)
}

// ForceFlush delegates to CentralWriter.
func (m *ScraperManager) ForceFlush(jobID string) {
	m.centralWriter.ForceFlush(jobID)
}

// Run starts the scraper manager loop. It creates a new scraper, runs it until
// the job threshold is reached, then restarts.
func (m *ScraperManager) Run(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		if err := m.runCycle(ctx); err != nil {
			if ctx.Err() != nil {
				return nil
			}

			return err
		}
	}
}

func (m *ScraperManager) runCycle(ctx context.Context) error {
	provider := NewProvider(m.concurrency * 64)

	// Update references atomically
	m.mu.Lock()
	m.provider = provider
	m.jobCount.Store(0)
	m.mu.Unlock()

	// Create scraper app
	app, err := m.createApp(provider)
	if err != nil {
		return err
	}

	defer func() { _ = app.Close() }()

	// Create cycle context
	cycleCtx, cycleCancel := context.WithCancel(ctx)
	defer cycleCancel()

	// Start scraper in goroutine with panic recovery
	scraperDone := make(chan error, 1)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				scraperDone <- fmt.Errorf("scraper panic: %v", r)
			}
		}()
		scraperDone <- app.Start(cycleCtx)
	}()

	log.Info("scraper cycle started", "max_jobs", m.maxJobs)

	// Wait for restart signal, scraper done, or context cancelled
	select {
	case <-ctx.Done():
		cycleCancel()
		<-scraperDone

		return nil

	case err := <-scraperDone:
		// Scraper exited unexpectedly
		if ctx.Err() != nil {
			return nil
		}

		return err

	case <-m.restartChan:
		log.Info("restart triggered, restarting scraper",
			"jobs_processed", m.jobCount.Load(),
		)

		cycleCancel()
		<-scraperDone

		return nil
	}
}

func (m *ScraperManager) createApp(provider *Provider) (*scrapemateapp.ScrapemateApp, error) {
	writers := []scrapemate.ResultWriter{m.centralWriter}

	var opts []func(*scrapemateapp.Config) error
	opts = append(opts,
		scrapemateapp.WithConcurrency(m.concurrency),
		scrapemateapp.WithProvider(provider),
	)

	if len(m.proxies) > 0 {
		checked := m.checkProxies()
		if len(checked) > 0 {
			opts = append(opts, scrapemateapp.WithProxies(checked))
			log.Info("proxies configured", "count", len(checked), "original", len(m.proxies))
		}
	}

	if m.fastMode {
		opts = append(opts, scrapemateapp.WithStealth("firefox"))
	} else {
		if m.debug {
			opts = append(opts, scrapemateapp.WithJS(
				scrapemateapp.Headfull(),
				scrapemateapp.DisableImages(),
			))
		} else {
			opts = append(opts, scrapemateapp.WithJS(scrapemateapp.DisableImages()))
		}
	}

	cfg, err := scrapemateapp.NewConfig(writers, opts...)
	if err != nil {
		return nil, err
	}

	return scrapemateapp.NewScrapeMateApp(cfg)
}

func (m *ScraperManager) checkProxies() []string {
	if len(m.proxies) == 0 {
		return nil
	}

	healthy := make([]string, 0, len(m.proxies))
	for _, p := range m.proxies {
		trimmed := strings.TrimSpace(p)
		if trimmed == "" {
			continue
		}

		parsed, err := url.Parse(trimmed)
		if err != nil {
			log.Error("proxy invalid URL, blacklisted", "proxy", trimmed, "error", err)
			continue
		}

		switch strings.ToLower(parsed.Scheme) {
		case "socks5", "socks5h":
			healthy = append(healthy, trimmed)
			continue
		case "http", "https":
		default:
			log.Error("proxy unsupported scheme, blacklisted", "proxy", trimmed, "scheme", parsed.Scheme)
			continue
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		err = probeProxy(ctx, trimmed)
		cancel()
		if err != nil {
			log.Error("proxy health check failed, blacklisted", "proxy", trimmed, "error", err)
			continue
		}

		healthy = append(healthy, trimmed)
	}

	return healthy
}

func probeProxy(ctx context.Context, proxyURL string) error {
	parsed, err := url.Parse(proxyURL)
	if err != nil {
		return err
	}

	client := &http.Client{
		Transport: &http.Transport{Proxy: http.ProxyURL(parsed)},
		Timeout:   5 * time.Second,
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodHead, "https://www.google.com/generate_204", http.NoBody)
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
