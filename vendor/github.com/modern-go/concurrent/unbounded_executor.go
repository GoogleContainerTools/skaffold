package concurrent

import (
	"context"
	"fmt"
	"runtime"
	"runtime/debug"
	"sync"
	"time"
	"reflect"
)

// HandlePanic logs goroutine panic by default
var HandlePanic = func(recovered interface{}, file string, line int, funcName string) {
	ErrorLogger.Println(fmt.Sprintf("%s defined at %s:%v panic: %v", funcName, file, line, recovered))
	ErrorLogger.Println(string(debug.Stack()))
}

// StopSignal will not be recovered, will propagate to upper level goroutine
const StopSignal = "STOP!"

// UnboundedExecutor is a executor without limits on counts of alive goroutines
// it tracks the goroutine started by it, and can cancel them when shutdown
type UnboundedExecutor struct {
	ctx                   context.Context
	cancel                context.CancelFunc
	activeGoroutinesMutex *sync.Mutex
	activeGoroutines      map[string]int
	HandlePanic           func(recovered interface{}, file string, line int, funcName string)
}

// GlobalUnboundedExecutor has the life cycle of the program itself
// any goroutine want to be shutdown before main exit can be started from this executor
// GlobalUnboundedExecutor expects the main function to call stop
// it does not magically knows the main function exits
var GlobalUnboundedExecutor = NewUnboundedExecutor()

// NewUnboundedExecutor creates a new UnboundedExecutor,
// UnboundedExecutor can not be created by &UnboundedExecutor{}
// HandlePanic can be set with a callback to override global HandlePanic
func NewUnboundedExecutor() *UnboundedExecutor {
	ctx, cancel := context.WithCancel(context.TODO())
	return &UnboundedExecutor{
		ctx:                   ctx,
		cancel:                cancel,
		activeGoroutinesMutex: &sync.Mutex{},
		activeGoroutines:      map[string]int{},
	}
}

// Go starts a new goroutine and tracks its lifecycle.
// Panic will be recovered and logged automatically, except for StopSignal
func (executor *UnboundedExecutor) Go(handler func(ctx context.Context)) {
	pc := reflect.ValueOf(handler).Pointer()
	f := runtime.FuncForPC(pc)
	funcName := f.Name()
	file, line := f.FileLine(pc)
	executor.activeGoroutinesMutex.Lock()
	defer executor.activeGoroutinesMutex.Unlock()
	startFrom := fmt.Sprintf("%s:%d", file, line)
	executor.activeGoroutines[startFrom] += 1
	go func() {
		defer func() {
			recovered := recover()
			if recovered != nil && recovered != StopSignal {
				if executor.HandlePanic == nil {
					HandlePanic(recovered, file, line, funcName)
				} else {
					executor.HandlePanic(recovered, file, line, funcName)
				}
			}
			executor.activeGoroutinesMutex.Lock()
			defer executor.activeGoroutinesMutex.Unlock()
			executor.activeGoroutines[startFrom] -= 1
		}()
		handler(executor.ctx)
	}()
}

// Stop cancel all goroutines started by this executor without wait
func (executor *UnboundedExecutor) Stop() {
	executor.cancel()
}

// StopAndWaitForever cancel all goroutines started by this executor and
// wait until all goroutines exited
func (executor *UnboundedExecutor) StopAndWaitForever() {
	executor.StopAndWait(context.Background())
}

// StopAndWait cancel all goroutines started by this executor and wait.
// Wait can be cancelled by the context passed in.
func (executor *UnboundedExecutor) StopAndWait(ctx context.Context) {
	executor.cancel()
	for {
		fiveSeconds := time.NewTimer(time.Millisecond * 100)
		select {
		case <-fiveSeconds.C:
		case <-ctx.Done():
			return
		}
		if executor.checkGoroutines() {
			return
		}
	}
}

func (executor *UnboundedExecutor) checkGoroutines() bool {
	executor.activeGoroutinesMutex.Lock()
	defer executor.activeGoroutinesMutex.Unlock()
	for startFrom, count := range executor.activeGoroutines {
		if count > 0 {
			InfoLogger.Println("event!unbounded_executor.still waiting goroutines to quit",
				"startFrom", startFrom,
				"count", count)
			return false
		}
	}
	return true
}
