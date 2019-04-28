package plugin

import (
	"context"
	"encoding/json"
	"errors"
	"math/rand"
	"net/rpc"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/rueian/godemand/plugin/mock"
	"github.com/rueian/godemand/types"
	"golang.org/x/xerrors"
)

var _ = Describe("Server", func() {
	var server *Server
	var ctrl *gomock.Controller
	var controller *mock.MockController

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		controller = mock.NewMockController(ctrl)
		server = &Server{controller: controller}
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	Describe("ProtocolVersion", func() {
		It("return current protocol version", func() {
			var version int
			err := server.ProtocolVersion(nil, &version)
			Expect(err).NotTo(HaveOccurred())
			Expect(version).To(Equal(CurrentProtocolVersion))
		})
	})

	var mockPool types.ResourcePool
	var mockRes types.Resource
	var mockParams map[string]interface{}
	var mockRetRes types.Resource
	var mockRetErr error
	var err error
	var ret types.Resource

	BeforeEach(func() {
		mockPool = types.ResourcePool{
			ID: "a",
			Resources: map[string]types.Resource{
				"b": makeResource(),
			},
		}
		mockRes = makeResource()
		mockParams = makeMeta()
		mockRetRes = makeResource()
		mockRetErr = errors.New("err")
		err = nil
		ret = types.Resource{}
	})

	Describe("FindResource", func() {
		JustBeforeEach(func() {
			controller.EXPECT().FindResource(mockPool, mockParams).Return(mockRetRes, mockRetErr)

			var in, out []byte
			in, _ = json.Marshal(FindResourceArgs{Pool: mockPool, Params: mockParams})
			err = server.FindResource(&in, &out)
			if err == nil {
				Expect(json.Unmarshal(out, &ret)).NotTo(HaveOccurred())
			}
		})

		Context("without err", func() {
			BeforeEach(func() {
				mockRetErr = nil
			})

			It("convert args and returns", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(ret).To(Equal(mockRetRes))
			})
		})

		Context("with err", func() {
			It("convert args and returns", func() {
				Expect(err).To(Equal(mockRetErr))
			})
		})

	})

	Describe("SyncResource", func() {
		JustBeforeEach(func() {
			controller.EXPECT().SyncResource(mockRes, mockParams).Return(mockRetRes, mockRetErr)

			var in, out []byte
			in, _ = json.Marshal(SyncResourceArgs{Resource: mockRes, Params: mockParams})
			err = server.SyncResource(&in, &out)
			if err == nil {
				Expect(json.Unmarshal(out, &ret)).NotTo(HaveOccurred())
			}
		})

		Context("without err", func() {
			BeforeEach(func() {
				mockRetErr = nil
			})

			It("convert args and returns", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(ret).To(Equal(mockRetRes))
			})
		})

		Context("with err", func() {
			It("convert args and returns", func() {
				Expect(err).To(Equal(mockRetErr))
			})
		})
	})
})

var _ = Describe("Serve", func() {
	var ctrl *gomock.Controller
	var controller *mock.MockController
	var ctx context.Context
	var cancel context.CancelFunc
	var doneCh chan error

	JustBeforeEach(func() {
		doneCh = make(chan error)
		ctrl = gomock.NewController(GinkgoT())
		controller = mock.NewMockController(ctrl)
		ctx, cancel = context.WithCancel(context.Background())
		go func() {
			doneCh <- Serve(ctx, controller)
			close(doneCh)
		}()
	})

	AfterEach(func() {
		cancel()
	})

	Context("without port", func() {
		BeforeEach(func() {
			os.Setenv(TCPPortEnvName, "")
		})

		It("fail to start server", func() {
			Expect(xerrors.Is(<-doneCh, TCPPortNotIntegerErr)).To(BeTrue())
		})
	})

	Context("with port", func() {
		var port string
		BeforeEach(func() {
			port, _ = getFreePort()
			os.Setenv(TCPPortEnvName, port)
		})

		It("start server", func() {
			cancel()
			Expect(<-doneCh).NotTo(HaveOccurred())
		})

		It("accept connection", func() {
			client, err := rpc.Dial("tcp", "localhost:"+port)
			Expect(err).NotTo(HaveOccurred())
			defer client.Close()

			var version int
			err = client.Call(RPCServerName+".ProtocolVersion", 0, &version)
			Expect(err).NotTo(HaveOccurred())
			Expect(version).To(Equal(CurrentProtocolVersion))
		})
	})

})

func makeResource() types.Resource {
	return types.Resource{
		ID:          strconv.Itoa(rand.Int()),
		Meta:        makeMeta(),
		State:       types.ResourcePending,
		StateChange: time.Now().Truncate(time.Millisecond),
		Clients: []types.Client{
			{
				ID:        strconv.Itoa(rand.Int()),
				Heartbeat: time.Now().Truncate(time.Millisecond),
				Meta:      makeMeta(),
			},
		},
	}
}

func makeMeta() map[string]interface{} {
	return map[string]interface{}{
		strconv.Itoa(rand.Int()): strconv.Itoa(rand.Int()),
	}
}

func TestPluginServer(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "PluginServer Suite")
}
