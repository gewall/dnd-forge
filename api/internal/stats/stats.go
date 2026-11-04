// Package stats summarises usage for the admin dashboard: totals, daily
// activity per WIB day and AI requests per provider and model.
package stats

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/gewall/dnd-forge/api/internal/apperr"
	"github.com/gewall/dnd-forge/api/internal/quota"
)

// MaxDays bounds the reporting period.
const MaxDays = 90

// ErrInvalidDays rejects a period outside 1..MaxDays.
var ErrInvalidDays = apperr.New(apperr.Invalid, fmt.Sprintf("periode harus 1 sampai %d hari", MaxDays))

// Totals are all-time counts.
type Totals struct {
	Users, BannedUsers, Admins                       int
	Characters, PublicCharacters                     int
	Portraits, PortraitsSucceeded, PortraitsFailed   int
	PortraitsPending, PortraitsHidden                int
	Lores, LoresSucceeded, LoresFailed, LoresPending int
}

// Day is one WIB day of activity. FAILED requests do not count against
// quotas but do cost provider calls, so they are reported separately.
type Day struct {
	Date               time.Time // midnight UTC of the WIB calendar date
	NewUsers           int
	NewCharacters      int
	PortraitsSucceeded int
	PortraitsFailed    int
	LoresSucceeded     int
	LoresFailed        int
}

// ModelUsage is the AI usage of one provider and model in the period.
// Token counts are what text vendors reported; images have none.
type ModelUsage struct {
	Capability       string // image or text
	ProviderID       uuid.UUID
	ProviderLabel    string
	Model            string
	Requests         int
	Succeeded        int
	Failed           int
	PromptTokens     int64
	CompletionTokens int64
}

// Report is the dashboard for one period.
type Report struct {
	From, To time.Time // first and last WIB day, as dates
	Totals   Totals
	Days     []Day
	Models   []ModelUsage
}

// Store runs the aggregate queries.
type Store interface {
	Totals(ctx context.Context) (Totals, error)
	Days(ctx context.Context, from, to, since time.Time) ([]Day, error)
	Models(ctx context.Context, since time.Time) ([]ModelUsage, error)
}

// Service builds reports.
type Service struct {
	store Store
	now   func() time.Time
}

// NewService returns a Service.
func NewService(store Store) *Service { return &Service{store: store, now: time.Now} }

// Report summarises the last days WIB days, today included.
func (s *Service) Report(ctx context.Context, days int) (Report, error) {
	if days < 1 || days > MaxDays {
		return Report{}, ErrInvalidDays
	}
	today, _ := quota.Day(s.now())
	since := today.AddDate(0, 0, -(days - 1))
	r := Report{From: date(since), To: date(today)}

	var err error
	if r.Totals, err = s.store.Totals(ctx); err != nil {
		return Report{}, fmt.Errorf("stats: totals: %w", err)
	}
	if r.Days, err = s.store.Days(ctx, r.From, r.To, since); err != nil {
		return Report{}, fmt.Errorf("stats: days: %w", err)
	}
	if r.Models, err = s.store.Models(ctx, since); err != nil {
		return Report{}, fmt.Errorf("stats: models: %w", err)
	}
	return r, nil
}

// date returns the calendar date of a WIB midnight as midnight UTC, the
// form PostgreSQL date values take.
func date(wibMidnight time.Time) time.Time {
	y, m, d := wibMidnight.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}
