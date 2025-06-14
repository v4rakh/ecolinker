//go:build !integration

package service

import (
	"context"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

const (
	testLockName = "test_lock"
)

func TestLockExpiresAndCannotBeReleased(t *testing.T) {
	a := assert.New(t)

	s := NewLockMemService()
	ctx := context.Background()

	lock, lockErr := s.LockWithOptions(ctx, testLockName, WithLockExpiry(250*time.Millisecond))
	a.Nil(lockErr)
	a.NotNil(lock)

	time.Sleep(251 * time.Millisecond)

	unlockErr := lock.Unlock(ctx)
	a.NotNil(unlockErr)
	a.ErrorContains(unlockErr, "could not release lock")
}
