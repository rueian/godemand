package api

import (
	"errors"
	"time"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/rueian/godemand/config"
	"github.com/rueian/godemand/plugin"
	"github.com/rueian/godemand/resource"
	"github.com/rueian/godemand/types"
	"github.com/rueian/godemand/types/mock"
	"golang.org/x/xerrors"
)

var _ = Describe("Service", func() {
	var service *Service
	var launchpad *mock.MockLaunchpad
	var controller *mock.MockController
	var pool types.ResourcePoolDAO
	var locker *mock.MockLocker
	var ctrl *gomock.Controller
	var cfg *config.Config
	var client types.Client

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		pool = resource.NewInMemoryResourcePool()
		locker = mock.NewMockLocker(ctrl)
		launchpad = mock.NewMockLaunchpad(ctrl)
		controller = mock.NewMockController(ctrl)
		cfg = &config.Config{
			Plugins: map[string]config.PluginConfig{
				"plugin1": {},
			},
			Pools: map[string]config.PoolConfig{
				"pool1": {
					Plugin: "plugin1",
					Params: map[string]interface{}{
						"a": "a",
						"1": 1,
					},
				},
			},
		}
		client = types.Client{
			ID: "ginkgo",
			Meta: types.Meta{
				"ip": "0.0.0.0",
			},
		}
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	JustBeforeEach(func() {
		service = &Service{
			Pool:      pool,
			Locker:    locker,
			Config:    cfg,
			Launchpad: launchpad,
		}
	})

	var res types.Resource
	var err error
	var poolID string
	var lockID string
	var lockErr error

	BeforeEach(func() {
		poolID = "a"
		lockID = "lockID"
		lockErr = nil
	})

	Describe("RequestResource", func() {
		JustBeforeEach(func() {
			locker.EXPECT().AcquireLock(poolID).Return(lockID, lockErr)
			if lockErr == nil {
				locker.EXPECT().ReleaseLock(poolID, lockID).Return(nil)
			}
			res, err = service.RequestResource(poolID, client)
		})

		Context("lock fail", func() {
			BeforeEach(func() {
				poolID = "any"
				lockErr = plugin.AcquireLaterErr
			})
			It("get err", func() {
				Expect(xerrors.Is(err, plugin.AcquireLaterErr)).To(BeTrue())
			})
		})

		Context("pool not in config", func() {
			BeforeEach(func() {
				poolID = "any"
			})

			It("get err", func() {
				Expect(xerrors.Is(err, config.PoolConfigNotFoundErr)).To(BeTrue())
			})
		})

		Context("controller not in launchpad", func() {
			BeforeEach(func() {
				poolID = "pool1"
				launchpad.EXPECT().GetController("plugin1").Return(nil, plugin.ControllerNotFoundErr)
			})

			It("get err", func() {
				Expect(xerrors.Is(err, plugin.ControllerNotFoundErr)).To(BeTrue())
			})
		})

		Context("call controller's FindResource", func() {
			var pcfg config.PoolConfig
			var p types.ResourcePool

			BeforeEach(func() {
				poolID = "pool1"
				launchpad.EXPECT().GetController("plugin1").Return(controller, nil)
				pool.SaveResource(types.Resource{ID: "a", PoolID: poolID})
				p, _ = pool.GetResources(poolID)
				pcfg, _ = cfg.GetPool(poolID)
			})

			Context("err from controller", func() {
				BeforeEach(func() {
					controller.EXPECT().FindResource(p, pcfg.Params).Return(types.Resource{}, errors.New("any"))
				})
				It("got err", func() {
					Expect(err).To(Equal(errors.New("any")))
				})
			})

			Context("one of resources from controller", func() {
				BeforeEach(func() {
					controller.EXPECT().FindResource(p, pcfg.Params).Return(types.Resource{ID: "a", PoolID: poolID}, nil)
				})
				It("got no err", func() {
					Expect(err).NotTo(HaveOccurred())
				})
				It("got res", func() {
					Expect(res).To(Equal(types.Resource{ID: "a", PoolID: poolID, Clients: map[string]types.Client{}}))
				})
				It("append requested events", func() {
					events, err := pool.GetEventsByPool(poolID, 1, time.Now())
					Expect(err).NotTo(HaveOccurred())
					Expect(events).To(HaveLen(1))
					Expect(events[0].ResourceID).To(Equal("a"))
					Expect(events[0].ResourcePoolID).To(Equal(poolID))
					Expect(events[0].Meta).To(HaveKeyWithValue("type", "requested"))
					Expect(events[0].Meta).To(HaveKeyWithValue("client", client))
				})
			})

			Context("new resources from controller", func() {
				BeforeEach(func() {
					controller.EXPECT().FindResource(p, pcfg.Params).Return(types.Resource{ID: "b", PoolID: poolID}, nil)
				})
				It("got no err", func() {
					Expect(err).NotTo(HaveOccurred())
				})
				It("got res", func() {
					Expect(res).To(Equal(types.Resource{ID: "b", PoolID: poolID, Clients: map[string]types.Client{}}))
				})
				It("append requested events", func() {
					events, err := pool.GetEventsByPool(poolID, 1, time.Now())
					Expect(err).NotTo(HaveOccurred())
					Expect(events).To(HaveLen(1))
					Expect(events[0].ResourceID).To(Equal("b"))
					Expect(events[0].ResourcePoolID).To(Equal(poolID))
					Expect(events[0].Meta).To(HaveKeyWithValue("type", "created"))
					Expect(events[0].Meta).To(HaveKeyWithValue("client", client))
				})
				It("save the resource", func() {
					p, _ := pool.GetResources(poolID)
					Expect(p.Resources).To(HaveKeyWithValue("b", types.Resource{ID: "b", PoolID: poolID, Clients: map[string]types.Client{}}))
				})
			})
		})
	})

	Describe("GetResource", func() {
		JustBeforeEach(func() {
			res, err = service.GetResource(poolID, "a")
		})

		Context("resource exists", func() {
			BeforeEach(func() {
				pool.SaveResource(types.Resource{ID: "a", PoolID: poolID})
			})
			It("get res", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(res).To(Equal(types.Resource{ID: "a", PoolID: poolID, Clients: map[string]types.Client{}}))
			})
		})

		Context("not found", func() {
			It("got err", func() {
				Expect(err.Error()).To(ContainSubstring("not found"))
			})
		})
	})

	Describe("Heartbeat", func() {
		var resID string

		BeforeEach(func() {
			resID = "a"
			pool.SaveResource(types.Resource{ID: "a", PoolID: poolID})
		})

		JustBeforeEach(func() {
			err = service.Heartbeat(poolID, resID, client)
		})

		Context("first heartbeat", func() {
			It("append client to resource", func() {
				p, _ := pool.GetResources(poolID)
				res := p.Resources[resID]
				Expect(res.Clients).To(HaveLen(1))
				Expect(res.Clients[client.ID].Meta).To(Equal(client.Meta))
				Expect(res.Clients[client.ID].Heartbeat).NotTo(BeZero())
			})
		})

		Context("second heartbeat", func() {
			var firstTime time.Time
			BeforeEach(func() {
				res := types.Resource{ID: "a", PoolID: poolID}
				_, err := pool.SaveResource(res)
				Expect(err).NotTo(HaveOccurred())

				c, err := pool.SaveClient(res, client)
				firstTime = c.Heartbeat
			})
			It("append client to resource", func() {
				p, _ := pool.GetResources(poolID)
				res := p.Resources[resID]
				Expect(res.Clients).To(HaveLen(1))
				Expect(res.Clients[client.ID].Meta).To(Equal(client.Meta))
				Expect(res.Clients[client.ID].Heartbeat.After(firstTime)).To(BeTrue())
			})
		})
	})
})
