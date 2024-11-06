package cronjob

import (
	"context"
	"fmt"
	"time"

	"github.com/avast/retry-go/v4"
	"github.com/go-co-op/gocron/v2"
	"github.com/ipfans/components/v2/lifecycle"
	"github.com/ipfans/components/v2/utils"
	"github.com/jonboulle/clockwork"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var (
	Clock      clockwork.Clock = clockwork.NewRealClock()
	LeaderKey                  = "cronjob:leader"
	DefaultTTL                 = 10 * time.Minute
)

type DistributedElector struct {
	cmder      redis.Cmdable
	currentID  string
	defaultTTL time.Duration
	isLeader   bool
}

func NewDistributedElector(cmder redis.Cmdable) *DistributedElector {
	currentID := utils.NewUUID()
	selector := &DistributedElector{
		currentID: currentID,
		cmder:     cmder,
	}
	err := selector.lockLeader()
	log.Error().Err(err).Msg("DistributedElector leader failed.")
	return selector
}

func (e *DistributedElector) lockLeader() error {
	var err error
	e.isLeader, err = retry.DoWithData(func() (bool, error) {
		return e.cmder.SetNX(context.TODO(), LeaderKey, e.currentID, 10*time.Minute).Result()
	}, retry.Attempts(3), retry.Delay(time.Second/4))
	return err
}

func (e *DistributedElector) IsLeader(ctx context.Context) error {
	if e.isLeader {
		if e.cmder.Exists(ctx, LeaderKey).Val() == 0 {
			_ = e.lockLeader()
		} else {
			e.isLeader = e.cmder.Get(ctx, LeaderKey).Val() == e.currentID
		}
	}

	if !e.isLeader {
		return fmt.Errorf("not leader")
	}
	return nil
}

func (e *DistributedElector) Stop(ctx context.Context) {
	if e.IsLeader(ctx) == nil {
		_ = retry.Do(func() error {
			return e.cmder.Del(ctx, LeaderKey).Err()
		}, retry.Attempts(3), retry.Delay(time.Second/4))
	}
}

func (e *DistributedElector) Lock(ctx context.Context, key string) (gocron.Lock, error) {
	err := retry.Do(func() error {
		return e.cmder.SetNX(ctx, key, e.currentID, e.defaultTTL).Err()
	}, retry.Attempts(3), retry.Delay(time.Second/4))
	if err != nil {
		return nil, err
	}
	return &unlocker{key: key, e: e}, nil
}

func (e *DistributedElector) Unlock(ctx context.Context, key string) error {
	return retry.Do(func() error {
		return e.cmder.Del(ctx, key).Err()
	}, retry.Attempts(3), retry.Delay(time.Second/4))
}

type unlocker struct {
	e   *DistributedElector
	key string
}

func (l *unlocker) Unlock(ctx context.Context) error {
	return l.e.Unlock(ctx, l.key)
}

type logger struct {
	zerolog.Logger
}

func (l *logger) Debug(msg string, args ...any) {
	l.Logger.Debug().Msgf(msg, args...)
}

func (l *logger) Error(msg string, args ...any) {
	l.Logger.Error().Msgf(msg, args...)
}

func (l *logger) Info(msg string, args ...any) {
	l.Logger.Info().Msgf(msg, args...)
}

func (l *logger) Warn(msg string, args ...any) {
	l.Logger.Warn().Msgf(msg, args...)
}

func New(lc lifecycle.Lifecycle, cmder redis.Cmdable) (gocron.Scheduler, error) {
	elector := &DistributedElector{cmder: cmder, defaultTTL: DefaultTTL}
	scheduler, err := gocron.NewScheduler(
		gocron.WithClock(Clock),
		gocron.WithDistributedElector(elector),
		gocron.WithDistributedLocker(elector),
		gocron.WithLocation(time.UTC),
		gocron.WithLogger(&logger{log.Logger}),
	)
	if err == nil {
		lc.Append(lifecycle.Hook{
			OnStart: func(ctx context.Context) error {
				scheduler.Start()
				return nil
			},
			OnStop: func(ctx context.Context) error {
				elector.Stop(ctx)
				return scheduler.Shutdown()
			},
		})
	}
	return scheduler, nil
}
