package traq

import (
	"context"
	"fmt"
	"github.com/kballard/go-shellquote"
	"github.com/traPtitech/DevOpsBot/pkg/config"
	"github.com/traPtitech/DevOpsBot/pkg/domain"
	traqwsbot "github.com/traPtitech/traq-ws-bot"
	"github.com/traPtitech/traq-ws-bot/payload"
	"go.uber.org/zap"
	"strings"
)

type traqBot struct {
	bot    *traqwsbot.Bot
	logger *zap.Logger
}

func NewBot(rootCmd domain.Command, logger *zap.Logger) (domain.Bot, error) {
	// Initialize bot
	bot, err := traqwsbot.NewBot(&traqwsbot.Options{
		AccessToken: config.C.Traq.Token,
		Origin:      config.C.Traq.Origin,
	})
	if err != nil {
		return nil, fmt.Errorf("creating bot: %w", err)
	}
	bot.OnMessageCreated(botMessageReceived(rootCmd))

	return &traqBot{
		bot:    bot,
		logger: logger,
	}, nil
}

func (b *traqBot) Start(_ context.Context) error {
	err := b.bot.Start()
	if err != nil {
		return fmt.Errorf("starting bot: %w", err)
	}
	return nil
}

// botMessageReceived BOTのMESSAGE_CREATEDイベントハンドラ
func botMessageReceived(rootCmd domain.Command) func(p *payload.MessageCreated) {
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
		err = rootCmd.Execute(ctx)
		if err != nil {
			ctx.L().Error("failed to execute command", zap.Error(err))
		}
	}
}
