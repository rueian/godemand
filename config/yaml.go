package config

import (
	"io/ioutil"
	"os"

	"github.com/rueian/godemand/plugin"
	"gopkg.in/yaml.v2"
)

type Config struct {
	Plugins map[string]struct {
		Path string   `yaml:"path"`
		Envs []string `yaml:"envs"`
	} `yaml:"plugins"`
	Pools map[string]struct {
		Plugin string                 `yaml:"plugin"`
		Params map[string]interface{} `yaml:"params"`
	} `yaml:"pools"`
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
