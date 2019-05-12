package types

import "time"

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

type ResourcePool struct {
	ID        string
	Resources map[string]Resource
}

type Resource struct {
	ID                  string
	PoolID              string
	Meta                Meta
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
	ID        string
	Heartbeat time.Time
	Meta      Meta
}

type Meta map[string]interface{}

type ResourceDAO interface {
	GetResources(id string) (ResourcePool, error)
	SaveResource(resource Resource) (Resource, error)
	DeleteResource(resource Resource) error
	SaveClient(resource Resource, client Client) (Client, error)
	DeleteClients(resource Resource, clients []Client) error
	AppendEvent(event ResourceEvent) error
	GetEventsByPool(id string, limit int, before time.Time) ([]ResourceEvent, error)
	GetEventsByResource(poolID, id string, limit int, before time.Time) ([]ResourceEvent, error)
}
