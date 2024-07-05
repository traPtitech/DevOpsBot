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

func (ctx *slackContext) MessageLimit() int {
	return 2000
}

func (ctx *slackContext) StampNames() *domain.StampNames {
	return &domain.StampNames{
		BadCommand: config.C.Stamps.BadCommand,
		Forbid:     config.C.Stamps.Forbid,
		Success:    config.C.Stamps.Success,
		Failure:    config.C.Stamps.Failure,
		Running:    config.C.Stamps.Running,
	}
}

func (ctx *slackContext) sendSlackMessage(channelID string, lines []string, color string) error {
	api := ctx.api
	return utils.WithRetry(ctx, 10, func(ctx context.Context) error {
		var options []slack.MsgOption
		options = append(options, slack.MsgOptionText(lines[0], false))
		if len(lines) >= 2 {
			options = append(options, slack.MsgOptionAttachments(
				slack.Attachment{
					Color: color,
					Fields: []slack.AttachmentField{
						{
							Title: "",
							Value: strings.Join(lines[1:], "\n"),
							Short: false,
						},
					},
				},
			))
		}
		_, _, err := api.PostMessage(channelID, options...)
		return err
	})
}

func (ctx *slackContext) pushSlackReaction(message slack.ItemRef, stampID string) error {
	api := ctx.api
	return utils.WithRetry(ctx, 10, func(ctx context.Context) error {
		return api.AddReaction(stampID, message)
	})
}

func (ctx *slackContext) reply(color string, message ...string) error {
	return ctx.sendSlackMessage(ctx.message.Channel, message, color)
}

func (ctx *slackContext) replyWithStamp(stamp string, color string, message ...string) error {
	err := ctx.pushSlackReaction(ctx.message, stamp)
	if err != nil {
		return err
	}
	if len(message) > 0 {
		err = ctx.reply(color, message...)
		if err != nil {
			return err
		}
	}
	return nil
}

func (ctx *slackContext) ReplyBad(message ...string) error {
	return ctx.replyWithStamp(config.C.Stamps.BadCommand, config.C.Slack.Colors.BadCommand, message...)
}

func (ctx *slackContext) ReplyForbid(message ...string) error {
	return ctx.replyWithStamp(config.C.Stamps.Forbid, config.C.Slack.Colors.Forbid, message...)
}

func (ctx *slackContext) ReplySuccess(message ...string) error {
	return ctx.replyWithStamp(config.C.Stamps.Success, config.C.Slack.Colors.Success, message...)
}

func (ctx *slackContext) ReplyFailure(message ...string) error {
	return ctx.replyWithStamp(config.C.Stamps.Failure, config.C.Slack.Colors.Failure, message...)
}

func (ctx *slackContext) ReplyRunning(message ...string) error {
	return ctx.replyWithStamp(config.C.Stamps.Running, config.C.Slack.Colors.Running, message...)
}
