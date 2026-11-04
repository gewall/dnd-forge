package quota

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
)

func TestDay(t *testing.T) {
	// 17:00 UTC is midnight WIB: the new day has just begun.
	start, next := Day(time.Date(2026, 10, 29, 17, 0, 0, 0, time.UTC))
	if !start.Equal(time.Date(2026, 10, 29, 17, 0, 0, 0, time.UTC)) || !next.Equal(start.Add(24*time.Hour)) {
		t.Errorf("Day(17:00 UTC) = %v, %v", start, next)
	}
	// 16:59 UTC is still the previous WIB day.
	start, _ = Day(time.Date(2026, 10, 29, 16, 59, 0, 0, time.UTC))
	if !start.Equal(time.Date(2026, 10, 28, 17, 0, 0, 0, time.UTC)) {
		t.Errorf("Day(16:59 UTC) starts %v", start)
	}
}

func TestNewUsage(t *testing.T) {
	if u := NewUsage(3, 7, time.Time{}); u.Remaining != 0 || u.Used != 7 {
		t.Errorf("over-used = %+v", u)
	}
	if u := NewUsage(10, 4, time.Time{}); u.Remaining != 6 {
		t.Errorf("Remaining = %d, want 6", u.Remaining)
	}
}

type settings map[string]string

func (s settings) SettingValue(_ context.Context, key string) ([]byte, error) {
	v, ok := s[key]
	if !ok {
		return nil, pgx.ErrNoRows
	}
	return []byte(v), nil
}

func TestDefaultLimit(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name string
		s    settings
		k    Kind
		want int
	}{
		{"set", settings{"default_daily_quota": "12"}, Portrait, 12},
		{"zero is allowed", settings{"default_daily_lore_quota": "0"}, Lore, 0},
		{"malformed", settings{"default_daily_quota": `"lots"`}, Portrait, 5},
		{"negative", settings{"default_daily_quota": "-3"}, Portrait, 5},
		{"missing portrait", settings{}, Portrait, 5},
		{"missing lore", settings{}, Lore, 10},
	}
	for _, tt := range tests {
		if got, err := DefaultLimit(ctx, tt.s, tt.k); err != nil || got != tt.want {
			t.Errorf("%s: DefaultLimit = %d, %v; want %d", tt.name, got, err, tt.want)
		}
	}
	boom := errors.New("down")
	if _, err := DefaultLimit(ctx, failing{boom}, Portrait); !errors.Is(err, boom) {
		t.Errorf("read error = %v", err)
	}
	if _, err := DefaultLimit(ctx, settings{}, "coins"); err == nil {
		t.Error("unknown kind accepted")
	}
}

type failing struct{ err error }

func (f failing) SettingValue(context.Context, string) ([]byte, error) { return nil, f.err }
