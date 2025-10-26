package service

import (
	"context"
	"errors"
	"git.myservermanager.com/varakh/ecolinker/internal/locker"
	"git.myservermanager.com/varakh/ecolinker/internal/service_error"
	"time"
)

type LockMemService struct {
	registry *locker.InMemoryLockRegistry
}

var (
	ErrLockMemNotReleased = service_error.NewServiceError(service_error.ErrCodeConflict, errors.New("lock: could not release lock"))
)

func NewLockMemService() LockService {
	return &LockMemService{registry: locker.NewInMemoryLockRegistry()}
}

// Lock locks a given resource without any options (default expiration)
func (s *LockMemService) Lock(ctx context.Context, resource string) (Lock, error) {
	return s.LockWithOptions(ctx, resource, WithLockExpiry(0))
}

// LockWithOptions locks a given resource, only TTL as option is supported
func (s *LockMemService) LockWithOptions(_ context.Context, resource string, options ...LockOption) (Lock, error) {
	if resource == "" {
		return nil, service_error.ErrValidationNotBlank
	}

	var expiration time.Duration = 0
	if options != nil {
		lockOptions := &LockOptions{}
		for _, o := range options {
			o.Apply(lockOptions)
		}

		if lockOptions.expiry != nil {
			expiration = *lockOptions.expiry
		}
	}

	s.registry.LockWithTTL(resource, expiration)

	l := &inMemoryLock{
		registry: s.registry,
		resource: resource,
	}

	return l, nil
}

var _ Lock = (*inMemoryLock)(nil)

type inMemoryLock struct {
	registry *locker.InMemoryLockRegistry
	resource string
}

func (r inMemoryLock) Unlock(_ context.Context) error {
	if err := r.registry.Unlock(r.resource); err != nil {
		return ErrLockMemNotReleased
	}
	return nil
}
