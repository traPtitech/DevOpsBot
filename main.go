package main

import (
	"context"
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

	// 設定ファイル読み込み
	config, err = LoadConfig(getEnvOrDefault("CONFIG_FILE", "./config.yml"))
	if err != nil {
		logger.Fatal("failed to load config", zap.Error(err))
	}
	commands["service"] = config.Services
	commands["server"] = config.Servers
	commands["exec-log"] = &ExecLogCommand{}
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
	if err = SendTRAQMessage(context.Background(), config.DevOpsChannelID, fmt.Sprintf(":up: DevOpsBot `v%s` is ready", version)); err != nil {
		logger.Fatal("failed to send starting message", zap.Error(err))
	}
	logger.Info(fmt.Sprintf("DevOpsBot `v%s` is ready", version))

	err = bot.Start()
	if err != nil {
		logger.Fatal("starting ws bot", zap.Error(err))
	}
}
