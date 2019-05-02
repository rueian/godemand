package types

//go:generate mockgen -destination=mock/service.go -package=mock github.com/rueian/godemand/types Service
type Service interface {
	RequestResource(poolID string, client Client) (res Resource, err error)
	GetResource(poolID, id string) (res Resource, err error)
	Heartbeat(poolID, id string, client Client) (err error)
}
