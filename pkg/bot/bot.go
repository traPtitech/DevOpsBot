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
	deployCmd, err := compileDeployConfig(&config.C.Commands.Deploy)
	if err != nil {
		return fmt.Errorf("compiling deploy command: %w", err)
	}
	commands["deploy"] = deployCmd

	commands["help"] = &HelpCommand{}

	// Start bot
	bot, err = traqwsbot.NewBot(&traqwsbot.Options{
		AccessToken: config.C.Token,
		Origin:      config.C.TraqOrigin,
	})
	if err != nil {
		return fmt.Errorf("creating bot: %w", err)
	}

	bot.OnMessageCreated(botMessageReceived)

	err = bot.Start()
	if err != nil {
		return fmt.Errorf("starting bot: %w", err)
	}

	return nil
}

// botMessageReceived BOTのMESSAGE_CREATEDイベントハンドラ
func botMessageReceived(p *payload.MessageCreated) {
	ctx := context.Background()

	if p.Message.User.Bot {
		return // Ignore bots
	}
	if p.Message.ChannelID != config.C.ChannelID {
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
	c, ok := commands[args[0]]
	if !ok {
		// コマンドが見つからない
		_ = cmdCtx.ReplyBad(fmt.Sprintf("Unknown command: `%s`", args[0]))
		return
	}
	err = c.Execute(cmdCtx)
	if err != nil {
		cmdCtx.L().Error("failed to execute command", zap.Error(err))
	}
}
