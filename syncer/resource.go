package syncer

import (
	"context"
	"time"

	"github.com/rueian/godemand/config"
	"github.com/rueian/godemand/metrics"
	"github.com/rueian/godemand/types"
)

type ResourceSyncer struct {
	Pool      types.ResourceDAO
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
					ret, err = controller.SyncResource(res, types.Merge(config.Params, res.Config))
					if err != nil {
						break
					}
					if ret.State != res.State && ret.StateChange == res.StateChange {
						ret.StateChange = time.Now()
					}
					if _, err = s.Pool.SaveResource(ret); err != nil {
						break
					}
					if ret.State == types.ResourceDeleted {
						if err = s.Pool.DeleteResource(ret); err == nil {
							err = s.Pool.AppendEvent(types.ResourceEvent{
								ResourcePoolID: ret.PoolID,
								ResourceID:     ret.ID,
								Timestamp:      time.Now(),
								Meta: map[string]interface{}{
									"type": "deleted",
								},
							})
						}
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

			// record metrics
			sc := stateCounter()
			clients := 0
			for _, res := range pool.Resources {
				sc[res.State.String()]++
				metrics.RecordResourceLife(res.PoolID, res.State.String(), res.ID, time.Since(res.StateChange))
				for _, c := range res.Clients {
					if time.Since(c.Heartbeat) < 1*time.Minute {
						clients++
						metrics.RecordClientLife(res.PoolID, c.ID, c.Heartbeat.Sub(c.CreatedAt))
						if rt, ok := c.Meta["requestAt"]; ok {
							rt := toTime(rt)
							ut := time.Now()
							if st, ok := c.Meta["servedAt"]; ok {
								ut = toTime(st)
							}
							metrics.RecordClientWait(res.PoolID, c.ID, ut.Sub(rt))
						}
					}
				}
			}
			for state, count := range sc {
				metrics.RecordResourceCount(id, state, count)
			}
			metrics.RecordClientCount(id, int64(clients))
		}

		if time.Since(begin) < time.Second {
			time.Sleep(time.Second)
		}
	}
}

func stateCounter() map[string]int64 {
	counter := make(map[string]int64)
	for _, s := range types.ResourceStates {
		counter[s.String()] = 0
	}
	return counter
}

func toTime(t interface{}) time.Time {
	switch v := t.(type) {
	case time.Time:
		return v
	case string:
		ts, _ := time.Parse(time.RFC3339, v)
		return ts
	}

	return time.Time{}
}
