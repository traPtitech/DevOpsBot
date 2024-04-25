package bot

import (
	"context"

	"github.com/traPtitech/go-traq"

	"github.com/traPtitech/DevOpsBot/pkg/utils"
)

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
