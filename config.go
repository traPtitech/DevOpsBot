package main

import (
	"os"

	"gopkg.in/yaml.v2"
)

type Config struct {
	TraqOrigin              string   `yaml:"traqOrigin"`
	DevOpsChannelID         string   `yaml:"devOpsChannelId"`
	BotAccessToken          string   `yaml:"botAccessToken"`
	ConohaIdentityApiOrigin string   `yaml:"conohaIdentityApiOrigin"`
	ConohaComputeApiOrigin  string   `yaml:"conohaComputeApiOrigin"`
	ConohaApiUsername       string   `yaml:"conohaApiUsername"`
	ConohaApiPassword       string   `yaml:"conohaApiPassword"`
	ConohaTenantID          string   `yaml:"conohaTenantId"`
	LocalHostName           string   `yaml:"localhostName"`
	DefaultSSHUser          string   `yaml:"defaultSSHUser"`
	SSHPrivateKey           string   `yaml:"sshPrivateKey"`
	LogsDir                 string   `yaml:"logsDir"`
	Stamps                  Stamps   `yaml:"stamps"`
	Services                Services `yaml:"services"`
	Servers                 Servers  `yaml:"servers"`
}

type Stamps struct {
	Accept     string `yaml:"accept"`
	BadCommand string `yaml:"badCommand"`
	Forbid     string `yaml:"forbid"`
	Success    string `yaml:"success"`
	Failure    string `yaml:"failure"`
	Running    string `yaml:"running"`
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
