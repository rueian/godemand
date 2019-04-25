package dao

import (
	"math/rand"
	"strconv"
	"sync"
	"time"

	"golang.org/x/xerrors"
)

var (
	AcquireLaterErr = xerrors.New("please acquire later")
	LockNotFoundErr = xerrors.New("lock not found")
)

type Locker interface {
	AcquireLock(key string) (id string, err error)
	ReleaseLock(key, id string) error
}

func NewInMemoryLocker() *InMemoryLocker {
	return &InMemoryLocker{
		seed: rand.NewSource(time.Now().Unix()),
	}
}

type InMemoryLocker struct {
	muMap sync.Map
	seed  rand.Source
}

func (l *InMemoryLocker) AcquireLock(key string) (string, error) {
	id := strconv.Itoa(rand.Int())
	if _, loaded := l.muMap.LoadOrStore(key, id); loaded {
		return "", xerrors.Errorf("fail to acquire lock on %s: %w", key, AcquireLaterErr)
	} else {
		return id, nil
	}
}

func (l *InMemoryLocker) ReleaseLock(key, id string) error {
	v, ok := l.muMap.Load(key)
	if !ok || v.(string) != id {
		return xerrors.Errorf("fail to release lock by (%s, %s): %w", key, id, LockNotFoundErr)
	}
	l.muMap.Delete(key)
	return nil
}
