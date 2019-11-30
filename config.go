package main

import (
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"sync"
)

type Config struct {
	BindAddr          string                   `yaml:"bindAddr"`
	TraqOrigin        string                   `yaml:"traqOrigin"`
	DevOpsChannelID   string                   `yaml:"devOpsChannelId"`
	VerificationToken string                   `yaml:"verificationToken"`
	BotAccessToken    string                   `yaml:"botAccessToken"`
	LogsDir           string                   `yaml:"logsDir"`
	Stamps            Stamps                   `yaml:"stamps"`
	Deploys           map[string]*DeployConfig `yaml:"deploys"`
}

type Stamps struct {
	Accept     string `yaml:"accept"`
	BadCommand string `yaml:"badCommand"`
	Forbid     string `yaml:"forbid"`
	Success    string `yaml:"success"`
	Failure    string `yaml:"failure"`
}

type DeployConfig struct {
	Name             string   `yaml:"-"`
	Command          string   `yaml:"command"`
	CommandArgs      []string `yaml:"commandArgs"`
	WorkingDirectory string   `yaml:"workingDir"`
	Operators        []string `yaml:"operators"`
	isRunning        bool
	mx               sync.Mutex
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

	for name, deployConfig := range c.Deploys {
		deployConfig.Name = name
	}
	return &c, nil
}
