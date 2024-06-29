package bot

import (
	"context"
	"fmt"
	"github.com/traPtitech/DevOpsBot/pkg/bot/slack"
	"github.com/traPtitech/DevOpsBot/pkg/config"
	"github.com/traPtitech/DevOpsBot/pkg/domain"

	"go.uber.org/zap"

	"github.com/traPtitech/DevOpsBot/pkg/bot/traq"
)

func Run(ctx context.Context) error {
	// Initialize logger
	logger, err := zap.NewProduction()
	if err != nil {
		return err
	}
	defer logger.Sync()

	// Compile commands
	cmds, err := Compile()
	if err != nil {
		return fmt.Errorf("compiling commands: %w", err)
	}

	// Initialize bot
	var bot domain.Bot
	switch config.C.Mode {
	case "traq":
		bot, err = traq.NewBot(cmds, logger)
		if err != nil {
			return fmt.Errorf("creating traq bot: %w", err)
		}
	case "slack":
		bot, err = slack.NewBot(cmds, logger)
		if err != nil {
			return fmt.Errorf("creating slack bot: %w", err)
		}
	default:
		return fmt.Errorf("unknown bot mode: %s", config.C.Mode)
	}

	// Start bot
	err = bot.Start(ctx)
	if err != nil {
		return fmt.Errorf("starting bot: %w", err)
	}

	return nil
}
