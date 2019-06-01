package redis

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"strconv"
	"time"

	"github.com/go-redis/redis"
	"github.com/rueian/godemand/types"
)

type ResourcePoolOptionFunc func(*ResourcePool)

func WithEventLimitPerPool(limit int64) ResourcePoolOptionFunc {
	return func(store *ResourcePool) {
		store.eventLimitPerPool = limit
	}
}

func NewResourcePool(client redis.UniversalClient, options ...ResourcePoolOptionFunc) *ResourcePool {
	s := &ResourcePool{
		client:            client,
		eventLimitPerPool: 1000,
	}

	for _, of := range options {
		of(s)
	}

	return s
}

type ResourcePool struct {
	client            redis.UniversalClient
	eventLimitPerPool int64
}

func (p *ResourcePool) GetResources(id string) (types.ResourcePool, error) {
	res, err := p.client.HGetAll(id).Result()
	if err != nil {
		return types.ResourcePool{}, err
	}

	pool := types.ResourcePool{
		ID:        id,
		Resources: make(map[string]types.Resource),
	}

	for k, v := range res {
		var resource types.Resource
		if err = json.Unmarshal([]byte(v), &resource); err != nil {
			return types.ResourcePool{}, err
		}

		clients, lastHeartbeat, err := p.getClients(resource)
		if err != nil {
			return types.ResourcePool{}, err
		}

		resource.Clients = clients
		resource.LastClientHeartbeat = lastHeartbeat

		pool.Resources[k] = resource
	}

	return pool, nil
}

func (p *ResourcePool) GetResource(pool, id string) (types.Resource, error) {
	res, err := p.client.HGet(pool, id).Result()
	if err != nil {
		if err == redis.Nil {
			err = types.ResourceNotFoundErr
		}
		return types.Resource{}, err
	}

	var resource types.Resource
	if err = json.Unmarshal([]byte(res), &resource); err != nil {
		return types.Resource{}, err
	}

	clients, lastHeartbeat, err := p.getClients(resource)
	if err != nil {
		return types.Resource{}, err
	}

	resource.Clients = clients
	resource.LastClientHeartbeat = lastHeartbeat

	return resource, nil
}

func (p *ResourcePool) getClients(resource types.Resource) (clients map[string]types.Client, lastHeartbeat time.Time, err error) {
	cs, err := p.client.HGetAll(clientHashKey(resource)).Result()
	if err != nil {
		return nil, time.Time{}, err
	}

	clients = make(map[string]types.Client)

	for ck, cv := range cs {
		var client types.Client
		if err = json.Unmarshal([]byte(cv), &client); err != nil {
			return nil, time.Time{}, err
		}
		clients[ck] = client

		if lastHeartbeat.Before(client.Heartbeat) {
			lastHeartbeat = client.Heartbeat
		}
	}
	return
}

func (p *ResourcePool) SaveResource(resource types.Resource) (cp types.Resource, err error) {
	versionKey := resourceVersionKey(resource)
	for {
		var current types.Resource

		err = p.client.Watch(func(tx *redis.Tx) error {
			res, err := tx.HGet(resource.PoolID, resource.ID).Result()

			if err == redis.Nil {
				current = resource
				current.LastClientHeartbeat = time.Time{}
				current.Clients = nil
			} else if err != nil {
				return err
			} else {
				if err = json.Unmarshal([]byte(res), &current); err != nil {
					return err
				}
			}

			if !current.LastSynced.After(resource.LastSynced) {
				current.LastSynced = resource.LastSynced
				current.Meta = resource.Meta
			}
			if !current.StateChange.After(resource.StateChange) {
				current.StateChange = resource.StateChange
				current.State = resource.State
			}

			v, err := json.Marshal(current)
			if err != nil {
				return err
			}

			_, err = tx.TxPipelined(func(pipe redis.Pipeliner) error {
				pipe.Incr(versionKey)
				pipe.HSet(resource.PoolID, resource.ID, string(v))
				return nil
			})

			return err
		}, versionKey)

		if err == nil {
			return current, nil
		}

		if err != redis.TxFailedErr {
			return types.Resource{}, err
		}
	}
}

func (p *ResourcePool) DeleteResource(resource types.Resource) error {
	_, err := p.client.TxPipelined(func(pipe redis.Pipeliner) error {
		pipe.HDel(resource.PoolID, resource.ID)
		pipe.Del(clientHashKey(resource))
		pipe.Del(resourceVersionKey(resource))
		return nil
	})
	return err
}

func (p *ResourcePool) SaveClient(resource types.Resource, client types.Client) (types.Client, error) {
	v, err := json.Marshal(client)
	if err != nil {
		return types.Client{}, err
	}

	if _, err = p.client.HSet(clientHashKey(resource), client.ID, string(v)).Result(); err != nil {
		return types.Client{}, err
	}

	return client, nil
}

func (p *ResourcePool) DeleteClients(resource types.Resource, clients []types.Client) error {
	if len(clients) == 0 {
		return nil
	}

	var ids []string
	for _, c := range clients {
		ids = append(ids, c.ID)
	}

	_, err := p.client.HDel(clientHashKey(resource), ids...).Result()
	return err
}

func (p *ResourcePool) AppendEvent(event types.ResourceEvent) error {
	if event.Meta == nil {
		event.Meta = types.Meta{}
	}
	event.Meta["rand"] = rand.Int63() // avoid event collision in sorted set

	v, err := json.Marshal(event)
	if err != nil {
		return err
	}

	l, err := p.client.ZCard(eventListKey(event.ResourcePoolID)).Result()
	if err != nil {
		return err
	}

	if l >= p.eventLimitPerPool {
		if _, err := p.client.ZPopMin(eventListKey(event.ResourcePoolID), l-p.eventLimitPerPool+1).Result(); err != nil {
			return err
		}
	}

	if _, err = p.client.ZAdd(eventListKey(event.ResourcePoolID), redis.Z{
		Score:  float64(event.Timestamp.UnixNano()),
		Member: string(v),
	}).Result(); err != nil {
		return err
	}

	return nil
}

func (p *ResourcePool) GetEventsByPool(id string, limit int, before time.Time) (events []types.ResourceEvent, err error) {
	res, err := p.client.ZRevRangeByScore(eventListKey(id), redis.ZRangeBy{
		Max:    "(" + strconv.FormatInt(before.UnixNano(), 10),
		Min:    "0",
		Offset: 0,
		Count:  int64(limit),
	}).Result()
	if err != nil {
		return nil, err
	}

	for _, e := range res {
		event := types.ResourceEvent{}
		if err := json.Unmarshal([]byte(e), &event); err != nil {
			return nil, err
		}
		events = append(events, event)
	}

	return
}

func (p *ResourcePool) GetEventsByResource(poolID, id string, limit int, before time.Time) (events []types.ResourceEvent, err error) {
	for len(events) != limit {
		evs, err := p.GetEventsByPool(poolID, limit, before)
		if err != nil {
			return nil, err
		}
		if len(evs) == 0 {
			return events, nil
		}
		for _, e := range evs {
			if e.ResourceID == id {
				events = append(events, e)
			}
			before = e.Timestamp
		}
	}
	return
}

func clientHashKey(resource types.Resource) string {
	return fmt.Sprintf("%s:%s:clients", resource.PoolID, resource.ID)
}

func resourceVersionKey(resource types.Resource) string {
	return fmt.Sprintf("%s:%s:version", resource.PoolID, resource.ID)
}

func eventListKey(poolID string) string {
	return fmt.Sprintf("%s:events", poolID)
}
