package plugin

import (
	"github.com/rueian/godemand/types"
)

type Controller interface {
	FindResource(pool types.ResourcePool, params map[string]interface{}) (types.Resource, error)
	SyncResource(resource types.Resource, params map[string]interface{}) (types.Resource, error)
}
