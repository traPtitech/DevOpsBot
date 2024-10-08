package domain

import (
	"context"
	"go.uber.org/zap"
)

type Bot interface {
	// Start connects the bot. Must block on success.
	Start(ctx context.Context) error
}

type StampNames struct {
	BadCommand string
	Forbid     string
	Success    string
	Failure    string
	Running    string
}

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
	// MessageLimit returns reply character limit.
	MessageLimit() int
	// StampNames returns list of stamp names which can be used in raw text.
	StampNames() *StampNames

	// ReplyBad コマンドメッセージにBadスタンプをつけて返信します
	ReplyBad(message ...string) error
	// ReplyForbid コマンドメッセージにForbidスタンプをつけて返信します
	ReplyForbid(message ...string) error
	// ReplySuccess コマンドメッセージにSuccessスタンプをつけて返信します
	ReplySuccess(message ...string) error
	// ReplyFailure コマンドメッセージにFailureスタンプをつけて返信します
	ReplyFailure(message ...string) error
	// ReplyRunning コマンドメッセージにRunningスタンプをつけて返信します
	ReplyRunning(message ...string) error
}

// Command コマンドインターフェース
type Command interface {
	Execute(ctx Context) error
	HasSubcommands() bool
	GetSubcommand(verb string) (Command, bool)
	HelpMessage(indent int, formatSub bool) []string
}
