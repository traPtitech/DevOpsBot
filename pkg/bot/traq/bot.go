package traq

import (
	"context"
	"fmt"
	"github.com/kballard/go-shellquote"
	"github.com/samber/lo"
	"github.com/traPtitech/DevOpsBot/pkg/config"
	"github.com/traPtitech/DevOpsBot/pkg/domain"
	"github.com/traPtitech/go-traq"
	traqwsbot "github.com/traPtitech/traq-ws-bot"
	"github.com/traPtitech/traq-ws-bot/payload"
	"go.uber.org/zap"
	"strings"
)

type traqBot struct {
	bot *traqwsbot.Bot
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

	stampNames, err := resolveStampNames(context.Background(), bot)
	if err != nil {
		return nil, fmt.Errorf("resolving stamp names: %w", err)
	}
	bot.OnMessageCreated(botMessageReceived(bot, logger, stampNames, rootCmd))

	return &traqBot{
		bot: bot,
	}, nil
}

func resolveStampNames(ctx context.Context, bot *traqwsbot.Bot) (*domain.StampNames, error) {
	stamps, _, err := bot.API().StampApi.GetStamps(ctx).Execute()
	if err != nil {
		return nil, err
	}

	idToName := lo.SliceToMap(stamps, func(s traq.StampWithThumbnail) (string, string) {
		return s.Id, s.Name
	})

	return &domain.StampNames{
		BadCommand: idToName[config.C.Stamps.BadCommand],
		Forbid:     idToName[config.C.Stamps.Forbid],
		Success:    idToName[config.C.Stamps.Success],
		Failure:    idToName[config.C.Stamps.Failure],
		Running:    idToName[config.C.Stamps.Running],
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
func botMessageReceived(
	bot *traqwsbot.Bot,
	logger *zap.Logger,
	stampNames *domain.StampNames,
	rootCmd domain.Command,
) func(p *payload.MessageCreated) {
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

			api:        bot.API(),
			logger:     logger,
			stampNames: stampNames,

			p:    p,
			args: nil,
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
