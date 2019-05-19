package redis

import (
	"time"

	"github.com/go-redis/redis"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/rueian/godemand/plugin"
	"golang.org/x/xerrors"
)

var _ = Describe("Locker", func() {
	var locker *Locker
	var key, id string
	var err error
	var client redis.UniversalClient

	BeforeEach(func() {
		client = redis.NewClient(&redis.Options{Addr: "127.0.0.1:6379"})
		_, err = client.FlushAll().Result()
		Expect(err).NotTo(HaveOccurred())

		key = "Default"
		locker = NewLocker(client, WithExpiration(1*time.Minute))
		id, err = locker.AcquireLock(key)
	})

	Describe("AcquireLock", func() {
		It("get id if acquire success", func() {
			Expect(err).NotTo(HaveOccurred())
			Expect(id).NotTo(BeEmpty())
		})

		It("get AcquireLaterErr if others acquired", func() {
			id, err = locker.AcquireLock(key)
			Expect(xerrors.Is(err, plugin.AcquireLaterErr)).To(BeTrue())
			Expect(id).To(BeEmpty())
		})
	})

	Describe("ReleaseLock", func() {
		JustBeforeEach(func() {
			err = locker.ReleaseLock(key, id)
		})
		Context("locked key and id", func() {
			It("return no error", func() {
				Expect(err).NotTo(HaveOccurred())
			})
			It("can lock again", func() {
				id, err = locker.AcquireLock(key)
				Expect(err).NotTo(HaveOccurred())
				Expect(id).NotTo(BeEmpty())
			})
		})
		Context("key not locked", func() {
			BeforeEach(func() {
				key = "other"
			})
			It("get LockNotFoundErr", func() {
				Expect(xerrors.Is(err, plugin.LockNotFoundErr)).To(BeTrue())
			})
		})
		Context("id mismatch", func() {
			BeforeEach(func() {
				id = "other"
			})
			It("get LockNotFoundErr", func() {
				Expect(xerrors.Is(err, plugin.LockNotFoundErr)).To(BeTrue())
			})
		})
	})
})
