package bot

import (
	"context"
	"fmt"
	traqwsbot "github.com/traPtitech/traq-ws-bot"
	"strings"

	"github.com/kballard/go-shellquote"
	"github.com/traPtitech/traq-ws-bot/payload"
	"go.uber.org/zap"

	"github.com/traPtitech/DevOpsBot/pkg/config"
)

var (
	logger *zap.Logger
	bot    *traqwsbot.Bot
)

func Run() error {
	var err error
	// Initialize logger
	logger, err = zap.NewProduction()
	if err != nil {
		return err
	}
	defer logger.Sync()

	// Register commands
	cmds, err := Compile()
	if err != nil {
		return fmt.Errorf("compiling commands: %w", err)
	}

	// Start bot
	bot, err = traqwsbot.NewBot(&traqwsbot.Options{
		AccessToken: config.C.Traq.Token,
		Origin:      config.C.Traq.Origin,
	})
	if err != nil {
		return fmt.Errorf("creating bot: %w", err)
	}

	bot.OnMessageCreated(botMessageReceived(cmds))

	err = bot.Start()
	if err != nil {
		return fmt.Errorf("starting bot: %w", err)
	}

	return nil
}

// botMessageReceived BOTのMESSAGE_CREATEDイベントハンドラ
func botMessageReceived(cmds *RootCommand) func(p *payload.MessageCreated) {
	return func(p *payload.MessageCreated) {
		// Validate command execution context
		if p.Message.User.Bot {
			return // Ignore bots
		}
		if p.Message.ChannelID != config.C.Traq.ChannelID {
			return // 指定チャンネル以外からのメッセージは無視
		}
		if !strings.HasPrefix(p.Message.PlainText, config.C.Prefix) {
			return // Command prefix does not match
		}

		// Prepare command args
		ctx := &traqContext{
			Context: context.Background(),
			p:       p,
		}
		prefixStripped := strings.TrimPrefix(p.Message.PlainText, config.C.Prefix)
		args, err := shellquote.Split(prefixStripped)
		if err != nil {
			_ = ctx.ReplyBad(fmt.Sprintf("failed to parse arguments: %v", err))
			return
		}
		ctx.args = args

		// Execute
		err = cmds.execute(ctx)
		if err != nil {
			ctx.L().Error("failed to execute command", zap.Error(err))
		}
	}
}
