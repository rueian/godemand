package syncer

import (
	"context"
	"time"

	"github.com/rueian/godemand/config"
	"github.com/rueian/godemand/types"
)

type ResourceSyncer struct {
	Pool      types.ResourcePoolDAO
	Locker    types.Locker
	Launchpad types.Launchpad
	Config    *config.Config

	queue chan types.Resource
}

func (s *ResourceSyncer) Run(ctx context.Context, workers int) error {
	s.queue = make(chan types.Resource, workers)

	for i := 0; i < workers; i++ {
		go func() {
			for res := range s.queue {
				config, err := s.Config.GetPool(res.PoolID)
				if err != nil {
					// TODO logging
					continue
				}

				controller, err := s.Launchpad.GetController(config.Plugin)
				if err != nil {
					// TODO logging
					continue
				}

				lockID, err := s.Locker.AcquireLock(res.ID)
				if err != nil {
					// TODO logging
					continue
				}

				var ret types.Resource
				for {
					ret, err = controller.SyncResource(res, config.Params)
					if err != nil {
						break
					}
					if ret.State != res.State && ret.StateChange == res.StateChange {
						ret.StateChange = time.Now()
					}
					if _, err = s.Pool.SaveResource(ret); err != nil {
						break
					}
					if ret.State == res.State {
						break
					}
					if err = s.Pool.AppendEvent(types.ResourceEvent{
						ResourcePoolID: ret.PoolID,
						ResourceID:     ret.ID,
						Timestamp:      time.Now(),
						Meta: map[string]interface{}{
							"type":  "state",
							"prev":  res.State,
							"next":  ret.State,
							"since": res.StateChange,
							"taken": int(time.Since(res.StateChange).Seconds()),
						},
					}); err != nil {
						break
					}
					res = ret
				}
				s.Locker.ReleaseLock(res.ID, lockID)
				if err != nil {
					// TODO logging
					continue
				}
			}
		}()
	}

	for {
		select {
		case <-ctx.Done():
			close(s.queue)
			return ctx.Err()
		default:
		}

		begin := time.Now()

		pools := s.Config.Pools

		for id := range pools {
			pool, err := s.Pool.GetResources(id)
			if err != nil {
				// TODO logging
				continue
			}

			for _, res := range pool.Resources {
				s.queue <- res
			}
		}

		if time.Since(begin) < time.Second {
			time.Sleep(time.Second)
		}
	}
}
