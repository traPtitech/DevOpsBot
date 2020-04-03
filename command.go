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
	P    *MessageCreatedPayload
	Args []string
}

func (ctx *Context) GetExecutor() string {
	return ctx.P.Message.User.Name
}

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

func (ctx *Context) ReplyBad(message ...string) (err error) {
	return ctx.Reply(stringOrEmpty(message...), config.Stamps.BadCommand)
}

func (ctx *Context) ReplyForbid(message ...string) error {
	return ctx.Reply(stringOrEmpty(message...), config.Stamps.Forbid)
}

func (ctx *Context) ReplyAccept(message ...string) error {
	return ctx.Reply(stringOrEmpty(message...), config.Stamps.Accept)
}

func (ctx *Context) ReplySuccess(message ...string) error {
	return ctx.Reply(stringOrEmpty(message...), config.Stamps.Success)
}

func (ctx *Context) ReplyFailure(message ...string) error {
	return ctx.Reply(stringOrEmpty(message...), config.Stamps.Failure)
}

func (ctx *Context) L() *zap.Logger {
	return logger.With(
		zap.String("executor", ctx.GetExecutor()),
		zap.String("command", ctx.P.Message.PlainText),
		zap.Time("datetime", ctx.P.EventTime),
	)
}
