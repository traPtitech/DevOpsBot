package bot

import (
	"context"
	"fmt"
	traqwsbot "github.com/traPtitech/traq-ws-bot"
	"strings"

	"github.com/kballard/go-shellquote"
	"github.com/samber/lo"
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
		ctx := context.Background()

		if p.Message.User.Bot {
			return // Ignore bots
		}
		if p.Message.ChannelID != config.C.Traq.ChannelID {
			return // DevOpsチャンネル以外からのメッセージは無視
		}

		args, err := shellquote.Split(p.Message.PlainText)
		if err != nil {
			_ = sendTRAQMessage(ctx, p.Message.ChannelID, fmt.Sprintf("invalid syntax: %s", err))
			_ = pushTRAQStamp(ctx, p.Message.ID, config.C.Stamps.BadCommand)
			return
		}
		_, argStart, ok := lo.FindIndexOf(args, func(arg string) bool { return strings.HasPrefix(arg, config.C.Prefix) })
		if !ok {
			return
		}
		args = args[argStart:]
		args[0] = strings.TrimPrefix(args[0], config.C.Prefix)

		cmdCtx := &Context{
			Context: ctx,
			P:       p,
			Args:    args,
		}
		err = cmds.execute(cmdCtx)
		if err != nil {
			cmdCtx.L().Error("failed to execute command", zap.Error(err))
		}
	}
}
