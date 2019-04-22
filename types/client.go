package types

import "time"

type Client struct {
	ID        string
	Heartbeat time.Time
	Meta      Meta
}
