package housekeeping

import (
	"context"
	"log/slog"
	"time"

	"github.com/riverqueue/river"
)

// PurgeArgs is the daily cleanup job.
type PurgeArgs struct{}

// Kind identifies the job in River.
func (PurgeArgs) Kind() string { return "housekeeping.purge" }

// PurgeWorker runs PurgeArgs jobs.
type PurgeWorker struct {
	river.WorkerDefaults[PurgeArgs]
	svc *Service
	log *slog.Logger
}

// NewPurgeWorker returns a worker for svc.
func NewPurgeWorker(svc *Service, log *slog.Logger) *PurgeWorker {
	return &PurgeWorker{svc: svc, log: log}
}

// Timeout bounds one cleanup run.
func (*PurgeWorker) Timeout(*river.Job[PurgeArgs]) time.Duration { return time.Minute }

func (w *PurgeWorker) Work(ctx context.Context, _ *river.Job[PurgeArgs]) error {
	n, err := w.svc.PurgeTokens(ctx)
	if err != nil {
		return err
	}
	w.log.InfoContext(ctx, "expired refresh tokens purged", "count", n)
	return nil
}

// PeriodicPurge schedules the cleanup once a day, and once at start so a
// restart never postpones it by a full day.
func PeriodicPurge() *river.PeriodicJob {
	return river.NewPeriodicJob(
		river.PeriodicInterval(24*time.Hour),
		func() (river.JobArgs, *river.InsertOpts) { return PurgeArgs{}, nil },
		&river.PeriodicJobOpts{RunOnStart: true},
	)
}
