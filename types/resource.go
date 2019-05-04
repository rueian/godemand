package types

import "time"

type ResourceState int

const (
	ResourcePending ResourceState = iota
	ResourceCreating
	ResourceRunning
	ResourceDeleting
	ResourceDeleted
	ResourceTerminating
	ResourceTerminated
	ResourceUnknown
	ResourceError
)

type Resource struct {
	ID                  string
	PoolID              string
	Meta                Meta
	State               ResourceState
	StateChange         time.Time
	LastSynced          time.Time
	LastClientHeartbeat time.Time
	Clients             []Client
}
