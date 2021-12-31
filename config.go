package main

import (
	"gopkg.in/yaml.v2"
	"os"
)

type Config struct {
	BindAddr          string   `yaml:"bindAddr"`
	TraqOrigin        string   `yaml:"traqOrigin"`
	DevOpsBotOrigin   string   `yaml:"devOpsBotOrigin"`
	DevOpsChannelID   string   `yaml:"devOpsChannelId"`
	VerificationToken string   `yaml:"verificationToken"`
	BotAccessToken    string   `yaml:"botAccessToken"`
	ConohaApiOrigin   string   `yaml:"conohaApiOrigin"`
	ConohaApiToken    string   `yaml:"conohaApiToken"`
	TenantID          string   `yaml:"tenantId"`
	LocalHostName     string   `yaml:"localhostName"`
	DefaultSSHUser    string   `yaml:"defaultSSHUser"`
	SSHPrivateKey     string   `yaml:"sshPrivateKey"`
	LogsDir           string   `yaml:"logsDir"`
	Stamps            Stamps   `yaml:"stamps"`
	Services          Services `yaml:"services"`
	Servers           Servers  `yaml:"servers"`
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
