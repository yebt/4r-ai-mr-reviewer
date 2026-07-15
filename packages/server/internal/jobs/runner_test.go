package jobs

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/webcloster-dev/ai-reviewer/internal/domain/job"
)

// fakeStore is an in-memory job.Store for runner unit tests.
type fakeStore struct {
	mu       sync.Mutex
	jobs     []*job.Job
	seq      int
	requeued int
}

func (s *fakeStore) Enqueue(_ context.Context, reviewID string) (job.Job, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.seq++
	j := &job.Job{ID: string(rune('a' + s.seq)), ReviewID: reviewID, Status: job.StatusPending}
	s.jobs = append(s.jobs, j)
	return *j, nil
}

func (s *fakeStore) Claim(context.Context) (job.Job, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, j := range s.jobs {
		if j.Status == job.StatusPending {
			j.Status = job.StatusRunning
			j.Attempts++
			return *j, true, nil
		}
	}
	return job.Job{}, false, nil
}

func (s *fakeStore) find(id string) *job.Job {
	for _, j := range s.jobs {
		if j.ID == id {
			return j
		}
	}
	return nil
}

func (s *fakeStore) Complete(_ context.Context, jobID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if j := s.find(jobID); j != nil {
		j.Status = job.StatusDone
	}
	return nil
}

func (s *fakeStore) Fail(_ context.Context, jobID, msg string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if j := s.find(jobID); j != nil {
		j.Status = job.StatusError
		j.LastError = msg
	}
	return nil
}

func (s *fakeStore) RequeueRunning(context.Context) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	n := 0
	for _, j := range s.jobs {
		if j.Status == job.StatusRunning {
			j.Status = job.StatusPending
			n++
		}
	}
	s.requeued = n
	return n, nil
}

func (s *fakeStore) Get(_ context.Context, jobID string) (job.Job, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if j := s.find(jobID); j != nil {
		return *j, nil
	}
	return job.Job{}, job.ErrNotFound
}

func quietRunner(store job.Store, h Handler) *Runner {
	return NewRunner(store, h, WithLogger(log.New(io.Discard, "", 0)))
}

func TestDrainRunsHandlerAndCompletes(t *testing.T) {
	ctx := context.Background()
	store := &fakeStore{}
	var got []string
	r := quietRunner(store, func(_ context.Context, reviewID string) error {
		got = append(got, reviewID)
		return nil
	})

	j, _ := r.Enqueue(ctx, "review-1")
	r.Drain(ctx)

	if len(got) != 1 || got[0] != "review-1" {
		t.Fatalf("handler calls = %v, want [review-1]", got)
	}
	if final, _ := store.Get(ctx, j.ID); final.Status != job.StatusDone {
		t.Fatalf("job status = %s, want done", final.Status)
	}
}

func TestDrainFailsJobOnHandlerError(t *testing.T) {
	ctx := context.Background()
	store := &fakeStore{}
	r := quietRunner(store, func(context.Context, string) error {
		return errors.New("kaboom")
	})

	j, _ := r.Enqueue(ctx, "review-1")
	r.Drain(ctx)

	final, _ := store.Get(ctx, j.ID)
	if final.Status != job.StatusError || final.LastError != "kaboom" {
		t.Fatalf("job = %+v, want error/kaboom", final)
	}
}

func TestStartRunsJobsConcurrently(t *testing.T) {
	store := &fakeStore{}
	var running, maxRunning int32
	release := make(chan struct{})
	entered := make(chan struct{}, 8)

	r := NewRunner(store, func(context.Context, string) error {
		cur := atomic.AddInt32(&running, 1)
		for {
			m := atomic.LoadInt32(&maxRunning)
			if cur <= m || atomic.CompareAndSwapInt32(&maxRunning, m, cur) {
				break
			}
		}
		entered <- struct{}{}
		<-release
		atomic.AddInt32(&running, -1)
		return nil
	}, WithLogger(log.New(io.Discard, "", 0)), WithConcurrency(2), WithPollInterval(5*time.Millisecond))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	for i := 0; i < 4; i++ {
		_, _ = r.Enqueue(ctx, fmt.Sprintf("r%d", i))
	}
	go r.Start(ctx)

	// Two handlers must be in flight at once for a pool of 2.
	<-entered
	<-entered
	if got := atomic.LoadInt32(&maxRunning); got < 2 {
		t.Fatalf("max concurrent handlers = %d, want >= 2", got)
	}
	close(release) // let the first two finish so the pool drains the rest
	<-entered
	<-entered
}

func TestDrainRecoversFromHandlerPanic(t *testing.T) {
	ctx := context.Background()
	store := &fakeStore{}
	r := quietRunner(store, func(context.Context, string) error {
		panic("boom")
	})

	j, _ := r.Enqueue(ctx, "review-1")
	r.Drain(ctx) // must not crash the process

	final, _ := store.Get(ctx, j.ID)
	if final.Status != job.StatusError || !strings.Contains(final.LastError, "panic: boom") {
		t.Fatalf("job = %+v, want error status with panic message", final)
	}
}

func TestDrainProcessesAllPending(t *testing.T) {
	ctx := context.Background()
	store := &fakeStore{}
	count := 0
	r := quietRunner(store, func(context.Context, string) error {
		count++
		return nil
	})

	for range 3 {
		_, _ = r.Enqueue(ctx, "rev")
	}
	r.Drain(ctx)

	if count != 3 {
		t.Fatalf("handled %d jobs, want 3", count)
	}
}
