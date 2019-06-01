package redis

import (
	"math/rand"
	"strconv"
	"time"

	"github.com/go-redis/redis"
	"github.com/rueian/godemand/plugin"
	"golang.org/x/xerrors"
)

type LockerOptionFunc func(*Locker)

func WithExpiration(expiration time.Duration) LockerOptionFunc {
	return func(locker *Locker) {
		locker.expiration = expiration
	}
}

func NewLocker(client redis.UniversalClient, options ...LockerOptionFunc) *Locker {
	s := &Locker{
		client:     client,
		expiration: time.Minute,
	}

	for _, of := range options {
		of(s)
	}

	return s
}

type Locker struct {
	client     redis.UniversalClient
	expiration time.Duration
}

func (l *Locker) AcquireLock(key string) (id string, err error) {
	id = strconv.Itoa(rand.Int())
	ok, err := l.client.SetNX(lockKey(key), id, l.expiration).Result()
	if err != nil {
		return "", err
	}
	if !ok {
		return "", xerrors.Errorf("fail to acquire lock on %s: %w", key, plugin.AcquireLaterErr)
	}
	return id, nil
}

func (l *Locker) ReleaseLock(key, id string) error {
	removed, err := releaseScript.Run(l.client, []string{lockKey(key)}, id).Result()
	if err != nil {
		return err
	}
	if removed.(int64) == 0 {
		return xerrors.Errorf("fail to release lock by (%s, %s): %w", key, id, plugin.LockNotFoundErr)
	}
	return nil
}

func lockKey(key string) string {
	return key + ":lock"
}

var releaseScript = redis.NewScript(`
if redis.call("get",KEYS[1]) == ARGV[1]
then
    return redis.call("del",KEYS[1])
else
    return 0
end`)
