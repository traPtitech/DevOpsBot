package main

import (
	"os"

	"gopkg.in/yaml.v2"
)

type Config struct {
	TraqOrigin      string         `yaml:"traqOrigin"`
	DevOpsChannelID string         `yaml:"devOpsChannelId"`
	BotAccessToken  string         `yaml:"botAccessToken"`
	Stamps          Stamps         `yaml:"stamps"`
	Commands        CommandsConfig `yaml:"commands"`
}

type Stamps struct {
	Accept     string `yaml:"accept"`
	BadCommand string `yaml:"badCommand"`
	Forbid     string `yaml:"forbid"`
	Success    string `yaml:"success"`
	Failure    string `yaml:"failure"`
	Running    string `yaml:"running"`
}

type CommandsConfig struct {
	Services ServicesConfig `yaml:"services"`
	Servers  ServersConfig  `yaml:"servers"`
}

type ServicesConfig struct {
	LogsDir        string     `yaml:"logsDir"`
	LocalHostName  string     `yaml:"localhostName"`
	DefaultSSHUser string     `yaml:"defaultSSHUser"`
	SSHPrivateKey  string     `yaml:"sshPrivateKey"`
	Services       []*Service `yaml:"services"`
}

type ServersConfig struct {
	ConohaIdentityApiOrigin string    `yaml:"conohaIdentityApiOrigin"`
	ConohaComputeApiOrigin  string    `yaml:"conohaComputeApiOrigin"`
	ConohaApiUsername       string    `yaml:"conohaApiUsername"`
	ConohaApiPassword       string    `yaml:"conohaApiPassword"`
	ConohaTenantID          string    `yaml:"conohaTenantId"`
	LogsDir                 string    `yaml:"logsDir"`
	Servers                 []*Server `yaml:"servers"`
}

func LoadConfig(configFile string) (*Config, error) {
	f, err := os.Open(configFile)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var c Config
	if err := yaml.NewDecoder(f).Decode(&c); err != nil {
		return nil, err
	}
	return &c, nil
}
