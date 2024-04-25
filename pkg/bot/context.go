package bot

import (
	"context"
	"go.uber.org/zap"
	"strings"

	"github.com/traPtitech/traq-ws-bot/payload"

	"github.com/traPtitech/DevOpsBot/pkg/config"
)

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
