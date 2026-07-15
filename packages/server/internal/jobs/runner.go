// Package jobs runs queued review jobs. A single background loop drains the
// queue: it claims pending jobs and invokes the handler for each. Claiming is
// atomic in the store, so the design extends to multiple runners later.
package jobs

import (
	"context"
	"fmt"
	"log"
	"runtime/debug"
	"sync"
	"time"

	"github.com/webcloster-dev/ai-reviewer/internal/domain/job"
)

// Handler executes the work for a review. Returning an error fails the job.
type Handler func(ctx context.Context, reviewID string) error

// Runner drives the job queue.
type Runner struct {
	store       job.Store
	handler     Handler
	poll        time.Duration
	logger      *log.Logger
	signal      chan struct{}
	concurrency int
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

// WithConcurrency sets how many jobs run at once. Values below 1 default to 1.
// The atomic Claim hands each worker a distinct pending job.
func WithConcurrency(n int) Option {
	return func(r *Runner) {
		if n < 1 {
			n = 1
		}
		r.concurrency = n
	}
}

// NewRunner builds a runner over a store and handler.
func NewRunner(store job.Store, handler Handler, opts ...Option) *Runner {
	r := &Runner{
		store:       store,
		handler:     handler,
		poll:        5 * time.Second,
		logger:      log.Default(),
		signal:      make(chan struct{}, 1),
		concurrency: 1,
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

// Start recovers interrupted jobs and runs the worker pool until ctx is done.
// It blocks; run it in its own goroutine. Concurrency workers claim and run jobs
// in parallel; a bound of 1 preserves the original sequential behavior.
func (r *Runner) Start(ctx context.Context) {
	if n, err := r.store.RequeueRunning(ctx); err != nil {
		r.logger.Printf("jobs: requeue running failed: %v", err)
	} else if n > 0 {
		r.logger.Printf("jobs: requeued %d interrupted job(s)", n)
	}

	var wg sync.WaitGroup
	for i := 0; i < r.concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			r.worker(ctx)
		}()
	}
	wg.Wait()
}

// worker claims and runs one job at a time until ctx is done. Running several of
// these gives parallelism: the atomic Claim hands each a distinct pending job,
// and each worker wakes another after claiming so a burst of enqueues spreads
// across the pool instead of one worker draining them serially.
func (r *Runner) worker(ctx context.Context) {
	ticker := time.NewTicker(r.poll)
	defer ticker.Stop()
	for {
		if ctx.Err() != nil {
			return
		}
		j, ok, err := r.store.Claim(ctx)
		if err != nil {
			r.logger.Printf("jobs: claim failed: %v", err)
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
			continue
		}
		if !ok {
			select {
			case <-ctx.Done():
				return
			case <-r.signal:
			case <-ticker.C:
			}
			continue
		}
		r.wake() // more may be pending — let another idle worker grab one
		r.run(ctx, j)
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
	if err := r.safeHandle(ctx, j.ReviewID); err != nil {
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

// safeHandle invokes the handler, converting a panic into an error so a single
// panicking review fails only its own job instead of crashing the whole process
// (there is no external supervisor to restart it).
func (r *Runner) safeHandle(ctx context.Context, reviewID string) (err error) {
	defer func() {
		if rec := recover(); rec != nil {
			err = fmt.Errorf("panic: %v\n%s", rec, debug.Stack())
		}
	}()
	return r.handler(ctx, reviewID)
}

func (r *Runner) wake() {
	select {
	case r.signal <- struct{}{}:
	default:
	}
}
