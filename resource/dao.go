package resource

import (
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/rueian/godemand/types"
)

type InMemoryResourcePoolOptionFunc func(*InMemoryResourcePool)

func NewInMemoryResourcePool(options ...InMemoryResourcePoolOptionFunc) *InMemoryResourcePool {
	s := &InMemoryResourcePool{
		pools:             make(map[string]types.ResourcePool),
		events:            make(map[string][]types.ResourceEvent),
		eventLimitPerPool: 1000,
	}

	for _, of := range options {
		of(s)
	}

	return s
}

func WithEventLimitPerPool(limit int) InMemoryResourcePoolOptionFunc {
	return func(store *InMemoryResourcePool) {
		store.eventLimitPerPool = limit
	}
}

type InMemoryResourcePool struct {
	mu                sync.RWMutex
	pools             map[string]types.ResourcePool
	events            map[string][]types.ResourceEvent
	eventLimitPerPool int
}

func (s *InMemoryResourcePool) GetResources(id string) (types.ResourcePool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if pool, ok := s.pools[id]; ok {
		return copyPool(pool)
	}
	return types.ResourcePool{ID: id, Resources: map[string]types.Resource{}}, nil
}

func (s *InMemoryResourcePool) SaveResource(resource types.Resource) (types.Resource, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	cp, err := copyResource(resource)
	if err != nil {
		return types.Resource{}, err
	}

	if _, ok := s.pools[cp.PoolID]; !ok {
		s.pools[cp.PoolID] = types.ResourcePool{ID: cp.PoolID, Resources: map[string]types.Resource{}}
	}

	current, ok := s.pools[cp.PoolID].Resources[cp.ID]
	if !ok {
		current = types.Resource{}
	}
	if current.Clients == nil {
		current.Clients = map[string]types.Client{}
	}

	cp.Clients = current.Clients
	cp.LastClientHeartbeat = current.LastClientHeartbeat
	if cp.StateChange.IsZero() {
		cp.StateChange = time.Now()
	}
	if current.State != cp.State && current.StateChange == cp.StateChange {
		cp.StateChange = time.Now()
	}

	s.pools[cp.PoolID].Resources[cp.ID] = cp
	return cp, nil
}

func (s *InMemoryResourcePool) DeleteResource(resource types.Resource) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if pool, ok := s.pools[resource.PoolID]; ok {
		delete(pool.Resources, resource.ID)
	}
	return nil
}

func (s *InMemoryResourcePool) SaveClient(resource types.Resource, client types.Client) (types.Client, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	pool, ok := s.pools[resource.PoolID]
	if !ok {
		return types.Client{}, nil
	}

	current, ok := pool.Resources[resource.ID]
	if !ok {
		return types.Client{}, nil
	}

	now := time.Now()
	client.Heartbeat = now

	current.Clients[client.ID] = client
	current.LastClientHeartbeat = now

	pool.Resources[resource.ID] = current

	return client, nil
}

func (s *InMemoryResourcePool) DeleteClients(resource types.Resource, clients []types.Client) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	pool, ok := s.pools[resource.PoolID]
	if !ok {
		return nil
	}

	current, ok := pool.Resources[resource.ID]
	if !ok {
		return nil
	}

	for _, c := range clients {
		delete(current.Clients, c.ID)
	}

	return nil
}

func (s *InMemoryResourcePool) AppendEvent(event types.ResourceEvent) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	log.Printf("Event: pool=%s id=%s meta=%v \n", event.ResourcePoolID, event.ResourceID, event.Meta)
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

func (s *InMemoryResourcePool) GetEventsByPool(id string, limit int, before time.Time) (result []types.ResourceEvent, err error) {
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

func (s *InMemoryResourcePool) GetEventsByResource(poolID, id string, limit int, before time.Time) (result []types.ResourceEvent, err error) {
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

func copyPool(pool types.ResourcePool) (out types.ResourcePool, err error) {
	bs, err := json.Marshal(pool)
	if err != nil {
		return out, err
	}
	err = json.Unmarshal(bs, &out)
	return out, err
}

func copyResource(pool types.Resource) (out types.Resource, err error) {
	bs, err := json.Marshal(pool)
	if err != nil {
		return out, err
	}
	err = json.Unmarshal(bs, &out)
	return out, err
}
