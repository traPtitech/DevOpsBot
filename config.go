package main

import (
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"sync"
)

type Config struct {
	TraqOrigin      string
	DevOpsChannelID string
	Stamps          struct {
		ThumbsUp string
	}
	Deploys map[string]DeployConfig
}

type DeployConfig struct {
	Command     string
	CommandArgs []string
	Operators   []string
	isRunning   bool
	mx          sync.Mutex
}

func LoadConfig(configFile string) (*Config, error) {
	b, err := ioutil.ReadFile(configFile)
	if err != nil {
		return nil, err
	}

	var c Config
	if err := yaml.Unmarshal(b, &c); err != nil {
		return nil, err
	}

	return &c, nil
}
