package plugin

import (
	"os/exec"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/rueian/godemand/types"
	"golang.org/x/xerrors"
)

var _ = Describe("LaunchPad", func() {
	var launchpad *Launchpad
	var params map[string]types.CmdParam
	var err error

	BeforeEach(func() {
		launchpad = NewLaunchpad()
		params = map[string]types.CmdParam{
			"puppet": {
				Name: "PuppetController",
				Path: "./mock/server/puppet",
				Envs: []string{"CUSTOM=1"},
			},
		}
	})

	AfterEach(func() {
		launchpad.Close()
	})

	JustBeforeEach(func() {
		err = launchpad.SetLaunchers(params)
	})

	Describe("SetLaunchers", func() {
		It("launched", func() {
			Expect(err).NotTo(HaveOccurred())
			for k := range params {
				Expect(launchpad.launchers).To(HaveKey(k))
			}
		})

		Context("with failed params", func() {
			BeforeEach(func() {
				params["failed"] = types.CmdParam{Path: "notfound"}
			})

			It("return error", func() {
				Expect(err.Error()).To(ContainSubstring(exec.ErrNotFound.Error()))
			})

			It("still success with worked param", func() {
				Expect(launchpad.launchers).To(HaveKey("puppet"))
			})
		})

		Context("with same input", func() {
			JustBeforeEach(func() {
				err = launchpad.SetLaunchers(params)
			})

			It("no error", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(launchpad.launchers["puppet"].CmdParam).To(Equal(params["puppet"]))
			})
		})

		Context("with update", func() {
			JustBeforeEach(func() {
				params["puppet"] = types.CmdParam{
					Name: "PuppetController",
					Path: "./mock/server/puppet",
					Envs: []string{"CUSTOM=2"},
				}
				err = launchpad.SetLaunchers(params)
			})

			It("no error", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(launchpad.launchers["puppet"].CmdParam.Envs).To(ConsistOf("CUSTOM=2"))
			})
		})

		Context("with update failed", func() {
			JustBeforeEach(func() {
				params["puppet"] = types.CmdParam{Path: "notfound"}
				err = launchpad.SetLaunchers(params)
			})

			It("error", func() {
				Expect(err.Error()).To(ContainSubstring(exec.ErrNotFound.Error()))
				Expect(launchpad.launchers).NotTo(HaveKey("puppet"))
			})
		})

		Context("with removed param", func() {
			JustBeforeEach(func() {
				params = map[string]types.CmdParam{}
				err = launchpad.SetLaunchers(params)
			})

			It("remove launcher", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(launchpad.launchers).NotTo(HaveKey("puppet"))
			})
		})
	})

	Describe("GetController", func() {
		It("get launched controller", func() {
			controller, _ := launchpad.GetController("puppet")
			Expect(controller).NotTo(BeNil())
		})
		It("get ControllerNotFoundErr if not launched", func() {
			_, err := launchpad.GetController("random")
			Expect(xerrors.Is(err, ControllerNotFoundErr)).To(BeTrue())
		})
	})

	Describe("Close", func() {
		It("close all launchers", func() {
			launchpad.Close()
			Expect(launchpad.launchers).To(HaveLen(0))
		})
	})
})
