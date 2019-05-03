package dao

import (
	"sync"
	"time"

	"github.com/rueian/godemand/types"
)

type ResourceDAO interface {
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
	mu                sync.RWMutex
	pools             map[string]types.ResourcePool
	events            map[string][]types.ResourceEvent
	eventLimitPerPool int
}

func (s *InMemoryResourceStore) GetResourcePool(id string) (types.ResourcePool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if pool, ok := s.pools[id]; ok {
		return pool, nil
	}
	return types.ResourcePool{ID: id, Resources: map[string]types.Resource{}}, nil
}

func (s *InMemoryResourceStore) SaveResource(resource types.Resource) (types.Resource, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.pools[resource.PoolID]; !ok {
		s.pools[resource.PoolID] = types.ResourcePool{ID: resource.PoolID, Resources: map[string]types.Resource{}}
	}
	s.pools[resource.PoolID].Resources[resource.ID] = resource
	return resource, nil
}

func (s *InMemoryResourceStore) DeleteResource(resource types.Resource) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if pool, ok := s.pools[resource.PoolID]; ok {
		delete(pool.Resources, resource.ID)
	}
	return nil
}

func (s *InMemoryResourceStore) AppendResourceEvent(event types.ResourceEvent) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	events, ok := s.events[event.ResourcePoolID]
	if !ok {
		events = make([]types.ResourceEvent, 1)
	}
	if len(events) >= s.eventLimitPerPool {
		s.events[event.ResourcePoolID] = append(events[1:], event)
	} else {
		s.events[event.ResourcePoolID] = append(events, event)
	}

	return nil
}

func (s *InMemoryResourceStore) GetEventsByPool(id string, limit int, before time.Time) (result []types.ResourceEvent, err error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	events, ok := s.events[id]
	if !ok {
		return nil, nil
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
	s.mu.RLock()
	defer s.mu.RUnlock()
	events, ok := s.events[poolID]
	if !ok {
		return nil, nil
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
