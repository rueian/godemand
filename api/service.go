package api

import (
	"time"

	"github.com/rueian/godemand/config"
	"github.com/rueian/godemand/types"
	"golang.org/x/xerrors"
)

var ResourceNotFoundErr = xerrors.New("resource not found in pool")

type Service struct {
	Pool      types.ResourceDAO
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

	pool, err := s.Pool.GetResources(poolID)
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
		event.Meta = types.Meta{
			"type":   "created",
			"client": client,
		}
		res.CreatedAt = time.Now()
		res.StateChange = time.Now()
	} else {
		event.Meta = types.Meta{
			"type":   "requested",
			"client": client,
		}
	}
	res.PoolID = pool.ID
	if pool.Resources[res.ID].State != res.State && pool.Resources[res.ID].StateChange == res.StateChange {
		res.StateChange = time.Now()
	}
	if res, err = s.Pool.SaveResource(res); err != nil {
		return types.Resource{}, err
	}
	if err := s.Pool.AppendEvent(event); err != nil {
		return types.Resource{}, err
	}

	return res, err
}

func (s *Service) GetResource(poolID, id string) (res types.Resource, err error) {
	pool, err := s.Pool.GetResources(poolID)
	if err != nil {
		return types.Resource{}, err
	}

	if res, ok := pool.Resources[id]; ok {
		return res, nil
	}

	return types.Resource{}, xerrors.Errorf("resource %q not found in pool %q: %w", id, poolID, ResourceNotFoundErr)
}

func (s *Service) Heartbeat(poolID, id string, client types.Client) (err error) {
	pool, err := s.Pool.GetResources(poolID)
	if err != nil {
		return err
	}

	res, ok := pool.Resources[id]
	if !ok {
		return xerrors.Errorf("resource %q not found in pool %q: %w", id, poolID, ResourceNotFoundErr)
	}

	now := time.Now()
	if c, ok := res.Clients[client.ID]; ok {
		client.CreatedAt = c.CreatedAt
	} else {
		client.CreatedAt = now
	}

	client.Heartbeat = now
	if _, err = s.Pool.SaveClient(types.Resource{ID: id, PoolID: poolID}, client); err != nil {
		return err
	}

	return nil
}
