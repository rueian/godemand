package plugin

import (
	"errors"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("InMemoryLocker", func() {
	var locker *InMemoryLocker
	var key, id string
	var err error

	BeforeEach(func() {
		key = "Default"
		locker = NewInMemoryLocker()
		id, err = locker.AcquireLock(key)
	})

	Describe("AcquireLock", func() {
		It("get id if acquire success", func() {
			Expect(err).NotTo(HaveOccurred())
			Expect(id).NotTo(BeEmpty())
		})

		It("get AcquireLaterErr if others acquired", func() {
			id, err = locker.AcquireLock(key)
			Expect(errors.Is(err, AcquireLaterErr)).To(BeTrue())
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
				Expect(errors.Is(err, LockNotFoundErr)).To(BeTrue())
			})
		})
		Context("id mismatch", func() {
			BeforeEach(func() {
				id = "other"
			})
			It("get LockNotFoundErr", func() {
				Expect(errors.Is(err, LockNotFoundErr)).To(BeTrue())
			})
		})
	})
})
