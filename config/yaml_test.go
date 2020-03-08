package config

import (
	"io/ioutil"
	"os"
	"testing"

	"errors"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/rueian/godemand/types"
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
				Plugins: map[string]PluginConfig{
					"plugin1": {
						Path: "/something",
						Envs: []string{"A=B", "C=D"},
					},
				},
				Pools: map[string]PoolConfig{
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

		Describe("GetPoolConfig", func() {
			var pool PoolConfig
			var id string

			JustBeforeEach(func() {
				pool, err = config.GetPool(id)
			})

			Context("exist", func() {
				BeforeEach(func() {
					id = "pool1"
				})
				It("loaded", func() {
					Expect(err).NotTo(HaveOccurred())
					Expect(pool).To(Equal(config.Pools[id]))
				})
			})
			Context("not exist", func() {
				BeforeEach(func() {
					id = "random"
				})
				It("not found", func() {
					Expect(errors.Is(err, PoolConfigNotFoundErr)).To(BeTrue())
				})
			})
		})

		Describe("GetPluginCmd", func() {
			It("turn config into map of CmdParam", func() {
				Expect(config.GetPluginCmd()).To(Equal(map[string]types.CmdParam{
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
