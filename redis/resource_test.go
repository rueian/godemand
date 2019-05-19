package redis

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/go-redis/redis"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/rueian/godemand/types"
)

var _ = Describe("ResourcePool", func() {
	var client redis.UniversalClient
	var err error
	var dao *ResourcePool
	var eventLimit int64

	BeforeEach(func() {
		eventLimit = 10
		client = redis.NewClient(&redis.Options{Addr: "127.0.0.1:6379"})
		_, err = client.FlushAll().Result()
		Expect(err).NotTo(HaveOccurred())

		dao = NewResourcePool(client, WithEventLimitPerPool(eventLimit))
	})

	AfterEach(func() {
		client.Close()
	})

	Describe("Resources", func() {
		var source, pool types.ResourcePool
		var deleteClient [][2]string
		var deleteResource []string
		var id string

		BeforeEach(func() {
			id = "pool"
			source = types.ResourcePool{
				ID:        id,
				Resources: map[string]types.Resource{},
			}
		})

		JustBeforeEach(func() {
			for _, v := range source.Resources {
				_, err = dao.SaveResource(v)
				Expect(err).NotTo(HaveOccurred())

				for _, c := range v.Clients {
					_, err = dao.SaveClient(v, c)
					Expect(err).NotTo(HaveOccurred())
				}
			}

			for _, d := range deleteClient {
				err = dao.DeleteClients(types.Resource{
					ID:     d[0],
					PoolID: id,
				}, []types.Client{
					{
						ID: d[1],
					},
				})
				Expect(err).NotTo(HaveOccurred())
				delete(source.Resources[d[0]].Clients, d[1])
			}

			for _, d := range deleteResource {
				err = dao.DeleteResource(types.Resource{
					ID:     d,
					PoolID: id,
				})
				Expect(err).NotTo(HaveOccurred())
				delete(source.Resources, d)
			}

			pool, err = dao.GetResources(id)
			Expect(err).NotTo(HaveOccurred())
		})

		Context("empty", func() {
			It("get empty", func() {
				Expect(pool.ID).To(Equal(id))
				Expect(pool.Resources).To(HaveLen(0))
			})
		})

		Context("with source", func() {
			BeforeEach(func() {
				t := time.Now()
				source.Resources = map[string]types.Resource{
					"1": {
						ID:     "1",
						PoolID: id,
						Meta: map[string]interface{}{
							"any": "thing",
						},
						Clients: map[string]types.Client{
							"1": {
								ID: "1",
								Meta: map[string]interface{}{
									"addr": "127.0.0.1",
								},
								Heartbeat: t,
							},
							"2": {
								ID: "2",
								Meta: map[string]interface{}{
									"addr": "127.0.0.1",
								},
								Heartbeat: t.Add(10 * time.Minute),
							},
						},
						LastClientHeartbeat: t,
					},
					"2": {
						ID:      "2",
						PoolID:  id,
						Clients: map[string]types.Client{},
					},
				}
				deleteClient = [][2]string{
					{"1", "2"},
				}
				deleteResource = []string{
					"2",
				}
			})
			It("get as source", func() {
				pj, _ := json.Marshal(pool)
				ps, _ := json.Marshal(source)
				Expect(pj).To(MatchJSON(ps))
			})
		})
	})

	Describe("Events", func() {
		var source, byPool, byResource []types.ResourceEvent
		var poolID string
		var resourceID string
		var limit int
		var before time.Time

		BeforeEach(func() {
			poolID = "pool"
			resourceID = "resource"
			source = nil
			limit = int(eventLimit)
			before = time.Now()
		})

		JustBeforeEach(func() {
			for _, e := range source {
				err = dao.AppendEvent(e)
				Expect(err).NotTo(HaveOccurred())
			}

			byPool, err = dao.GetEventsByPool(poolID, limit, before)
			Expect(err).NotTo(HaveOccurred())
			byResource, err = dao.GetEventsByResource(poolID, resourceID, limit, before)
			Expect(err).NotTo(HaveOccurred())
		})

		Context("empty", func() {
			It("got empty", func() {
				Expect(byPool).To(HaveLen(0))
				Expect(byResource).To(HaveLen(0))
			})
		})

		Context("with source", func() {
			BeforeEach(func() {
				t := time.Now()
				resourceIDs := []string{resourceID, "other1", "other2"}
				for i := 0; i < int(eventLimit)*2; i++ {
					source = append(source, types.ResourceEvent{
						ResourcePoolID: poolID,
						ResourceID:     resourceIDs[i%len(resourceIDs)],
						Meta: map[string]interface{}{
							"i": i,
						},
						Timestamp: t.Add(-10 * time.Minute * time.Duration(i+1)),
					})
				}
			})
			Context("default", func() {
				It("got events", func() {
					Expect(byPool).To(HaveLen(int(eventLimit)))
					Expect(byResource).NotTo(HaveLen(0))
					for _, e := range byResource {
						Expect(e.ResourceID).To(Equal(resourceID))
						Expect(int(e.Meta["i"].(float64)) % 3).To(Equal(0))
					}
				})
			})
			Context("smaller", func() {
				BeforeEach(func() {
					limit = int(eventLimit - 4)
					before = time.Now().Add(-11 * time.Minute)
				})
				It("got events", func() {
					Expect(byPool).To(HaveLen(limit))
					Expect(byPool[0].Timestamp.Before(before)).To(BeTrue())
				})
			})
		})
	})
})

func TestRedisResourceStore(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "RedisResourcePool Suite")
}
