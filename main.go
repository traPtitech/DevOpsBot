package main

import (
	"fmt"

	traqwsbot "github.com/traPtitech/traq-ws-bot"

	"go.uber.org/zap"
)

var (
	version = "UNKNOWN"
	logger  *zap.Logger
	bot     *traqwsbot.Bot
)

func main() {
	var err error

	// Initialize logger
	logger, err = zap.NewProduction()
	if err != nil {
		panic(err)
	}
	defer logger.Sync()

	logger.Info(fmt.Sprintf("DevOpsBot `v%s` initializing", version))

	// Load config
	err = LoadConfig()
	if err != nil {
		logger.Fatal("failed to load config", zap.Error(err))
	}

	// Register commands
	deployCmd, err := config.Commands.Deploy.Compile()
	if err != nil {
		logger.Fatal("compiling deploy cmd", zap.Error(err))
	}
	commands["deploy"] = deployCmd

	svrCmd, err := config.Commands.Servers.Compile()
	if err != nil {
		logger.Fatal("invalid servers config", zap.Error(err))
	}
	commands["server"] = svrCmd

	commands["help"] = &HelpCommand{}

	// Start bot
	bot, err = traqwsbot.NewBot(&traqwsbot.Options{
		AccessToken: config.Token,
		Origin:      config.TraqOrigin,
	})
	if err != nil {
		logger.Fatal("setting up traq-ws-bot", zap.Error(err))
	}

	bot.OnMessageCreated(BotMessageReceived)

	err = bot.Start()
	if err != nil {
		logger.Fatal("starting ws bot", zap.Error(err))
	}
}
