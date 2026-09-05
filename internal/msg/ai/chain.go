package ai

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"
)

const cooldownAfterFailure = 60 * time.Second

type Chain struct {
	mu       sync.Mutex
	cooldown map[string]time.Time
	build    func(slot KeySlot) (Provider, error)
}

func NewChain(build func(slot KeySlot) (Provider, error)) *Chain {
	return &Chain{cooldown: map[string]time.Time{}, build: build}
}

func (c *Chain) available(slots []KeySlot) []KeySlot {
	now := time.Now()
	out := []KeySlot{}
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, sl := range slots {
		if !sl.Enabled {
			continue
		}
		if until, ok := c.cooldown[sl.ID]; ok && now.Before(until) {
			continue
		}
		out = append(out, sl)
	}
	return out
}

func (c *Chain) markFailed(id string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cooldown[id] = time.Now().Add(cooldownAfterFailure)
}

func (c *Chain) run(ctx context.Context, slots []KeySlot, fn func(p Provider) (any, error)) (any, string, error) {
	avail := c.available(slots)
	if len(avail) == 0 {
		return nil, "", fmt.Errorf("no enabled AI key available (all disabled or in cooldown) — tambah key di Settings atau tunggu 60 detik")
	}
	var errs []string
	for _, sl := range avail {
		if ctx.Err() != nil {
			return nil, "", ctx.Err()
		}
		p, err := c.build(sl)
		if err != nil {
			errs = append(errs, sl.Label+": build failed")
			continue
		}
		res, err := fn(p)
		if err == nil {
			return res, sl.ID, nil
		}
		c.markFailed(sl.ID)
		errs = append(errs, fmt.Sprintf("%s (%s): %s", sl.Label, sl.Provider, shortErr(err)))
	}
	return nil, "", fmt.Errorf("all %d AI keys failed: %s", len(avail), strings.Join(errs, " | "))
}

func (c *Chain) AnalyzeOpportunity(ctx context.Context, slots []KeySlot, businessName, category, city, websiteInfo string) (*OpportunityAnalysis, string, error) {
	res, usedID, err := c.run(ctx, slots, func(p Provider) (any, error) {
		return p.AnalyzeOpportunity(ctx, businessName, category, city, websiteInfo)
	})
	if err != nil {
		return nil, "", err
	}
	return res.(*OpportunityAnalysis), usedID, nil
}

func (c *Chain) DraftOutreach(ctx context.Context, slots []KeySlot, businessName, category, offer, channel, tone string) (string, string, error) {
	res, usedID, err := c.run(ctx, slots, func(p Provider) (any, error) {
		return p.DraftOutreach(ctx, businessName, category, offer, channel, tone)
	})
	if err != nil {
		return "", "", err
	}
	return res.(string), usedID, nil
}

func shortErr(err error) string {
	msg := err.Error()
	if len(msg) > 160 {
		return msg[:160] + "..."
	}
	return msg
}
