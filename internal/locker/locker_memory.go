package locker

import (
	"errors"
	"sync"
	"sync/atomic"
	"time"
)

const (
	MemoryLockerDefaultExpiration time.Duration = 0
	MemoryLockerNoExpiration      time.Duration = -1
)

// ErrNoSuchLock is returned when the requested lock does not exist
var ErrNoSuchLock = errors.New("no such lock")

// InMemoryLockRegistry provides a locking mechanism based on the passed in reference name
type InMemoryLockRegistry struct {
	mu            sync.Mutex
	locks         map[string]*lockCtr
	defaultExpiry int64
}

// lockCtr is used by InMemoryLockRegistry to represent a lock with a given name.
type lockCtr struct {
	mu sync.Mutex
	// waiters is the number of waiters waiting to acquire the lock
	// this is int32 instead of uint32, so we can add `-1` in `dec()`
	waiters int32
	// expires is the time when
	expires int64
}

// inc increments the number of waiters waiting for the lock
func (l *lockCtr) inc() {
	atomic.AddInt32(&l.waiters, 1)
}

// dec decrements the number of waiters waiting on the lock
func (l *lockCtr) dec() {
	atomic.AddInt32(&l.waiters, -1)
}

// count gets the current number of waiters
func (l *lockCtr) count() int32 {
	return atomic.LoadInt32(&l.waiters)
}

// Lock locks the mutex
func (l *lockCtr) Lock() {
	l.mu.Lock()
}

// Unlock unlocks the mutex
func (l *lockCtr) Unlock() {
	l.mu.Unlock()
}

// NewInMemoryLockRegistry creates a new InMemoryLockRegistry
func NewInMemoryLockRegistry() *InMemoryLockRegistry {
	return &InMemoryLockRegistry{
		defaultExpiry: MemoryLockerNoExpiration.Milliseconds(),
		locks:         make(map[string]*lockCtr),
	}
}

// Clear clears all locks by initializing a new map
func (r *InMemoryLockRegistry) Clear() {
	r.locks = make(map[string]*lockCtr)
}

// Exists exists a lock by name
func (r *InMemoryLockRegistry) Exists(name string) bool {
	r.deleteExpired()
	r.mu.Lock()
	_, exists := r.locks[name]
	r.mu.Unlock()
	return exists
}

// Lock locks a mutex with the given name and no expiration. If it doesn't exist, one is created.
func (r *InMemoryLockRegistry) Lock(name string) {
	r.LockWithTTL(name, MemoryLockerDefaultExpiration)
}

// LockWithTTL locks a mutex with the given name and duration. If it doesn't exist, one is created. If duration is greater than 0, expiration is added.
func (r *InMemoryLockRegistry) LockWithTTL(name string, duration time.Duration) {
	r.deleteExpired()

	r.mu.Lock()
	if r.locks == nil {
		r.locks = make(map[string]*lockCtr)
	}

	nameLock, exists := r.locks[name]
	if !exists {
		e := r.defaultExpiry
		if duration > 0 {
			e = time.Now().Add(duration).UnixNano()
		}

		nameLock = &lockCtr{expires: e}

		r.locks[name] = nameLock
	}

	// increment the nameLock waiters while inside the main mutex
	// this makes sure that the lock isn't deleted if `Lock` and `Unlock` are called concurrently
	nameLock.inc()
	r.mu.Unlock()

	// Lock the nameLock outside the main mutex, so we don't block other operations
	// once locked then we can decrement the number of waiters for this lock
	nameLock.Lock()
	nameLock.dec()
}

// Unlock unlocks the mutex with the given name
// If the given lock is not being waited on by any other callers, it is deleted
func (r *InMemoryLockRegistry) Unlock(name string) error {
	r.deleteExpired()

	r.mu.Lock()
	nameLock, exists := r.locks[name]
	if !exists {
		r.mu.Unlock()
		return ErrNoSuchLock
	}

	if nameLock.count() == 0 {
		delete(r.locks, name)
	}
	nameLock.Unlock()

	r.mu.Unlock()
	return nil
}

// deleteExpired deletes expired entries if their expiration value is greater than 0 (expiration enabled) and it expired. This is a costly operation and is guarded by the global registry mutex.
func (r *InMemoryLockRegistry) deleteExpired() {
	now := time.Now().UnixNano()
	r.mu.Lock()
	for k, v := range r.locks {
		if v.expires > 0 && now > v.expires {
			nameLock, exists := r.locks[k]

			if !exists {
				continue
			}

			if nameLock.count() == 0 {
				delete(r.locks, k)
			}
			nameLock.Unlock()
		}
	}
	r.mu.Unlock()
}
