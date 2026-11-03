package portrait

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/riverqueue/river"
)

// GenerateArgs is the River job that generates one portrait.
type GenerateArgs struct {
	PortraitID     uuid.UUID `json:"portraitId"`
	NegativePrompt string    `json:"negativePrompt,omitempty"`
}

func (GenerateArgs) Kind() string { return "portrait.generate" }

// InsertOpts: the first attempt plus two retries, as the design specifies.
func (GenerateArgs) InsertOpts() river.InsertOpts { return river.InsertOpts{MaxAttempts: 3} }

// Generator runs one generation attempt.
type Generator interface {
	Generate(ctx context.Context, args GenerateArgs, lastAttempt bool) error
}

// Worker adapts Generator to River.
type Worker struct {
	river.WorkerDefaults[GenerateArgs]
	gen Generator
}

// NewWorker returns a River worker for portrait jobs.
func NewWorker(gen Generator) *Worker { return &Worker{gen: gen} }

// Timeout covers slow providers (tens of seconds) plus encoding and upload.
func (w *Worker) Timeout(*river.Job[GenerateArgs]) time.Duration { return 3 * time.Minute }

func (w *Worker) Work(ctx context.Context, job *river.Job[GenerateArgs]) error {
	err := w.gen.Generate(ctx, job.Args, job.Attempt >= job.MaxAttempts)
	if errors.Is(err, errPermanent) {
		return river.JobCancel(err) // already marked FAILED; do not retry
	}
	return err
}
