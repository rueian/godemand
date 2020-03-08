package plugin

import (
	"bufio"
	"bytes"
	"errors"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/rueian/godemand/types"
	"log"
	"strings"
	"syscall"
)

var _ = Describe("PluginLauncher", func() {
	var launcher *Launcher
	var cmdParam types.CmdParam
	var buf *bytes.Buffer
	var controller types.Controller
	var err error

	BeforeEach(func() {
		cmdParam = types.CmdParam{
			Name: "PuppetController",
			Path: "./mock/server/puppet",
		}
		buf = &bytes.Buffer{}
	})

	JustBeforeEach(func() {
		logger := log.New(buf, "", log.LstdFlags)
		launcher = NewLauncher(cmdParam, logger)
		controller, err = launcher.Launch()
	})

	JustAfterEach(func() {
		launcher.Close()
	})

	Context("plugin not found", func() {
		BeforeEach(func() {
			cmdParam.Path = "any path"
		})

		It("fail with exec error", func() {
			Expect(strings.Contains(err.Error(), "executable file not found")).To(BeTrue())
		})
	})

	Context("with custom envs", func() {
		BeforeEach(func() {
			cmdParam.Envs = []string{"CUSTOM_1=1", "CUSTOM_2=2"}
		})

		It("pass custom envs", func() {
			Expect(err).NotTo(HaveOccurred())

			scanner := bufio.NewScanner(buf)
			scanner.Scan()
			line := scanner.Text()

			for _, env := range cmdParam.Envs {
				Expect(strings.Contains(line, env)).To(BeTrue())
			}
		})
	})

	Context("with non supported protocol version", func() {
		BeforeEach(func() {
			MinimumProtocolVersion = 2 // temporary make it higher
		})
		AfterEach(func() {
			MinimumProtocolVersion = 1 // change it back
		})
		It("reject with err", func() {
			Expect(errors.Is(err, ProtocolVersionTooOldErr)).To(BeTrue())
		})
	})

	Context("with controller", func() {
		It("can get err", func() {
			_, err := controller.FindResource(types.ResourcePool{}, map[string]interface{}{
				"err": "FindResourceErr",
			})
			Expect(err.Error()).To(Equal("FindResourceErr"))
			_, err = controller.SyncResource(types.Resource{}, map[string]interface{}{
				"err": "SyncResourceErr",
			})
			Expect(err.Error()).To(Equal("SyncResourceErr"))
		})
		It("can read param and get ret", func() {
			fakeRes := makeResource()
			res, err := controller.FindResource(types.ResourcePool{
				Resources: map[string]types.Resource{fakeRes.ID: fakeRes},
			}, map[string]interface{}{
				"ret": fakeRes.ID,
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(res).To(Equal(fakeRes))
			res, err = controller.SyncResource(fakeRes, map[string]interface{}{
				"state": types.ResourceDeleted,
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(res.State).To(Equal(types.ResourceDeleted))
		})
	})

	Context("with plugin terminated", func() {
		JustBeforeEach(func() {
			launcher.command.Process.Signal(syscall.SIGINT)
			err = launcher.Err()
		})

		It("get return code via Err()", func() {
			Expect(err.Error()).To(ContainSubstring("exit status 1"))
		})

		It("get same err via Err()", func() {
			Expect(err).To(Equal(launcher.Err()))
		})

		It("capture stdout and stderr", func() {
			var line string
			scanner := bufio.NewScanner(buf)
			for scanner.Scan() {
				line += scanner.Text()
			}
			Expect(line).To(ContainSubstring(ListenedSign))                       // stdout
			Expect(line).To(ContainSubstring("use of closed network connection")) // stderr
		})
	})
})
