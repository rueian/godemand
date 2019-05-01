package types

//go:generate mockgen -destination=mock/locker.go -package=mock github.com/rueian/godemand/types Locker
type Locker interface {
	AcquireLock(key string) (id string, err error)
	ReleaseLock(key, id string) error
}
