package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/rueian/godemand/config"
	"github.com/rueian/godemand/plugin"
	"github.com/rueian/godemand/types"
	"github.com/rueian/godemand/types/mock"
)

var _ = Describe("NewHTTPMux", func() {
	var ctrl *gomock.Controller
	var service *mock.MockService
	var mux *http.ServeMux
	var req *http.Request
	var rec *httptest.ResponseRecorder
	var endpoint string
	var form url.Values
	var poolID string
	var resID string
	var client types.Client
	var clientJson string
	var res types.Resource
	var err error

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		service = mock.NewMockService(ctrl)
		mux = NewHTTPMux(service)
		rec = httptest.NewRecorder()
		form = url.Values{}
		poolID = "pool1"
		resID = "res1"
		client = types.Client{ID: "client1", Meta: map[string]interface{}{"ip": "0.0.0.0"}}
		bc, _ := json.Marshal(client)
		clientJson = string(bc)
		res = types.Resource{}
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	JustBeforeEach(func() {
		req = httptest.NewRequest("POST", endpoint, strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		mux.ServeHTTP(rec, req)
		if rec.Body.Len() > 0 {
			err = json.Unmarshal(rec.Body.Bytes(), &res)
		}
	})

	Describe("/RequestResource", func() {
		BeforeEach(func() {
			endpoint = "/RequestResource"
		})
		Context("no client", func() {
			BeforeEach(func() {
				form.Add("poolID", poolID)
			})
			It("err", func() {
				Expect(rec.Code).To(Equal(422))
			})
		})
		Context("param correct", func() {
			BeforeEach(func() {
				form.Add("poolID", poolID)
				form.Add("client", clientJson)
			})

			for _, c := range []errorCase{
				makeErrorCase("lock fail", 429, types.Resource{}, plugin.AcquireLaterErr),
				makeErrorCase("no config", 500, types.Resource{}, config.PoolConfigNotFoundErr),
				makeErrorCase("no plugin", 500, types.Resource{}, plugin.ControllerNotFoundErr),
				makeErrorCase("other err", 500, types.Resource{}, errors.New("random")),
			} {
				func(c errorCase) {
					Context(c.Name, func() {
						BeforeEach(func() {
							service.EXPECT().RequestResource(poolID, client).Return(c.Returns...)
						})
						It("err", func() {
							Expect(rec.Code).To(Equal(c.ExpectCode))
						})
					})
				}(c)
			}

			Context("success", func() {
				BeforeEach(func() {
					service.EXPECT().RequestResource(poolID, client).Return(types.Resource{ID: resID}, nil)
				})
				It("got res", func() {
					Expect(rec.Code).To(Equal(200))
					Expect(err).NotTo(HaveOccurred())
					Expect(res).To(Equal(types.Resource{ID: resID}))
				})
			})
		})
	})

	Describe("/GetResource", func() {
		BeforeEach(func() {
			endpoint = "/GetResource"
		})
		Context("param correct", func() {
			BeforeEach(func() {
				form.Add("poolID", poolID)
				form.Add("id", resID)
			})

			for _, c := range []errorCase{
				makeErrorCase("no res", 404, types.Resource{}, types.ResourceNotFoundErr),
				makeErrorCase("other err", 500, types.Resource{}, errors.New("random")),
			} {
				func(c errorCase) {
					Context(c.Name, func() {
						BeforeEach(func() {
							service.EXPECT().GetResource(poolID, resID).Return(c.Returns...)
						})
						It("err", func() {
							Expect(rec.Code).To(Equal(c.ExpectCode))
						})
					})
				}(c)
			}

			Context("success", func() {
				BeforeEach(func() {
					service.EXPECT().GetResource(poolID, resID).Return(types.Resource{ID: resID}, nil)
				})
				It("got res", func() {
					Expect(rec.Code).To(Equal(200))
					Expect(err).NotTo(HaveOccurred())
					Expect(res).To(Equal(types.Resource{ID: resID}))
				})
			})
		})
	})

	Describe("/Heartbeat", func() {
		BeforeEach(func() {
			endpoint = "/Heartbeat"
		})
		Context("no client", func() {
			BeforeEach(func() {
				form.Add("poolID", poolID)
				form.Add("id", resID)
			})
			It("err", func() {
				Expect(rec.Code).To(Equal(422))
			})
		})
		Context("param correct", func() {
			BeforeEach(func() {
				form.Add("poolID", poolID)
				form.Add("id", resID)
				form.Add("client", clientJson)
			})

			for _, c := range []errorCase{
				makeErrorCase("no res", 404, types.ResourceNotFoundErr),
				makeErrorCase("lock fail", 429, plugin.AcquireLaterErr),
				makeErrorCase("other err", 500, errors.New("random")),
			} {
				func(c errorCase) {
					Context(c.Name, func() {
						BeforeEach(func() {
							service.EXPECT().Heartbeat(poolID, resID, client).Return(c.Returns...)
						})
						It("err", func() {
							Expect(rec.Code).To(Equal(c.ExpectCode))
						})
					})
				}(c)
			}

			Context("success", func() {
				BeforeEach(func() {
					service.EXPECT().Heartbeat(poolID, resID, client).Return(nil)
				})
				It("got res", func() {
					Expect(rec.Code).To(Equal(200))
				})
			})
		})
	})
})

type errorCase struct {
	Name       string
	Returns    []interface{}
	ExpectCode int
}

func makeErrorCase(name string, expect int, returns ...interface{}) errorCase {
	return errorCase{name, returns, expect}
}
