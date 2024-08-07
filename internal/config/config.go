package config

import (
	"sync"

	"github.com/sgostarter/libconfig"
)

type Config struct {
	Debug       bool   `json:"debug" yaml:"debug"`
	Listens     string `json:"listens" yaml:"listens"`
	Root        string `json:"root" yaml:"root"`
	NotifierURL string `json:"notifierURL" yaml:"notifierURL"`
}

var (
	_config Config
	_once   sync.Once
)

func GetConfig() *Config {
	_once.Do(func() {
		_, err := libconfig.Load("config.yaml", &_config)
		if err != nil {
			panic(err)
		}
	})

	return &_config
}
