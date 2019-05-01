package config

import (
	"io/ioutil"
	"os"

	"github.com/rueian/godemand/plugin"
	"golang.org/x/xerrors"
	"gopkg.in/yaml.v2"
)

var PoolConfigNotFoundErr = xerrors.New("pool not found in config")

type Config struct {
	Plugins map[string]PluginConfig `yaml:"plugins"`
	Pools   map[string]PoolConfig   `yaml:"pools"`
}

type PluginConfig struct {
	Path string   `yaml:"path"`
	Envs []string `yaml:"envs"`
}

type PoolConfig struct {
	Plugin string                 `yaml:"plugin"`
	Params map[string]interface{} `yaml:"params"`
}

func (c *Config) GetPluginCmd() map[string]plugin.CmdParam {
	ret := make(map[string]plugin.CmdParam)
	for k, v := range c.Plugins {
		ret[k] = plugin.CmdParam{
			Name: k,
			Path: v.Path,
			Envs: v.Envs,
		}
	}
	return ret
}

func (c *Config) GetPool(poolID string) (pool PoolConfig, err error) {
	if pool, ok := c.Pools[poolID]; ok {
		return pool, nil
	}
	return PoolConfig{}, xerrors.Errorf("fail to get pool config %q: %w", poolID, PoolConfigNotFoundErr)
}

func LoadConfig(path string) (c *Config, err error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	b, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, err
	}
	c = &Config{}
	err = yaml.Unmarshal(b, c)
	if err != nil {
		return nil, err
	}
	return c, nil
}
