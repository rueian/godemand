package types

import "time"

type ResourceEvent struct {
	ResourceID     string
	ResourcePoolID string
	Meta           Meta
	Timestamp      time.Time
}
