package resource

import (
	"strconv"
	"testing"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/rueian/godemand/types"
)

var _ = Describe("InMemoryResourcePool", func() {
	var DefaultPool = "default"
	var store *InMemoryResourcePool

	var err error
	var pool types.ResourcePool

	BeforeEach(func() {
		store = NewInMemoryResourcePool(WithEventLimitPerPool(5))
	})

	Describe("GetResources", func() {
		It("get pool", func() {
			pool, err := store.GetResources(DefaultPool)
			Expect(err).NotTo(HaveOccurred())
			Expect(pool.ID).To(Equal(DefaultPool))
		})
	})

	Describe("SaveResource", func() {
		It("override the resource", func() {
			input := types.Resource{ID: "a", PoolID: DefaultPool, Clients: map[string]types.Client{}}
			_, err := store.SaveResource(input)
			Expect(err).NotTo(HaveOccurred())
			pool, err = store.GetResources(DefaultPool)
			Expect(err).NotTo(HaveOccurred())
			Expect(pool.Resources).To(HaveKey("a"))
			Expect(pool.Resources["a"].ID).To(Equal("a"))
			Expect(pool.Resources["a"].PoolID).To(Equal(DefaultPool))
			Expect(pool.Resources["a"].StateChange).NotTo(BeZero())
		})
	})

	Describe("DeleteResource", func() {
		It("get no error if no such resource", func() {
			err := store.DeleteResource(types.Resource{ID: "a", PoolID: DefaultPool})
			Expect(err).NotTo(HaveOccurred())
		})

		It("delete the resource if resource exists", func() {
			res := types.Resource{ID: "a", PoolID: DefaultPool}
			_, err := store.SaveResource(res)
			Expect(err).NotTo(HaveOccurred())

			err = store.DeleteResource(res)
			Expect(err).NotTo(HaveOccurred())

			pool, err = store.GetResources(DefaultPool)
			Expect(err).NotTo(HaveOccurred())
			Expect(pool.Resources).NotTo(HaveKey(res.ID))
		})
	})

	Describe("SaveClient", func() {
		It("set client heartbeat", func() {
			res := types.Resource{ID: "a", PoolID: DefaultPool}
			_, err := store.SaveResource(res)
			Expect(err).NotTo(HaveOccurred())

			client := types.Client{ID: "a", Meta: map[string]interface{}{"a": "a"}}
			_, err = store.SaveClient(res, client)
			Expect(err).NotTo(HaveOccurred())

			pool, err = store.GetResources(DefaultPool)
			Expect(err).NotTo(HaveOccurred())
			Expect(pool.Resources[res.ID].Clients[client.ID].Meta).To(Equal(client.Meta))
			Expect(pool.Resources[res.ID].Clients[client.ID].Heartbeat).NotTo(BeZero())
			Expect(pool.Resources[res.ID].LastClientHeartbeat).NotTo(BeZero())
		})
	})

	Describe("DeleteClients", func() {
		It("delete clients", func() {
			res := types.Resource{ID: "a", PoolID: DefaultPool}
			_, err := store.SaveResource(res)
			Expect(err).NotTo(HaveOccurred())

			client := types.Client{ID: "a", Meta: map[string]interface{}{"a": "a"}}
			_, err = store.SaveClient(res, client)
			Expect(err).NotTo(HaveOccurred())

			err = store.DeleteClients(res, []types.Client{client})

			pool, err = store.GetResources(DefaultPool)
			Expect(err).NotTo(HaveOccurred())
			Expect(pool.Resources[res.ID].Clients).NotTo(HaveKey(client.ID))
		})
	})

	Describe("AppendEvent", func() {
		It("append events and respect to eventLimitPerPool", func() {
			for i := 0; i < store.eventLimitPerPool*2; i++ {
				err := store.AppendEvent(types.ResourceEvent{
					ResourceID:     "a",
					ResourcePoolID: DefaultPool,
					Timestamp:      time.Now(),
					Meta: map[string]interface{}{
						"type":  "state",
						"prev":  types.ResourceServing,
						"next":  types.ResourceTerminating,
						"since": time.Now(),
						"taken": 0,
					},
				})
				Expect(err).NotTo(HaveOccurred())
			}
			Expect(store.events[DefaultPool]).To(HaveLen(store.eventLimitPerPool))
		})
	})

	Describe("GetEvents", func() {
		BeforeEach(func() {
			for i := 0; i < store.eventLimitPerPool; i++ {
				err := store.AppendEvent(types.ResourceEvent{
					ResourceID:     strconv.Itoa(i % 2),
					ResourcePoolID: DefaultPool,
					Timestamp:      time.Now().Add(-1 * time.Duration(i) * time.Hour),
				})
				Expect(err).NotTo(HaveOccurred())
			}
			Expect(err).NotTo(HaveOccurred())
			err = store.AppendEvent(types.ResourceEvent{
				ResourceID:     "b",
				ResourcePoolID: "other",
				Timestamp:      time.Now(),
			})
			Expect(err).NotTo(HaveOccurred())
		})

		Describe("GetEventsByPool", func() {
			It("get events desc by timestamp and filter by time", func() {
				evs, err := store.GetEventsByPool(DefaultPool, 5, time.Now().Add(-1*time.Hour))
				Expect(err).NotTo(HaveOccurred())
				Expect(evs).To(HaveLen(4))
				for i, ev := range evs {
					Expect(ev.ResourcePoolID).To(Equal(DefaultPool))
					Expect(ev.Timestamp.Before(time.Now().Add(-1 * time.Hour))).To(BeTrue())
					if i > 0 {
						Expect(ev.Timestamp.After(evs[i-1].Timestamp)).To(BeTrue())
					}
				}
			})
		})

		Describe("GetEventsByResource", func() {
			It("get events desc by timestamp and filter by time", func() {
				evs, err := store.GetEventsByResource(DefaultPool, "1", 5, time.Now().Add(-1*time.Hour))
				Expect(err).NotTo(HaveOccurred())
				Expect(evs).To(HaveLen(2))
				for i, ev := range evs {
					Expect(ev.ResourcePoolID).To(Equal(DefaultPool))
					Expect(ev.Timestamp.Before(time.Now().Add(-1 * time.Hour))).To(BeTrue())
					if i > 0 {
						Expect(ev.Timestamp.After(evs[i-1].Timestamp)).To(BeTrue())
					}
				}
			})
		})
	})
})

func TestInMemoryResourceStore(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "InMemoryResourcePool Suite")
}
