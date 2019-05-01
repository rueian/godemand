package api

import (
	"time"

	"github.com/rueian/godemand/config"
	"github.com/rueian/godemand/dao"
	"github.com/rueian/godemand/types"
	"golang.org/x/xerrors"
)

type Service struct {
	Resource  dao.ResourceDAO
	Locker    types.Locker
	Launchpad types.Launchpad
	Config    *config.Config
}

func (s *Service) RequestResource(poolID string, client types.Client) (res types.Resource, err error) {
	lockID, err := s.Locker.AcquireLock(poolID)
	if err != nil {
		return types.Resource{}, err
	}
	defer s.Locker.ReleaseLock(poolID, lockID)

	poolConfig, err := s.Config.GetPool(poolID)
	if err != nil {
		return types.Resource{}, err
	}

	controller, err := s.Launchpad.GetController(poolConfig.Plugin)
	if err != nil {
		return types.Resource{}, err
	}

	pool, err := s.Resource.GetResourcePool(poolID)
	if err != nil {
		return types.Resource{}, err
	}

	res, err = controller.FindResource(pool, poolConfig.Params)
	if err != nil {
		return types.Resource{}, err
	}

	event := types.ResourceEvent{
		ResourceID:     res.ID,
		ResourcePoolID: res.PoolID,
		Timestamp:      time.Now(),
	}

	if _, ok := pool.Resources[res.ID]; !ok {
		res.PoolID = pool.ID
		res, err = s.Resource.SaveResource(res)
		event.Meta = types.Meta{
			"type":   "created",
			"client": client,
		}
	} else {
		event.Meta = types.Meta{
			"type":   "requested",
			"client": client,
		}
	}
	if err := s.Resource.AppendResourceEvent(event); err != nil {
		return types.Resource{}, err
	}

	return res, err
}

func (s *Service) GetResource(poolID, id string) (res types.Resource, err error) {
	pool, err := s.Resource.GetResourcePool(poolID)
	if err != nil {
		return types.Resource{}, err
	}

	if res, ok := pool.Resources[id]; ok {
		return res, nil
	}

	return types.Resource{}, xerrors.Errorf("resource %q not found in pool", id)
}

func (s *Service) Heartbeat(poolID, id string, client types.Client) (err error) {
	lockID, err := s.Locker.AcquireLock(poolID)
	if err != nil {
		return err
	}
	defer s.Locker.ReleaseLock(poolID, lockID)

	res, err := s.GetResource(poolID, id)
	if err != nil {
		return err
	}

	index := -1
	for i, c := range res.Clients {
		if c.ID == client.ID {
			index = i
		}
	}

	now := time.Now()
	client.Heartbeat = now
	if index != -1 {
		res.Clients[index] = client
	} else {
		res.Clients = append(res.Clients, client)
	}

	res.LastClientHeartbeat = now
	_, err = s.Resource.SaveResource(res)
	if err != nil {
		return err
	}

	return nil
}
