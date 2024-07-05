package traq

import (
	"context"
	"github.com/traPtitech/DevOpsBot/pkg/config"
	"github.com/traPtitech/DevOpsBot/pkg/domain"
	"github.com/traPtitech/traq-ws-bot/payload"
	"go.uber.org/zap"
	"strings"

	"github.com/traPtitech/go-traq"

	"github.com/traPtitech/DevOpsBot/pkg/utils"
)

type traqContext struct {
	context.Context

	api        *traq.APIClient
	logger     *zap.Logger
	stampNames *domain.StampNames

	// p BOTが受信したMESSAGE_CREATEDイベントの生のペイロード
	p    *payload.MessageCreated
	args []string
}

func (ctx *traqContext) Executor() string {
	return ctx.p.Message.User.Name
}

func (ctx *traqContext) Args() []string {
	return ctx.args
}

func (ctx *traqContext) ShiftArgs() domain.Context {
	newCtx := *ctx
	newCtx.args = newCtx.args[1:]
	return &newCtx
}

func (ctx *traqContext) L() *zap.Logger {
	return ctx.logger.With(
		zap.String("executor", ctx.Executor()),
		zap.String("command", ctx.p.Message.PlainText),
		zap.Time("datetime", ctx.p.EventTime),
	)
}

func (ctx *traqContext) MessageLimit() int {
	return 9900
}

func (ctx *traqContext) StampNames() *domain.StampNames {
	return ctx.stampNames
}

// sendTRAQMessage traQにメッセージ送信
func (ctx *traqContext) sendTRAQMessage(channelID string, text string) error {
	api := ctx.api
	return utils.WithRetry(ctx, 10, func(ctx context.Context) error {
		_, _, err := api.
			ChannelApi.
			PostMessage(ctx, channelID).
			PostMessageRequest(traq.PostMessageRequest{Content: text}).
			Execute()
		return err
	})
}

// pushTRAQStamp traQのメッセージにスタンプを押す
func (ctx *traqContext) pushTRAQStamp(messageID, stampID string) error {
	api := ctx.api
	return utils.WithRetry(ctx, 10, func(ctx context.Context) error {
		_, err := api.
			MessageApi.
			AddMessageStamp(ctx, messageID, stampID).
			PostMessageStampRequest(traq.PostMessageStampRequest{Count: 1}).
			Execute()
		return err
	})
}

func (ctx *traqContext) reply(message ...string) error {
	return ctx.sendTRAQMessage(ctx.p.Message.ChannelID, strings.Join(message, "\n"))
}

func (ctx *traqContext) replyWithStamp(stamp string, message ...string) error {
	err := ctx.pushTRAQStamp(ctx.p.Message.ID, stamp)
	if err != nil {
		return err
	}
	if len(message) > 0 {
		err = ctx.reply(message...)
		if err != nil {
			return err
		}
	}
	return nil
}

func (ctx *traqContext) ReplyBad(message ...string) error {
	return ctx.replyWithStamp(config.C.Stamps.BadCommand, message...)
}

func (ctx *traqContext) ReplyForbid(message ...string) error {
	return ctx.replyWithStamp(config.C.Stamps.Forbid, message...)
}

func (ctx *traqContext) ReplySuccess(message ...string) error {
	return ctx.replyWithStamp(config.C.Stamps.Success, message...)
}

func (ctx *traqContext) ReplyFailure(message ...string) error {
	return ctx.replyWithStamp(config.C.Stamps.Failure, message...)
}

func (ctx *traqContext) ReplyRunning(message ...string) error {
	return ctx.replyWithStamp(config.C.Stamps.Running, message...)
}
