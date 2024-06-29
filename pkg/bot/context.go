package bot

import (
	"context"
	"go.uber.org/zap"
)

// Context コマンド実行コンテキスト
type Context interface {
	context.Context

	// Executor コマンドを実行した人 (メッセージの投稿者のID) を返します
	Executor() string
	// Args 投稿メッセージを空白区切りで分けたもの
	Args() []string
	// ShiftArgs pops the first argument and creates a new command context.
	ShiftArgs() Context

	// L returns logger.
	L() *zap.Logger

	// ReplyBad コマンドメッセージにBadスタンプをつけて返信します
	ReplyBad(message ...string) error
	// ReplyForbid コマンドメッセージにForbidスタンプをつけて返信します
	ReplyForbid(message ...string) error
	// ReplyAccept コマンドメッセージにAcceptスタンプをつけて返信します
	ReplyAccept(message ...string) error
	// ReplySuccess コマンドメッセージにSuccessスタンプをつけて返信します
	ReplySuccess(message ...string) error
	// ReplyFailure コマンドメッセージにFailureスタンプをつけて返信します
	ReplyFailure(message ...string) error
	// ReplyRunning コマンドメッセージにRunningスタンプをつけて返信します
	ReplyRunning(message ...string) error
}
