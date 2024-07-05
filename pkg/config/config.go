package config

import (
	"os"
	"strings"

	"github.com/spf13/viper"
)

var C Config

type Config struct {
	// Mode selects the origin of the bot.
	// Available values: "traq", "slack"
	Mode string `mapstructure:"mode" yaml:"mode"`
	// Traq is traQ-related authentication config
	Traq TraqConfig `mapstructure:"traq" yaml:"traq"`
	// Slack is slack-related authentication config
	Slack SlackConfig `mapstructure:"slack" yaml:"slack"`

	// Prefix is bot command prefix
	Prefix string `mapstructure:"prefix" yaml:"prefix"`
	// Stamps define which stamps to use for bot reactions
	Stamps Stamps `mapstructure:"stamps" yaml:"stamps"`

	// TmpDir is temporary directory in which executables from inlined config "command" are created
	TmpDir string `mapstructure:"tmpDir" yaml:"tmpDir"`
	// Templates define all command templates
	Templates []*CommandTemplateConfig `mapstructure:"templates" yaml:"templates"`
	// Commands define the command tree
	Commands []*CommandConfig `mapstructure:"commands" yaml:"commands"`

	// Servers define server auth information if this bot binary is used with "server" sub-command
	Servers ServersConfig `mapstructure:"servers" yaml:"servers"`
}

type TraqConfig struct {
	// Origin is WebSocket traQ origin. (example: wss://q.trap.jp)
	Origin string `mapstructure:"origin" yaml:"origin"`
	// ChannelID is the channel in which to await for commands
	ChannelID string `mapstructure:"channelID" yaml:"channelID"`
	// Token is traQ bot token
	Token string `mapstructure:"token" yaml:"token"`
}

type SlackConfig struct {
	OAuthToken string `mapstructure:"oauthToken" yaml:"oauthToken"`
	AppToken   string `mapstructure:"appToken" yaml:"appToken"`
	// ChannelID is the channel in which to await for commands
	ChannelID string `mapstructure:"channelID" yaml:"channelID"`
	// TrustedWorkflows is the list of bot IDs of trusted workflows.
	//
	// Trusted workflows are allowed to impersonate the execution user via adding user mention at the start of message.
	TrustedWorkflows []string `mapstructure:"trustedWorkflows" yaml:"trustedWorkflows"`
}

type Stamps struct {
	BadCommand string `mapstructure:"badCommand" yaml:"badCommand"`
	Forbid     string `mapstructure:"forbid" yaml:"forbid"`
	Success    string `mapstructure:"success" yaml:"success"`
	Failure    string `mapstructure:"failure" yaml:"failure"`
	Running    string `mapstructure:"running" yaml:"running"`
}

type CommandTemplateConfig struct {
	// Name is template name referenced by "templateRef" by each command
	Name string `mapstructure:"name" yaml:"name"`
	// Command is inlined executable file contents (usually a shell script)
	//
	// Cannot be set together with ExecFile.
	Command string `mapstructure:"command" yaml:"command"`
	// ExecFile is executable file name
	//
	// Cannot be set together with Command.
	ExecFile string `mapstructure:"execFile" yaml:"execFile"`
}

type CommandConfig struct {
	// Name is the name of this (sub-)command.
	Name string `mapstructure:"name" yaml:"name"`

	// TemplateRef is the name of CommandTemplateConfig.Name.
	TemplateRef string `mapstructure:"templateRef" yaml:"templateRef"`
	// Description should describe what this command does in one line.
	Description string `mapstructure:"description" yaml:"description"`
	// AllowArgs is a flag to allow passing extra user command arguments to exec arguments.
	AllowArgs bool `mapstructure:"allowArgs" yaml:"allowArgs"`
	// ArgsSyntax is an optional arguments syntax to display in help command.
	ArgsSyntax string `mapstructure:"argsSyntax" yaml:"argsSyntax"`
	// ArgsPrefix is always prefixed the arguments (before the user-provided arguments, if any) when executing the command template.
	ArgsPrefix []string `mapstructure:"argsPrefix" yaml:"argsPrefix"`
	// Operators is an optional list of user IDs (traQ IDs in traQ, member or bot IDs in Slack)
	// who are allowed to execute this command (and any sub-commands).
	// If left empty, everyone will be able to execute this command (and any sub-commands).
	Operators []string `mapstructure:"operators" yaml:"operators"`

	// SubCommands define any sub-commands under this command.
	// Note that Operators config is inherited.
	SubCommands []*CommandConfig `mapstructure:"subCommands" yaml:"subCommands"`
}

type ServersConfig struct {
	Conoha struct {
		Origin struct {
			Identity string `mapstructure:"identity" yaml:"identity"`
			Compute  string `mapstructure:"compute" yaml:"compute"`
		} `mapstructure:"origin" yaml:"origin"`
		Username string `mapstructure:"username" yaml:"username"`
		Password string `mapstructure:"password" yaml:"password"`
		TenantID string `mapstructure:"tenantID" yaml:"tenantID"`
	} `mapstructure:"conoha" yaml:"conoha"`
}

func init() {
	viper.SetDefault("mode", "traq")

	viper.SetDefault("traq.origin", "wss://q.trap.jp")
	viper.SetDefault("traq.channelID", "")
	viper.SetDefault("traq.token", "")

	viper.SetDefault("slack.oauthToken", "")
	viper.SetDefault("slack.appToken", "")
	viper.SetDefault("slack.channelID", "")
	viper.SetDefault("slack.trustedWorkflows", nil)

	viper.SetDefault("prefix", "/")

	viper.SetDefault("stamps.badCommand", "")
	viper.SetDefault("stamps.forbid", "")
	viper.SetDefault("stamps.success", "")
	viper.SetDefault("stamps.failure", "")
	viper.SetDefault("stamps.running", "")

	viper.SetDefault("tmpDir", "/commands")
	viper.SetDefault("templates", nil)
	viper.SetDefault("commands", nil)

	viper.SetDefault("servers.conoha.origin.identity", "https://identity.tyo1.conoha.io/")
	viper.SetDefault("servers.conoha.origin.compute", "https://compute.tyo1.conoha.io/")
	viper.SetDefault("servers.conoha.username", "")
	viper.SetDefault("servers.conoha.password", "")
	viper.SetDefault("servers.conoha.tenantID", "")
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
