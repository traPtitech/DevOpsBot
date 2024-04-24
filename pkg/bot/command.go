package bot

import (
	"context"
	"strings"

	"github.com/traPtitech/go-traq"
	"github.com/traPtitech/traq-ws-bot/payload"
	"go.uber.org/zap"

	"github.com/traPtitech/DevOpsBot/pkg/config"
	"github.com/traPtitech/DevOpsBot/pkg/utils"
)

var commands = map[string]command{}

// command コマンドインターフェース
type command interface {
	// Execute コマンドを実行します
	Execute(ctx *Context) error
}

// sendTRAQMessage traQにメッセージ送信
func sendTRAQMessage(ctx context.Context, channelID string, text string) error {
	return utils.WithRetry(ctx, 10, func(ctx context.Context) error {
		_, _, err := bot.API().
			ChannelApi.
			PostMessage(ctx, channelID).
			PostMessageRequest(traq.PostMessageRequest{Content: text}).
			Execute()
		return err
	})
}

// sendTRAQDirectMessage traQにダイレクトメッセージ送信
func sendTRAQDirectMessage(ctx context.Context, userID string, text string) error {
	return utils.WithRetry(ctx, 10, func(ctx context.Context) error {
		_, _, err := bot.API().
			UserApi.
			PostDirectMessage(ctx, userID).
			PostMessageRequest(traq.PostMessageRequest{Content: text}).
			Execute()
		return err
	})
}

// pushTRAQStamp traQのメッセージにスタンプを押す
func pushTRAQStamp(ctx context.Context, messageID, stampID string) error {
	return utils.WithRetry(ctx, 10, func(ctx context.Context) error {
		_, err := bot.API().
			MessageApi.
			AddMessageStamp(ctx, messageID, stampID).
			PostMessageStampRequest(traq.PostMessageStampRequest{Count: 1}).
			Execute()
		return err
	})
}

// Context コマンド実行コンテキスト
type Context struct {
	context.Context
	// P BOTが受信したMESSAGE_CREATEDイベントの生のペイロード
	P *payload.MessageCreated
	// Args 投稿メッセージを空白区切りで分けたもの
	Args []string
}

// GetExecutor コマンドを実行した人(traQメッセージの投稿者のtraQ IDを返します
func (ctx *Context) GetExecutor() string {
	return ctx.P.Message.User.Name
}

// Reply コマンドメッセージに返信します
func (ctx *Context) Reply(message ...string) error {
	return sendTRAQMessage(ctx, ctx.P.Message.ChannelID, strings.Join(message, "\n"))
}

func (ctx *Context) ReplyWithStamp(stamp string, message ...string) error {
	err := pushTRAQStamp(ctx, ctx.P.Message.ID, stamp)
	if err != nil {
		return err
	}
	if len(message) > 0 {
		err = ctx.Reply(message...)
		if err != nil {
			return err
		}
	}
	return nil
}

// ReplyViaDM コマンド実行者にDMで返信します
func (ctx *Context) ReplyViaDM(message ...string) error {
	return sendTRAQDirectMessage(ctx, ctx.P.Message.User.ID, strings.Join(message, "\n"))
}

// ReplyBad コマンドメッセージにBadスタンプをつけて返信します
func (ctx *Context) ReplyBad(message ...string) (err error) {
	return ctx.ReplyWithStamp(config.C.Stamps.BadCommand, message...)
}

// ReplyForbid コマンドメッセージにForbidスタンプをつけて返信します
func (ctx *Context) ReplyForbid(message ...string) error {
	return ctx.ReplyWithStamp(config.C.Stamps.Forbid, message...)
}

// ReplyAccept コマンドメッセージにAcceptスタンプをつけて返信します
func (ctx *Context) ReplyAccept(message ...string) error {
	return ctx.ReplyWithStamp(config.C.Stamps.Accept, message...)
}

// ReplySuccess コマンドメッセージにSuccessスタンプをつけて返信します
func (ctx *Context) ReplySuccess(message ...string) error {
	return ctx.ReplyWithStamp(config.C.Stamps.Success, message...)
}

// ReplyFailure コマンドメッセージにFailureスタンプをつけて返信します
func (ctx *Context) ReplyFailure(message ...string) error {
	return ctx.ReplyWithStamp(config.C.Stamps.Failure, message...)
}

// ReplyRunning コマンドメッセージにRunningスタンプをつけて返信します
func (ctx *Context) ReplyRunning(message ...string) error {
	return ctx.ReplyWithStamp(config.C.Stamps.Running, message...)
}

func (ctx *Context) L() *zap.Logger {
	return logger.With(
		zap.String("executor", ctx.GetExecutor()),
		zap.String("command", ctx.P.Message.PlainText),
		zap.Time("datetime", ctx.P.EventTime),
	)
}
