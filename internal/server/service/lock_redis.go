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
	"github.com/rs/zerolog/log"
)

type LockRedisService struct {
	rs *redsync.Redsync
}

var (
	errLockRedisNotObtained = service_error.NewServiceError(service_error.ErrCodeConflict, errors.New("lock service: could not obtain Lock"))
	errLockRedisNotReleased = service_error.NewServiceError(service_error.ErrCodeConflict, errors.New("lock service: could not release Lock"))
)

func NewLockRedisService(lc *config.Lock) (LockService, error) {
	log.Info().Msg("Initializing REDIS locking service")

	var err error
	var c *redis.Client
	if c, err = config.NewRedisClient(fmt.Sprintf("%s-lock", app.Name), lc.RedisUrl); err != nil {
		return nil, fmt.Errorf("lock service: cannot initialize REDIS client: %w", err)
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

	log.Debug().Msgf("Trying to Lock '%s'", resource)

	if err := mu.LockContext(ctx); err != nil {
		return nil, errLockRedisNotObtained
	}

	log.Debug().Msgf("Locked '%s'", resource)

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
	log.Debug().Msgf("Unlocking '%s'", r.mu.Name())

	unlocked, err := r.mu.UnlockContext(ctx)
	if err != nil {
		return errLockRedisNotReleased
	}
	if !unlocked {
		return errLockRedisNotReleased
	}

	return nil
}
