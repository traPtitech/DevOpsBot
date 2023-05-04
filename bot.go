package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/kballard/go-shellquote"
	"github.com/samber/lo"
	"github.com/traPtitech/go-traq"
	"github.com/traPtitech/traq-ws-bot/payload"
	"go.uber.org/zap"
)

type Map map[string]interface{}

// BotMessageReceived BOTのMESSAGE_CREATEDイベントハンドラ
func BotMessageReceived(p *payload.MessageCreated) {
	ctx := context.Background()

	if p.Message.ChannelID != config.DevOpsChannelID {
		return // DevOpsチャンネル以外からのメッセージは無視
	}

	args, err := shellquote.Split(p.Message.PlainText)
	if err != nil {
		_ = SendTRAQMessage(ctx, p.Message.ChannelID, fmt.Sprintf("invalid syntax error\n%s", cite(p.Message.ID)))
		_ = PushTRAQStamp(ctx, p.Message.ID, config.Stamps.BadCommand)
		return
	}
	_, argStart, ok := lo.FindIndexOf(args, func(arg string) bool { return strings.HasPrefix(arg, config.Prefix) })
	if !ok {
		return
	}
	args = args[argStart:]
	args[0] = strings.TrimPrefix(args[0], config.Prefix)

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

// SendTRAQMessage traQにメッセージ送信
func SendTRAQMessage(ctx context.Context, channelID string, text string) error {
	_, _, err := bot.API().
		ChannelApi.
		PostMessage(ctx, channelID).
		PostMessageRequest(traq.PostMessageRequest{Content: text}).
		Execute()
	return err
}

// SendTRAQDirectMessage traQにダイレクトメッセージ送信
func SendTRAQDirectMessage(ctx context.Context, userID string, text string) error {
	_, _, err := bot.API().
		UserApi.
		PostDirectMessage(ctx, userID).
		PostMessageRequest(traq.PostMessageRequest{Content: text}).
		Execute()
	return err
}

// PushTRAQStamp traQのメッセージにスタンプを押す
func PushTRAQStamp(ctx context.Context, messageID, stampID string) error {
	_, err := bot.API().
		MessageApi.
		AddMessageStamp(ctx, messageID, stampID).
		PostMessageStampRequest(traq.PostMessageStampRequest{Count: 1}).
		Execute()
	return err
}

// cite traQのメッセージ引用形式を作る
func cite(messageId string) string {
	return fmt.Sprintf(`%smessages/%s`, config.TraqOrigin, messageId)
}
