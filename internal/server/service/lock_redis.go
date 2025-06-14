package service

import (
	"context"
	"errors"
	"fmt"
	"git.myservermanager.com/varakh/ecolinker/internal/app"
	"git.myservermanager.com/varakh/ecolinker/internal/server/config"
	"git.myservermanager.com/varakh/ecolinker/internal/server/service_error"
	"github.com/go-redsync/redsync/v4"
	redsyncgoredis "github.com/go-redsync/redsync/v4/redis/goredis/v9"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

type LockRedisService struct {
	rs *redsync.Redsync
}

var (
	errLockRedisNotObtained = service_error.NewServiceError(service_error.ErrCodeConflict, errors.New("lock service: could not obtain Lock"))
	errLockRedisNotReleased = service_error.NewServiceError(service_error.ErrCodeConflict, errors.New("lock service: could not release Lock"))
)

func NewLockRedisService(lc *config.Lock) (LockService, error) {
	zap.L().Info("Initializing REDIS locking service")

	var err error
	var redisOptions *redis.Options
	if redisOptions, err = redis.ParseURL(lc.RedisUrl); err != nil {
		return nil, fmt.Errorf("lock service: cannot parse REDIS URL '%s' to set up locking: %s", lc.RedisUrl, err)
	}
	redisOptions.ClientName = fmt.Sprintf("%s-lock", app.Name)

	c := redis.NewClient(redisOptions)
	if err = c.Ping(context.Background()).Err(); err != nil {
		return nil, fmt.Errorf("lock service: failed to connect to REDIS: %w", err)
	}

	pool := redsyncgoredis.NewPool(c)
	rs := redsync.New(pool)

	return &LockRedisService{rs: rs}, nil
}

// Lock locks a given resource without any options
func (s *LockRedisService) Lock(ctx context.Context, resource string) (Lock, error) {
	return s.LockWithOptions(ctx, resource, nil)
}

// LockWithOptions locks a given resource considering all options
func (s *LockRedisService) LockWithOptions(ctx context.Context, resource string, options ...LockOption) (Lock, error) {
	if resource == "" {
		return nil, service_error.ErrValidationNotBlank
	}

	var rsOptions []redsync.Option

	if options != nil {
		lockOptions := &LockOptions{}
		for _, o := range options {
			o.Apply(lockOptions)
		}

		if lockOptions.expiry != nil {
			rsOptions = append(rsOptions, redsync.WithExpiry(*lockOptions.expiry))
		}
		if lockOptions.maxRetries != nil {
			rsOptions = append(rsOptions, redsync.WithTries(*lockOptions.maxRetries))
		}
		if lockOptions.retryDelay != nil {
			rsOptions = append(rsOptions, redsync.WithRetryDelay(*lockOptions.retryDelay))
		}
	}

	mu := s.rs.NewMutex(resource, rsOptions...)

	zap.L().Sugar().Debugf("Trying to Lock '%s'", resource)

	if err := mu.LockContext(ctx); err != nil {
		return nil, errLockRedisNotObtained
	}

	zap.L().Sugar().Debugf("Locked '%s'", resource)

	l := &redisLock{
		mu: mu,
	}

	return l, nil
}

var _ Lock = (*redisLock)(nil)

type redisLock struct {
	mu *redsync.Mutex
}

func (r redisLock) Unlock(ctx context.Context) error {
	zap.L().Sugar().Debugf("Unlocking '%s'", r.mu.Name())

	unlocked, err := r.mu.UnlockContext(ctx)
	if err != nil {
		return errLockRedisNotReleased
	}
	if !unlocked {
		return errLockRedisNotReleased
	}

	return nil
}
