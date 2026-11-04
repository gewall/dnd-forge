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

// Kind names one daily quota.
type Kind string

const (
	Portrait Kind = "portrait"
	Lore     Kind = "lore"
)

// MaxLimit bounds every daily limit an admin can set.
const MaxLimit = 1000

// defaults maps each quota to its app_settings key and the limit used when
// that setting is missing or malformed.
var defaults = map[Kind]struct {
	key      string
	fallback int
}{
	Portrait: {"default_daily_quota", 5},
	Lore:     {"default_daily_lore_quota", 10},
}

// SettingReader reads an app_settings value (JSON).
type SettingReader interface {
	SettingValue(ctx context.Context, key string) ([]byte, error)
}

// DefaultLimit returns the admin-set default daily limit of k, or the
// built-in fallback when the setting is missing or malformed.
func DefaultLimit(ctx context.Context, r SettingReader, k Kind) (int, error) {
	d, ok := defaults[k]
	if !ok {
		return 0, fmt.Errorf("quota: unknown kind %q", k)
	}
	raw, err := r.SettingValue(ctx, d.key)
	if db.IsNoRows(err) {
		return d.fallback, nil
	}
	if err != nil {
		return 0, fmt.Errorf("setting %s: %w", d.key, err)
	}
	var n int
	if err := json.Unmarshal(raw, &n); err != nil || n < 0 {
		return d.fallback, nil //nolint:nilerr // a malformed setting falls back to the default
	}
	return n, nil
}
