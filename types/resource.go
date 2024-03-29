package types

import (
	"errors"
	"time"
)

type ResourceState int

const (
	ResourcePending ResourceState = iota
	ResourceBooting
	ResourceServing
	ResourceDeleting
	ResourceDeleted
	ResourceTerminating
	ResourceTerminated
	ResourceUnknown
	ResourceError
)

func (s ResourceState) String() string {
	switch s {
	case ResourcePending:
		return "pending"
	case ResourceBooting:
		return "booting"
	case ResourceServing:
		return "serving"
	case ResourceDeleting:
		return "deleting"
	case ResourceDeleted:
		return "deleted"
	case ResourceTerminating:
		return "terminating"
	case ResourceTerminated:
		return "terminated"
	case ResourceUnknown:
		return "unknown"
	case ResourceError:
		return "error"
	}
	return "unknown"
}

var ResourceStates = []ResourceState{
	ResourcePending,
	ResourceBooting,
	ResourceServing,
	ResourceDeleting,
	ResourceDeleted,
	ResourceTerminating,
	ResourceTerminated,
	ResourceUnknown,
	ResourceError,
}

type ResourcePool struct {
	ID        string
	Resources map[string]Resource
}

type Resource struct {
	ID                  string
	PoolID              string
	Meta                Meta
	Config              Meta
	State               ResourceState
	StateChange         time.Time
	CreatedAt           time.Time
	LastSynced          time.Time
	LastClientHeartbeat time.Time
	Clients             map[string]Client
}

type ResourceEvent struct {
	ResourceID     string
	ResourcePoolID string
	Meta           Meta
	Timestamp      time.Time
}

type Client struct {
	ID         string
	CreatedAt  time.Time
	Heartbeat  time.Time
	Meta       Meta
	PoolConfig Meta
}

type Meta map[string]interface{}

type ResourceDAO interface {
	GetResource(pool, id string) (Resource, error)
	GetResources(id string) (ResourcePool, error)
	SaveResource(resource Resource) (Resource, error)
	DeleteResource(resource Resource) error
	SaveClient(resource Resource, client Client) (Client, error)
	DeleteClients(resource Resource, clients []Client) error
	AppendEvent(event ResourceEvent) error
	GetEventsByPool(id string, limit int, before time.Time) ([]ResourceEvent, error)
	GetEventsByResource(poolID, id string, limit int, before time.Time) ([]ResourceEvent, error)
}

var ResourceNotFoundErr = errors.New("resource not found in pool")

func Merge(a, b map[string]interface{}) map[string]interface{} {
	out := make(map[string]interface{})
	for k, v := range a {
		out[k] = v
	}
	for k, v := range b {
		out[k] = v
	}
	return out
}
