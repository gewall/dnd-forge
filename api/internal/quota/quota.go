// Package quota holds what the daily AI generation quotas share: the WIB day
// they reset on, the usage report, the admin-set default limits and the
// endpoint that shows every quota of the signed-in user.
package quota

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gewall/dnd-forge/api/internal/db"
)

// wib is Indonesia's western time zone; quotas reset at local midnight.
// Indonesia has no daylight saving, so a fixed zone is exact and avoids
// shipping tzdata in the static image.
var wib = time.FixedZone("WIB", 7*60*60)

// Day returns local midnight (WIB) of t and the next midnight.
func Day(t time.Time) (start, next time.Time) {
	y, m, d := t.In(wib).Date()
	start = time.Date(y, m, d, 0, 0, 0, 0, wib)
	return start, start.AddDate(0, 0, 1)
}

// Usage is a user's allowance of one quota for one day.
type Usage struct {
	Limit     int
	Used      int
	Remaining int
	ResetsAt  time.Time
}

// NewUsage reports used of limit; Remaining never goes below zero, e.g.
// after an admin lowered the limit.
func NewUsage(limit, used int, resetsAt time.Time) Usage {
	return Usage{Limit: limit, Used: used, Remaining: max(limit-used, 0), ResetsAt: resetsAt}
}

// SettingReader reads an app_settings value (JSON).
type SettingReader interface {
	SettingValue(ctx context.Context, key string) ([]byte, error)
}

// DefaultLimit reads the integer setting key, or fallback when it is
// missing or malformed.
func DefaultLimit(ctx context.Context, r SettingReader, key string, fallback int) (int, error) {
	raw, err := r.SettingValue(ctx, key)
	if db.IsNoRows(err) {
		return fallback, nil
	}
	if err != nil {
		return 0, fmt.Errorf("setting %s: %w", key, err)
	}
	var n int
	if err := json.Unmarshal(raw, &n); err != nil {
		return fallback, nil //nolint:nilerr // a malformed setting falls back to the default
	}
	return n, nil
}
