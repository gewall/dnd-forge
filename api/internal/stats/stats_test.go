package stats

import (
	"context"
	"errors"
	"testing"
	"time"
)

type recorder struct {
	from, to, since time.Time
	modelsSince     time.Time
}

func (r *recorder) Totals(context.Context) (Totals, error) { return Totals{Users: 3}, nil }

func (r *recorder) Days(_ context.Context, from, to, since time.Time) ([]Day, error) {
	r.from, r.to, r.since = from, to, since
	return nil, nil
}

func (r *recorder) Models(_ context.Context, since time.Time) ([]ModelUsage, error) {
	r.modelsSince = since
	return nil, nil
}

func TestReportPeriod(t *testing.T) {
	store := &recorder{}
	svc := NewService(store)
	// 18:30 UTC on 3 Nov is already 4 Nov in WIB.
	svc.now = func() time.Time { return time.Date(2026, 11, 3, 18, 30, 0, 0, time.UTC) }

	r, err := svc.Report(context.Background(), 7)
	if err != nil {
		t.Fatal(err)
	}
	if want := time.Date(2026, 11, 4, 0, 0, 0, 0, time.UTC); !r.To.Equal(want) || !store.to.Equal(want) {
		t.Errorf("to = %v, want %v", r.To, want)
	}
	if want := time.Date(2026, 10, 29, 0, 0, 0, 0, time.UTC); !r.From.Equal(want) {
		t.Errorf("from = %v, want %v (7 days, today included)", r.From, want)
	}
	// The period starts at WIB midnight: 17:00 UTC the day before.
	if want := time.Date(2026, 10, 28, 17, 0, 0, 0, time.UTC); !store.since.Equal(want) || !store.modelsSince.Equal(want) {
		t.Errorf("since = %v / %v, want %v", store.since, store.modelsSince, want)
	}
	if r.Totals.Users != 3 {
		t.Errorf("totals = %+v", r.Totals)
	}

	for _, days := range []int{0, MaxDays + 1} {
		if _, err := svc.Report(context.Background(), days); !errors.Is(err, ErrInvalidDays) {
			t.Errorf("Report(%d) err = %v", days, err)
		}
	}
}
