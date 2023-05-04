package main

import (
	"os"

	"gopkg.in/yaml.v2"
)

type Config struct {
	TraqOrigin      string         `yaml:"traqOrigin"`
	DevOpsChannelID string         `yaml:"devOpsChannelId"`
	BotAccessToken  string         `yaml:"botAccessToken"`
	Prefix          string         `yaml:"prefix"`
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
	Deploy  DeployConfig  `yaml:"deploy"`
	Servers ServersConfig `yaml:"servers"`
}

type DeployConfig struct {
	Templates []*DeployTemplateConfig `yaml:"templates"`
	Commands  []*DeployCommandConfig  `yaml:"commands"`
}

type DeployTemplateConfig struct {
	Name        string `yaml:"name"`
	Command     string `yaml:"command"`
	CommandFile string `yaml:"commandFile"`
}

type DeployCommandConfig struct {
	Name        string   `yaml:"name"`
	TemplateRef string   `yaml:"templateRef"`
	ArgsPrefix  []string `yaml:"argsPrefix"`
	Operators   []string `yaml:"operators"`
}

type ServersConfig struct {
	ConohaIdentityApiOrigin string                  `yaml:"conohaIdentityApiOrigin"`
	ConohaComputeApiOrigin  string                  `yaml:"conohaComputeApiOrigin"`
	ConohaApiUsername       string                  `yaml:"conohaApiUsername"`
	ConohaApiPassword       string                  `yaml:"conohaApiPassword"`
	ConohaTenantID          string                  `yaml:"conohaTenantId"`
	LogsDir                 string                  `yaml:"logsDir"`
	Servers                 []*ServerInstanceConfig `yaml:"servers"`
}

type ServerInstanceConfig struct {
	// Name サーバー名
	Name string `yaml:"name"`
	// ServerID サーバーID
	ServerID string `yaml:"serverId"`
	// Description サーバー説明
	Description string `yaml:"description"`
	// Operators コマンド実行可能なユーザーの名前のデフォルト配列
	Operators []string `yaml:"operators"`
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
