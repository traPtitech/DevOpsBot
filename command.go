package main

import (
	"go.uber.org/zap"
)

var commands = map[string]Command{}

// Command コマンドインターフェース
type Command interface {
	// Execute コマンドを実行します
	Execute(ctx *Context) error
}

// Context コマンド実行コンテキスト
type Context struct {
	// P BOTが受信したMESSAGE_CREATEDイベントの生のペイロード
	P *MessageCreatedPayload
	// Args 投稿メッセージを空白区切りで分けたもの
	Args []string
}

// GetExecutor コマンドを実行した人(traQメッセージの投稿者のtraQ IDを返します
func (ctx *Context) GetExecutor() string {
	return ctx.P.Message.User.Name
}

// ReplyViaDM コマンドメッセージに返信します
func (ctx *Context) Reply(message, stamp string) (err error) {
	if len(message) > 0 {
		err = SendTRAQMessage(ctx.P.Message.ChannelID, message)
		if err != nil {
			return
		}
	}
	if len(stamp) > 0 {
		err = PushTRAQStamp(ctx.P.Message.ID, stamp)
		if err != nil {
			return
		}
	}
	return
}

// ReplyViaDM コマンド実行者にDMで返信します
func (ctx *Context) ReplyViaDM(message string) error {
	return SendTRAQDirectMessage(ctx.P.Message.User.ID, message)
}

// ReplyBad コマンドメッセージにBadスタンプをつけて返信します
func (ctx *Context) ReplyBad(message ...string) (err error) {
	return ctx.Reply(stringOrEmpty(message...), config.Stamps.BadCommand)
}

// ReplyForbid コマンドメッセージにForbidスタンプをつけて返信します
func (ctx *Context) ReplyForbid(message ...string) error {
	return ctx.Reply(stringOrEmpty(message...), config.Stamps.Forbid)
}

// ReplyAccept コマンドメッセージにAcceptスタンプをつけて返信します
func (ctx *Context) ReplyAccept(message ...string) error {
	return ctx.Reply(stringOrEmpty(message...), config.Stamps.Accept)
}

// ReplySuccess コマンドメッセージにSuccessスタンプをつけて返信します
func (ctx *Context) ReplySuccess(message ...string) error {
	return ctx.Reply(stringOrEmpty(message...), config.Stamps.Success)
}

// ReplyFailure コマンドメッセージにFailureスタンプをつけて返信します
func (ctx *Context) ReplyFailure(message ...string) error {
	return ctx.Reply(stringOrEmpty(message...), config.Stamps.Failure)
}

// ReplyRunning コマンドメッセージにRunningスタンプをつけて返信します
func (ctx *Context) ReplyRunning(message ...string) error {
	return ctx.Reply(stringOrEmpty(message...), config.Stamps.Running)
}

func (ctx *Context) L() *zap.Logger {
	return logger.With(
		zap.String("executor", ctx.GetExecutor()),
		zap.String("command", ctx.P.Message.PlainText),
		zap.Time("datetime", ctx.P.EventTime),
	)
}
