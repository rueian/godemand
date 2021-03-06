package client

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/rueian/godemand/api"
	"github.com/rueian/godemand/plugin"
	"github.com/rueian/godemand/types"
	"github.com/rueian/godemand/types/mock"
)

var _ = Describe("Client", func() {
	var client *HTTPClient
	var ctrl *gomock.Controller
	var service *mock.MockService
	var mux *http.ServeMux
	var server *httptest.Server
	var err error
	var res types.Resource
	var ctx context.Context
	var poolID = "pool1"
	var info = types.Client{ID: "fake"}

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		service = mock.NewMockService(ctrl)
		mux = api.NewHTTPMux(service)
		server = httptest.NewServer(mux)
		client = NewHTTPClient(server.URL, info, server.Client())
	})

	AfterEach(func() {
		ctrl.Finish()
		server.Close()
	})

	Describe("RequestResource", func() {
		JustBeforeEach(func() {
			res, err = client.RequestResource(ctx, poolID)
		})

		Context("deadline", func() {
			BeforeEach(func() {
				ctx, _ = context.WithDeadline(context.Background(), time.Now())
			})
			It("err", func() {
				Expect(errors.Is(err, context.DeadlineExceeded)).To(BeTrue())
			})
		})

		Context("api resource not found", func() {
			BeforeEach(func() {
				ctx = context.Background()
				service.EXPECT().RequestResource(poolID, info).Return(types.Resource{ID: "b", PoolID: poolID}, nil).After(
					service.EXPECT().RequestResource(poolID, info).Return(types.Resource{ID: "a", PoolID: poolID}, nil),
				)

				service.EXPECT().GetResource(poolID, "b").Return(types.Resource{ID: "b", PoolID: poolID, State: types.ResourceServing}, nil).After(
					service.EXPECT().GetResource(poolID, "a").Return(types.Resource{}, types.ResourceNotFoundErr),
				)
			})
			It("request resource again", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(res).To(Equal(types.Resource{ID: "b", PoolID: poolID, State: types.ResourceServing}))
			})
		})

		Context("api lock retry", func() {
			BeforeEach(func() {
				ctx, _ = context.WithTimeout(context.Background(), 1500*time.Millisecond)
				service.EXPECT().RequestResource(poolID, info).Return(types.Resource{}, plugin.AcquireLaterErr).Times(2)
			})
			It("err", func() {
				Expect(err.Error()).To(ContainSubstring("acquire later"))
			})
		})

		Context("api fail", func() {
			BeforeEach(func() {
				ctx, _ = context.WithTimeout(context.Background(), 6*time.Second)
				service.EXPECT().RequestResource(poolID, info).Return(types.Resource{ID: "a", PoolID: poolID}, nil)
				service.EXPECT().GetResource(poolID, "a").Return(types.Resource{}, errors.New("random")).AnyTimes()
			})
			It("err", func() {
				Expect(err.Error()).To(ContainSubstring("random"))
			})
		})

		Context("api success", func() {
			var resource types.Resource
			Context("with running resource", func() {
				BeforeEach(func() {
					ctx = context.Background()
					resource = types.Resource{ID: "a", PoolID: poolID, State: types.ResourceServing}
					service.EXPECT().RequestResource(poolID, info).Return(resource, nil)
				})
				It("err", func() {
					Expect(err).NotTo(HaveOccurred())
					Expect(res).To(Equal(resource))
				})
			})
			Context("with none running resource", func() {
				BeforeEach(func() {
					ctx = context.Background()
					resource = types.Resource{ID: "a", PoolID: poolID}
					service.EXPECT().RequestResource(poolID, info).Return(resource, nil)
					service.EXPECT().GetResource(poolID, "a").Return(
						types.Resource{ID: "a", PoolID: poolID, State: types.ResourceServing}, nil,
					).After(service.EXPECT().GetResource(poolID, "a").Return(
						resource, nil,
					).Times(1))
				})
				It("err", func() {
					Expect(err).NotTo(HaveOccurred())
					Expect(res).To(Equal(types.Resource{ID: "a", PoolID: poolID, State: types.ResourceServing}))
				})
			})
		})
	})

	Describe("Heartbeat", func() {
		JustBeforeEach(func() {
			err = client.Heartbeat(ctx, types.Resource{ID: "a", PoolID: poolID})
		})

		Context("call Heartbeat to api", func() {
			BeforeEach(func() {
				service.EXPECT().Heartbeat(poolID, "a", info).Return(nil)
			})
			It("success", func() {
				Expect(err).NotTo(HaveOccurred())
			})
		})
	})

	Describe("Info", func() {
		It("success", func() {
			Expect(client.Info()).To(Equal(info))
		})
	})
})

func TestClient(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Client Suite")
}
