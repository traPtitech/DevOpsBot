package config

import (
	"os"
	"strings"

	"github.com/spf13/viper"
)

var C Config

type Config struct {
	TraqOrigin string         `mapstructure:"traqOrigin" yaml:"traqOrigin"`
	ChannelID  string         `mapstructure:"channelID" yaml:"channelID"`
	Token      string         `mapstructure:"token" yaml:"token"`
	Prefix     string         `mapstructure:"prefix" yaml:"prefix"`
	Stamps     Stamps         `mapstructure:"stamps" yaml:"stamps"`
	Commands   CommandsConfig `mapstructure:"commands" yaml:"commands"`
}

type Stamps struct {
	Accept     string `mapstructure:"accept" yaml:"accept"`
	BadCommand string `mapstructure:"badCommand" yaml:"badCommand"`
	Forbid     string `mapstructure:"forbid" yaml:"forbid"`
	Success    string `mapstructure:"success" yaml:"success"`
	Failure    string `mapstructure:"failure" yaml:"failure"`
	Running    string `mapstructure:"running" yaml:"running"`
}

type CommandsConfig struct {
	Deploy  DeployConfig  `mapstructure:"deploy" yaml:"deploy"`
	Servers ServersConfig `mapstructure:"servers" yaml:"servers"`
}

type DeployConfig struct {
	CommandsDir string                  `mapstructure:"commandsDir" yaml:"commandsDir"`
	Templates   []*DeployTemplateConfig `mapstructure:"templates" yaml:"templates"`
	Commands    []*DeployCommandConfig  `mapstructure:"commands" yaml:"commands"`
}

type DeployTemplateConfig struct {
	Name        string `mapstructure:"name" yaml:"name"`
	Command     string `mapstructure:"command" yaml:"command"`
	CommandFile string `mapstructure:"commandFile" yaml:"commandFile"`
}

type DeployCommandConfig struct {
	Name        string   `mapstructure:"name" yaml:"name"`
	TemplateRef string   `mapstructure:"templateRef" yaml:"templateRef"`
	Description string   `mapstructure:"description" yaml:"description"`
	ArgsSyntax  string   `mapstructure:"argsSyntax" yaml:"argsSyntax"`
	ArgsPrefix  []string `mapstructure:"argsPrefix" yaml:"argsPrefix"`
	Operators   []string `mapstructure:"operators" yaml:"operators"`
}

type ServersConfig struct {
	Servers []*ServerInstanceConfig `mapstructure:"servers" yaml:"servers"`
	Conoha  struct {
		Origin struct {
			Identity string `mapstructure:"identity" yaml:"identity"`
			Compute  string `mapstructure:"compute" yaml:"compute"`
		} `mapstructure:"origin" yaml:"origin"`
		Username string `mapstructure:"username" yaml:"username"`
		Password string `mapstructure:"password" yaml:"password"`
		TenantID string `mapstructure:"tenantID" yaml:"tenantID"`
	} `mapstructure:"conoha" yaml:"conoha"`
}

type ServerInstanceConfig struct {
	Name        string   `mapstructure:"name" yaml:"name"`
	ServerID    string   `mapstructure:"serverID" yaml:"serverID"`
	Description string   `mapstructure:"description" yaml:"description"`
	Operators   []string `mapstructure:"operators" yaml:"operators"`
}

func init() {
	viper.SetDefault("traqOrigin", "wss://q.trap.jp")
	viper.SetDefault("channelID", "")
	viper.SetDefault("token", "")
	viper.SetDefault("prefix", "/")

	viper.SetDefault("stamps.accept", "")
	viper.SetDefault("stamps.badCommand", "")
	viper.SetDefault("stamps.forbid", "")
	viper.SetDefault("stamps.success", "")
	viper.SetDefault("stamps.failure", "")
	viper.SetDefault("stamps.running", "")

	viper.SetDefault("commands.deploy.commandsDir", "/commands")
	viper.SetDefault("commands.deploy.templates", nil)
	viper.SetDefault("commands.deploy.commands", nil)

	viper.SetDefault("commands.servers.servers", nil)
	viper.SetDefault("commands.servers.conoha.origin.identity", "https://identity.tyo1.conoha.io/")
	viper.SetDefault("commands.servers.conoha.origin.compute", "https://compute.tyo1.conoha.io/")
	viper.SetDefault("commands.servers.conoha.username", "")
	viper.SetDefault("commands.servers.conoha.password", "")
	viper.SetDefault("commands.servers.conoha.tenantID", "")
}

func init() {
	configFile := os.Getenv("CONFIG_FILE")
	if configFile == "" {
		configFile = "./config.yaml"
	}
	viper.SetConfigFile(configFile)
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	err := viper.ReadInConfig()
	if err != nil {
		panic(err)
	}
	err = viper.Unmarshal(&C)
	if err != nil {
		panic(err)
	}
}
