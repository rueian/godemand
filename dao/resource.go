package dao

import (
	"time"

	"github.com/rueian/godemand/types"
	"golang.org/x/xerrors"
)

var (
	PoolNotFoundErr = xerrors.New("pool not found")
)

type ResourceDAO interface {
	AddResourcePool(id string) (types.ResourcePool, error)
	GetResourcePool(id string) (types.ResourcePool, error)
	SaveResource(resource types.Resource) (types.Resource, error)
	DeleteResource(resource types.Resource) error
	AppendResourceEvent(event types.ResourceEvent) error
	GetEventsByPool(id string, limit int, before time.Time) ([]types.ResourceEvent, error)
	GetEventsByResource(poolID, id string, limit int, before time.Time) ([]types.ResourceEvent, error)
}

type InMemoryResourceStoreOptionFunc func(*InMemoryResourceStore)

func NewInMemoryResourceStore(options ...InMemoryResourceStoreOptionFunc) *InMemoryResourceStore {
	s := &InMemoryResourceStore{
		pools:             make(map[string]types.ResourcePool),
		events:            make(map[string][]types.ResourceEvent),
		eventLimitPerPool: 1000,
	}

	for _, of := range options {
		of(s)
	}

	return s
}

func WithEventLimitPerPool(limit int) InMemoryResourceStoreOptionFunc {
	return func(store *InMemoryResourceStore) {
		store.eventLimitPerPool = limit
	}
}

type InMemoryResourceStore struct {
	pools             map[string]types.ResourcePool
	events            map[string][]types.ResourceEvent
	eventLimitPerPool int
}

func (s *InMemoryResourceStore) AddResourcePool(id string) (types.ResourcePool, error) {
	if pool, ok := s.pools[id]; ok {
		return pool, nil
	}
	s.pools[id] = types.ResourcePool{
		ID:        id,
		Resources: make(map[string]types.Resource),
	}
	s.events[id] = nil
	return s.pools[id], nil
}

func (s *InMemoryResourceStore) GetResourcePool(id string) (types.ResourcePool, error) {
	if pool, ok := s.pools[id]; ok {
		return pool, nil
	}
	return types.ResourcePool{}, xerrors.Errorf("fail to get resource pool %s: %w", id, PoolNotFoundErr)
}

func (s *InMemoryResourceStore) SaveResource(resource types.Resource) (types.Resource, error) {
	pool, err := s.GetResourcePool(resource.PoolID)
	if err != nil {
		return types.Resource{}, err
	}
	pool.Resources[resource.ID] = resource
	return resource, nil
}

func (s *InMemoryResourceStore) DeleteResource(resource types.Resource) error {
	pool, err := s.GetResourcePool(resource.PoolID)
	if err != nil {
		return err
	}
	delete(pool.Resources, resource.ID)
	return nil
}

func (s *InMemoryResourceStore) AppendResourceEvent(event types.ResourceEvent) error {
	events, ok := s.events[event.ResourcePoolID]
	if !ok {
		return xerrors.Errorf("fail to append event %v: %w", event, PoolNotFoundErr)
	}
	if len(events) >= s.eventLimitPerPool {
		s.events[event.ResourcePoolID] = append(events[1:], event)
	} else {
		s.events[event.ResourcePoolID] = append(events, event)
	}

	return nil
}

func (s *InMemoryResourceStore) GetEventsByPool(id string, limit int, before time.Time) (result []types.ResourceEvent, err error) {
	events, ok := s.events[id]
	if !ok {
		return nil, xerrors.Errorf("fail to get event by pool %s: %w", id, PoolNotFoundErr)
	}

	count := 0
	for i := len(events) - 1; i >= 0 && count < limit; i-- {
		e := events[i]
		if e.Timestamp.Before(before) {
			result = append(result, e)
			count++
		}
	}
	return result, nil
}

func (s *InMemoryResourceStore) GetEventsByResource(poolID, id string, limit int, before time.Time) (result []types.ResourceEvent, err error) {
	events, ok := s.events[poolID]
	if !ok {
		return nil, xerrors.Errorf("fail to get event by pool %s: %w", id, PoolNotFoundErr)
	}

	count := 0
	for i := len(events) - 1; i >= 0 && count < limit; i-- {
		e := events[i]
		if e.Timestamp.Before(before) && e.ResourceID == id {
			result = append(result, e)
			count++
		}
	}
	return result, nil
}
