package main

import (
	"fmt"

	traqwsbot "github.com/traPtitech/traq-ws-bot"

	"go.uber.org/zap"
)

var (
	version = "UNKNOWN"
	config  *Config
	logger  *zap.Logger
	bot     *traqwsbot.Bot
)

func main() {
	var err error

	// ロガー初期化
	logger, err = zap.NewProduction()
	if err != nil {
		panic(err)
	}
	defer logger.Sync()

	logger.Info(fmt.Sprintf("DevOpsBot `v%s` initializing", version))

	// 設定ファイル読み込み
	config, err = LoadConfig(getEnvOrDefault("CONFIG_FILE", "./config.yml"))
	if err != nil {
		logger.Fatal("failed to load config", zap.Error(err))
	}

	// Register commands
	deployCmd, err := config.Commands.Deploy.Compile()
	if err != nil {
		logger.Fatal("compiling deploy cmd", zap.Error(err))
	}
	commands["deploy"] = deployCmd

	svcCmd, err := config.Commands.Services.Compile()
	if err != nil {
		logger.Fatal("invalid services config", zap.Error(err))
	}
	commands["service"] = svcCmd

	svrCmd, err := config.Commands.Servers.Compile()
	if err != nil {
		logger.Fatal("invalid servers config", zap.Error(err))
	}
	commands["server"] = svrCmd

	commands["exec-log"] = &ExecLogCommand{svc: svcCmd, svr: svrCmd}
	commands["version"] = &VersionCommand{}

	bot, err = traqwsbot.NewBot(&traqwsbot.Options{
		AccessToken: config.BotAccessToken,
		Origin:      config.TraqOrigin,
	})
	if err != nil {
		logger.Fatal("setting up traq-ws-bot", zap.Error(err))
	}

	bot.OnMessageCreated(BotMessageReceived)

	// 起動
	err = bot.Start()
	if err != nil {
		logger.Fatal("starting ws bot", zap.Error(err))
	}
}
