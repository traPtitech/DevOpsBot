package bot

import (
	"context"
	"fmt"

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
	bot, err := traq.NewBot(cmds, logger)
	if err != nil {
		return fmt.Errorf("creating bot: %w", err)
	}

	// Start bot
	err = bot.Start(ctx)
	if err != nil {
		return fmt.Errorf("starting bot: %w", err)
	}

	return nil
}
