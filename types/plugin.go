package types

type CmdParam struct {
	Name string
	Path string
	Envs []string
}

//go:generate mockgen -destination=mock/controller.go -package=mock github.com/rueian/godemand/types Controller
type Controller interface {
	FindResource(pool ResourcePool, params map[string]interface{}) (Resource, error)
	SyncResource(resource Resource, params map[string]interface{}) (Resource, error)
}

//go:generate mockgen -destination=mock/launchpad.go -package=mock github.com/rueian/godemand/types Launchpad
type Launchpad interface {
	SetLaunchers(params map[string]CmdParam) error
	GetController(name string) (controller Controller, err error)
	Close()
}
