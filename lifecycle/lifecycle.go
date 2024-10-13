package lifecycle

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sourcegraph/conc/pool"
)

type Lifecycle interface {
	Append(Hook)
}

type Hook struct {
	OnStart func(context.Context) error
	OnStop  func(context.Context) error
}

type (
	ContextFunc      func(context.Context)
	ContextErrorFunc func(context.Context) error
)

type HookFunc interface {
	~func(context.Context) | ~func(context.Context) error
}

func wrap[T HookFunc](fn T) ContextErrorFunc {
	if fn == nil {
		return nil
	}
	switch any(fn).(type) {
	case func(context.Context):
		return func(ctx context.Context) error {
			any(fn).(func(context.Context))(ctx)
			return nil
		}
	case func(context.Context) error:
		return any(fn).(func(context.Context) error)
	}
	return nil
}

func StartHook[T HookFunc](start T) Hook {
	return Hook{
		OnStart: wrap(start),
	}
}

func StopHook[T HookFunc](stop T) Hook {
	return Hook{
		OnStop: wrap(stop),
	}
}

type lifecycleWrapper struct {
	startHooks []Hook
	stopHooks  []Hook
	timeout    time.Duration
}

type Option func(*lifecycleWrapper)

func WithTimeout(timeout time.Duration) Option {
	return func(l *lifecycleWrapper) {
		l.timeout = timeout
	}
}

func New(opts ...Option) *lifecycleWrapper {
	wrapper := &lifecycleWrapper{
		startHooks: []Hook{},
		stopHooks:  []Hook{},
		timeout:    6 * time.Second,
	}
	for _, opt := range opts {
		opt(wrapper)
	}
	return wrapper
}

func (l *lifecycleWrapper) Append(hook Hook) {
	if hook.OnStart != nil {
		l.startHooks = append(l.startHooks, hook)
	}
	if hook.OnStop != nil {
		l.stopHooks = append(l.stopHooks, hook)
	}
}

// Start starts the lifecycle.
func (l *lifecycleWrapper) Start(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, l.timeout)
	defer cancel()

	wg := pool.New().WithErrors()
	for _, hook := range l.startHooks {
		hook := hook
		wg.Go(func() error {
			return hook.OnStart(ctx)
		})
	}
	return wg.Wait()
}

// Stop stops the lifecycle.
func (l *lifecycleWrapper) Stop(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, l.timeout)
	defer cancel()

	wg := pool.New().WithErrors()
	for _, hook := range l.stopHooks {
		hook := hook
		wg.Go(func() error {
			return hook.OnStop(ctx)
		})
	}
	return wg.Wait()
}

// WaitStop waits for an interrupt signal and stops the lifecycle.
func (l *lifecycleWrapper) WaitStop(ctx context.Context) error {
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
	<-signalChan

	return l.Stop(ctx)
}
