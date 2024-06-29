package slack

import (
	"context"
	"github.com/slack-go/slack"
	"github.com/traPtitech/DevOpsBot/pkg/config"
	"github.com/traPtitech/DevOpsBot/pkg/domain"
	"github.com/traPtitech/DevOpsBot/pkg/utils"
	"go.uber.org/zap"
	"strings"
)

type slackContext struct {
	context.Context
	api    *slack.Client
	logger *zap.Logger

	message    slack.ItemRef
	executorID string
	args       []string
}

func (ctx *slackContext) Executor() string {
	return ctx.executorID
}

func (ctx *slackContext) Args() []string {
	return ctx.args
}

func (ctx *slackContext) ShiftArgs() domain.Context {
	newCtx := *ctx
	newCtx.args = newCtx.args[1:]
	return &newCtx
}

func (ctx *slackContext) L() *zap.Logger {
	return ctx.logger.With(
		zap.String("executor", ctx.Executor()),
		//zap.String("command", s.p.Message.PlainText),
		//zap.Time("datetime", s.p.EventTime),
	)
}

func (ctx *slackContext) sendSlackMessage(channelID string, text string) error {
	api := ctx.api
	return utils.WithRetry(ctx, 10, func(ctx context.Context) error {
		_, _, err := api.PostMessage(channelID, slack.MsgOptionBlocks(
			slack.NewSectionBlock(
				slack.NewTextBlockObject(
					slack.MarkdownType,
					text,
					false,
					false,
				),
				nil,
				nil,
			),
		))
		return err
	})
}

func (ctx *slackContext) pushSlackReaction(message slack.ItemRef, stampID string) error {
	api := ctx.api
	return utils.WithRetry(ctx, 10, func(ctx context.Context) error {
		return api.AddReaction(stampID, message)
	})
}

func (ctx *slackContext) reply(message ...string) error {
	return ctx.sendSlackMessage(ctx.message.Channel, strings.Join(message, "\n"))
}

func (ctx *slackContext) replyWithStamp(stamp string, message ...string) error {
	err := ctx.pushSlackReaction(ctx.message, stamp)
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

func (ctx *slackContext) ReplyBad(message ...string) error {
	return ctx.replyWithStamp(config.C.Stamps.BadCommand, message...)
}

func (ctx *slackContext) ReplyForbid(message ...string) error {
	return ctx.replyWithStamp(config.C.Stamps.Forbid, message...)
}

func (ctx *slackContext) ReplySuccess(message ...string) error {
	return ctx.replyWithStamp(config.C.Stamps.Success, message...)
}

func (ctx *slackContext) ReplyFailure(message ...string) error {
	return ctx.replyWithStamp(config.C.Stamps.Failure, message...)
}

func (ctx *slackContext) ReplyRunning(message ...string) error {
	return ctx.replyWithStamp(config.C.Stamps.Running, message...)
}
