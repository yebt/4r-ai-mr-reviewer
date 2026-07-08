// Package jobs runs queued review jobs. A single background loop drains the
// queue: it claims pending jobs and invokes the handler for each. Claiming is
// atomic in the store, so the design extends to multiple runners later.
package jobs

import (
	"context"
	"log"
	"time"

	"github.com/webcloster-dev/ai-reviewer/internal/domain/job"
)

// Handler executes the work for a review. Returning an error fails the job.
type Handler func(ctx context.Context, reviewID string) error

// Runner drives the job queue.
type Runner struct {
	store   job.Store
	handler Handler
	poll    time.Duration
	logger  *log.Logger
	signal  chan struct{}
}

// Option configures a Runner.
type Option func(*Runner)

// WithPollInterval sets how often the runner sweeps for pending jobs even
// without an explicit wake-up. It bounds recovery latency.
func WithPollInterval(d time.Duration) Option {
	return func(r *Runner) { r.poll = d }
}

// WithLogger sets the logger used for job failures.
func WithLogger(l *log.Logger) Option {
	return func(r *Runner) { r.logger = l }
}

// NewRunner builds a runner over a store and handler.
func NewRunner(store job.Store, handler Handler, opts ...Option) *Runner {
	r := &Runner{
		store:   store,
		handler: handler,
		poll:    5 * time.Second,
		logger:  log.Default(),
		signal:  make(chan struct{}, 1),
	}
	for _, o := range opts {
		o(r)
	}
	return r
}

// Enqueue queues a review for execution and wakes the loop.
func (r *Runner) Enqueue(ctx context.Context, reviewID string) (job.Job, error) {
	j, err := r.store.Enqueue(ctx, reviewID)
	if err != nil {
		return job.Job{}, err
	}
	r.wake()
	return j, nil
}

// Start recovers interrupted jobs and runs the drain loop until ctx is done.
// It blocks; run it in its own goroutine.
func (r *Runner) Start(ctx context.Context) {
	if n, err := r.store.RequeueRunning(ctx); err != nil {
		r.logger.Printf("jobs: requeue running failed: %v", err)
	} else if n > 0 {
		r.logger.Printf("jobs: requeued %d interrupted job(s)", n)
	}

	ticker := time.NewTicker(r.poll)
	defer ticker.Stop()
	for {
		r.Drain(ctx)
		select {
		case <-ctx.Done():
			return
		case <-r.signal:
		case <-ticker.C:
		}
	}
}

// Drain claims and runs pending jobs one at a time until the queue is empty or
// the context is cancelled. Exposed for deterministic testing.
func (r *Runner) Drain(ctx context.Context) {
	for {
		if ctx.Err() != nil {
			return
		}
		j, ok, err := r.store.Claim(ctx)
		if err != nil {
			r.logger.Printf("jobs: claim failed: %v", err)
			return
		}
		if !ok {
			return
		}
		r.run(ctx, j)
	}
}

func (r *Runner) run(ctx context.Context, j job.Job) {
	if err := r.handler(ctx, j.ReviewID); err != nil {
		r.logger.Printf("jobs: review %s failed (attempt %d): %v", j.ReviewID, j.Attempts, err)
		if ferr := r.store.Fail(ctx, j.ID, err.Error()); ferr != nil {
			r.logger.Printf("jobs: marking job %s failed: %v", j.ID, ferr)
		}
		return
	}
	if err := r.store.Complete(ctx, j.ID); err != nil {
		r.logger.Printf("jobs: marking job %s done: %v", j.ID, err)
	}
}

func (r *Runner) wake() {
	select {
	case r.signal <- struct{}{}:
	default:
	}
}
