// Package semgroup provides synchronization and error propagation, for groups
// of goroutines working on subtasks of a common task. It uses a weighted
// semaphore implementation to make sure that only a number of maximum tasks
// can be run at any time.
//
// Unlike golang.org/x/sync/errgroup, it doesn't return the first non-nil
// error, rather it accumulates all errors and returns a set of errors,
// allowing each task to fullfil their task.
package semgroup

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	"golang.org/x/sync/semaphore"
)

// A Group is a collection of goroutines working on subtasks that are part of
// the same overall task.
type Group struct {
	sem *semaphore.Weighted
	wg  sync.WaitGroup
	ctx context.Context

	errs MultiError
	mu   sync.Mutex // protects errs
}

// NewGroup returns a new Group with the given maximum combined weight for
// concurrent access.
func NewGroup(ctx context.Context, maxWorkers int64) *Group {
	return &Group{
		ctx: ctx,
		sem: semaphore.NewWeighted(maxWorkers),
	}
}

// Go calls the given function in a new goroutine. It also acquires the
// semaphore with a weight of 1, blocking until resources are available or ctx
// is done.

// On success, returns nil. On failure, returns ctx.Err() and leaves the
// semaphore unchanged. Any function call to return a non-nil error is
// accumulated; the accumulated errors will be returned by Wait.
func (g *Group) Go(f func() error) {
	g.wg.Add(1)

	err := g.sem.Acquire(g.ctx, 1)
	if err != nil {
		g.wg.Done()
		g.mu.Lock()
		g.errs = append(g.errs, fmt.Errorf("couldn't acquire semaphore: %s", err))
		g.mu.Unlock()
		return
	}

	go func() {
		defer g.sem.Release(1)
		defer g.wg.Done()

		if err := f(); err != nil {
			g.mu.Lock()
			g.errs = append(g.errs, err)
			g.mu.Unlock()
		}
	}()
}

// Wait blocks until all function calls from the Go method have returned, then
// returns all accumulated non-nil errors (if any) from them.
//
// If a non-nil error is returned, it will be of type [MultiError].
func (g *Group) Wait() error {
	g.wg.Wait()
	return g.errs.ErrorOrNil()
}

type MultiError []error

func (e MultiError) Error() string {
	var b strings.Builder
	fmt.Fprintf(&b, "%d error(s) occurred:\n", len(e))

	for i, err := range e {
		fmt.Fprintf(&b, "* %s", err.Error())
		if i != len(e)-1 {
			fmt.Fprintln(&b, "")
		}
	}

	return b.String()
}

// ErrorOrNil returns nil if there are no errors, otherwise returns itself.
func (e MultiError) ErrorOrNil() error {
	if len(e) == 0 {
		return nil
	}

	return e
}

func (e MultiError) Is(target error) bool {
	for _, err := range e {
		if errors.Is(err, target) {
			return true
		}
	}
	return false
}

func (e MultiError) As(target interface{}) bool {
	for _, err := range e {
		if errors.As(err, target) {
			return true
		}
	}
	return false
}
