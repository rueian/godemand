package config

import (
	"io/ioutil"
	"os"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/rueian/godemand/plugin"
	"gopkg.in/yaml.v2"
)

var _ = Describe("LoadConfig", func() {
	var content []byte
	var config *Config
	var err error

	JustBeforeEach(func() {
		path, _ := tmp(content)
		config, err = LoadConfig(path)
		os.Remove(path)
	})

	Context("correct yaml", func() {
		BeforeEach(func() {
			content = []byte(`
---
plugins:
  plugin1:
     path: /something
     envs:
     - A=B
     - C=D
pools:
  pool1:
    plugin: plugin1
    params:
      str: something
      int: 1234
`)
		})

		It("parsed", func() {
			Expect(err).NotTo(HaveOccurred())
			Expect(*config).To(Equal(Config{
				Plugins: map[string]struct {
					Path string   `yaml:"path"`
					Envs []string `yaml:"envs"`
				}{
					"plugin1": {
						Path: "/something",
						Envs: []string{"A=B", "C=D"},
					},
				},
				Pools: map[string]struct {
					Plugin string                 `yaml:"plugin"`
					Params map[string]interface{} `yaml:"params"`
				}{
					"pool1": {
						Plugin: "plugin1",
						Params: map[string]interface{}{
							"str": "something",
							"int": 1234,
						},
					},
				},
			}))
		})

		Describe("GetPluginCmd", func() {
			It("turn config into map of CmdParam", func() {
				Expect(config.GetPluginCmd()).To(Equal(map[string]plugin.CmdParam{
					"plugin1": {
						Name: "plugin1",
						Path: "/something",
						Envs: []string{"A=B", "C=D"},
					},
				}))
			})
		})
	})

	Context("malformed", func() {
		BeforeEach(func() {
			content = []byte("any")
		})
		It("failed", func() {
			_, ok := err.(*yaml.TypeError)
			Expect(ok).To(BeTrue())
		})
	})

	Context("no config", func() {
		It("file not found err", func() {
			_, err = LoadConfig("anything")
			Expect(os.IsNotExist(err)).To(BeTrue())
		})
	})
})

func tmp(content []byte) (name string, err error) {
	f, err := ioutil.TempFile("", "example")
	if err != nil {
		return "", err
	}
	defer f.Close()
	if _, err := f.Write(content); err != nil {
		return "", err
	}
	return f.Name(), nil
}

func TestConfig(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Config Suite")
}
