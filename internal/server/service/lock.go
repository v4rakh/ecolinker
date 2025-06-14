package service

import (
	"context"
	"math"
	"time"
)

var (
	lockOptionMaxRetries = math.MaxInt32
)

// LockService provides methods for locking resources, behavior depends on underlying implementation
type LockService interface {
	// Lock locks a resource applying default options (varies for implementations)
	Lock(ctx context.Context, resource string) (Lock, error)

	// LockWithOptions locks a resource with given options, not all options are applied (varies for implementations)
	LockWithOptions(ctx context.Context, resource string, options ...LockOption) (Lock, error)
}

type Lock interface {
	// Unlock unlocks a Lock
	Unlock(ctx context.Context) error
}

type LockOption interface {
	Apply(l *LockOptions)
}

type LockOptionFunc func(o *LockOptions)

func (f LockOptionFunc) Apply(o *LockOptions) {
	f(o)
}

type LockOptions struct {
	expiry     *time.Duration
	retryDelay *time.Duration
	maxRetries *int
}

func WithLockExpiry(expiry time.Duration) LockOption {
	return LockOptionFunc(func(o *LockOptions) {
		o.expiry = &expiry
	})
}

func WithLockRetries(retries int) LockOption {
	return LockOptionFunc(func(o *LockOptions) {
		o.maxRetries = &retries
	})
}

func WithLockInfiniteRetries() LockOption {
	return LockOptionFunc(func(o *LockOptions) {
		o.maxRetries = &lockOptionMaxRetries
	})
}

func WithLockRetryDelay(retryDelay time.Duration) LockOption {
	return LockOptionFunc(func(o *LockOptions) {
		o.retryDelay = &retryDelay
	})
}
