//go:build !integration

package locker

import (
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

const (
	testLockName = "test_lock"
)

func TestLockExpires(t *testing.T) {
	a := assert.New(t)

	r := NewInMemoryLockRegistry()
	r.LockWithTTL(testLockName, 250*time.Millisecond)
	a.True(r.Exists(testLockName))

	time.Sleep(251 * time.Millisecond)
	a.False(r.Exists(testLockName))
}

func TestLockNeverExpires(t *testing.T) {
	a := assert.New(t)

	r := NewInMemoryLockRegistry()
	r.Lock(testLockName)
	a.True(r.Exists(testLockName))

	time.Sleep(2 * time.Second)
	a.True(r.Exists(testLockName))
}

func TestLockLocksAndUnlocks(t *testing.T) {
	a := assert.New(t)

	r := NewInMemoryLockRegistry()
	r.LockWithTTL(testLockName, 250*time.Millisecond)
	a.True(r.Exists(testLockName))
	_ = r.Unlock(testLockName)
	a.False(r.Exists(testLockName))
}
