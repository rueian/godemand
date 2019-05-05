package syncer

import (
	"context"
	"errors"
	"testing"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/rueian/godemand/config"
	"github.com/rueian/godemand/plugin"
	"github.com/rueian/godemand/resource"
	"github.com/rueian/godemand/types"
	"github.com/rueian/godemand/types/mock"
)

var _ = Describe("Syncer", func() {
	var syncer *ResourceSyncer
	var launchpad *mock.MockLaunchpad
	var controller *mock.MockController
	var pool types.ResourcePoolDAO
	var locker *mock.MockLocker
	var ctrl *gomock.Controller
	var cfg *config.Config
	var res types.Resource

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
		res = types.Resource{
			ID:      "a",
			PoolID:  "pool1",
			Meta:    map[string]interface{}{},
			Clients: map[string]types.Client{},
		}
		pool.SaveResource(res)
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	JustBeforeEach(func() {
		syncer = &ResourceSyncer{
			Pool:      pool,
			Locker:    locker,
			Config:    cfg,
			Launchpad: launchpad,
		}
	})

	var ctx context.Context
	var cancel context.CancelFunc
	var worker int
	var err error

	Describe("Run", func() {
		BeforeEach(func() {
			worker = 1
		})

		JustBeforeEach(func() {
			err = syncer.Run(ctx, worker)
		})

		Context("call SyncResource", func() {
			BeforeEach(func() {
				ctx, cancel = context.WithCancel(context.Background())
				launchpad.EXPECT().GetController("plugin1").Return(controller, nil)
				locker.EXPECT().AcquireLock(res.ID).Return("lockID", nil)
				locker.EXPECT().ReleaseLock(res.ID, "lockID").Return(nil)
				controller.EXPECT().SyncResource(res, cfg.Pools["pool1"].Params).Return(res, nil).Do(func(interface{}, interface{}) {
					cancel()
				})
			})
			It("call SyncResource", func() {
				Expect(err).To(Equal(context.Canceled))
			})
		})
		Context("call SyncResource multiple times if state change", func() {
			BeforeEach(func() {
				ctx, cancel = context.WithCancel(context.Background())
				launchpad.EXPECT().GetController("plugin1").Return(controller, nil)
				locker.EXPECT().AcquireLock(res.ID).Return("lockID", nil)
				locker.EXPECT().ReleaseLock(res.ID, "lockID").Return(nil)
				alter := res
				alter.State = types.ResourceRunning
				controller.EXPECT().SyncResource(gomock.Any(), cfg.Pools["pool1"].Params).DoAndReturn(func(res types.Resource, params map[string]interface{}) (types.Resource, error) {
					if res.State != alter.State || res.StateChange.IsZero() {
						return types.Resource{}, errors.New("input not match")
					}
					cancel()
					return res, nil
				}).After(
					controller.EXPECT().SyncResource(res, cfg.Pools["pool1"].Params).Return(alter, nil),
				)
			})
			It("call SyncResource", func() {
				Expect(err).To(Equal(context.Canceled))
			})
		})
		Context("plugin error", func() {
			BeforeEach(func() {
				ctx, cancel = context.WithCancel(context.Background())
				launchpad.EXPECT().GetController("plugin1").Return(controller, nil)
				locker.EXPECT().AcquireLock(res.ID).Return("lockID", nil)
				locker.EXPECT().ReleaseLock(res.ID, "lockID").Return(nil)
				controller.EXPECT().SyncResource(res, cfg.Pools["pool1"].Params).Return(res, errors.New("random")).Do(func(interface{}, interface{}) {
					cancel()
				})
			})
			It("call SyncResource", func() {
				Expect(err).To(Equal(context.Canceled))
			})
		})
		Context("no controller", func() {
			BeforeEach(func() {
				ctx, cancel = context.WithCancel(context.Background())
				launchpad.EXPECT().GetController("plugin1").Return(nil, plugin.ControllerNotFoundErr).Do(func(x string) {
					cancel()
				})
			})
			It("run", func() {
				Expect(err).To(Equal(context.Canceled))
			})
		})
		Context("lock fail", func() {
			BeforeEach(func() {
				ctx, cancel = context.WithCancel(context.Background())
				launchpad.EXPECT().GetController("plugin1").Return(controller, nil)
				locker.EXPECT().AcquireLock(res.ID).Return("lockID", nil).Return("", plugin.AcquireLaterErr).Do(func(x string) {
					cancel()
				})
			})
			It("run", func() {
				Expect(err).To(Equal(context.Canceled))
			})
		})
	})
})

func TestSyncer(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Syncer Suite")
}
