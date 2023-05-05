package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"

	"go.uber.org/zap"
)

// ExecLogCommand `exec-log [service|server] [name] [command] [unix]`
type ExecLogCommand struct {
	svr *ServersCommand
}

func (ec *ExecLogCommand) Execute(ctx *Context) error {
	// ctx.Args = exec-log server [name] [command] [unix]
	if len(ctx.Args) != 5 {
		return ctx.ReplyBad("Invalid Arguments")
	}
	args := ctx.Args[1:]
	unix, err := strconv.ParseInt(args[3], 10, 64)
	if err != nil {
		return ctx.ReplyBad("Invalid Arguments")
	}

	var logName string
	var logsDir string

	switch args[0] {
	case "server":
		s, ok := ec.svr.instances[args[1]]
		if !ok {
			// サーバーが見つからない
			return ctx.ReplyBad(fmt.Sprintf("Unknown server: `%s`", args[1]))
		}
		c, ok := s.Commands[args[2]]
		if !ok {
			// コマンドが見つからない
			return ctx.ReplyBad(fmt.Sprintf("Unknown command: `%s`", args[2]))
		}

		// オペレーター確認
		if !s.CheckOperator(ctx.GetExecutor()) {
			return ctx.ReplyForbid()
		}

		logsDir = config.Commands.Servers.LogsDir
		logName = c.getLogFileNameByUnixTime(unix)
	default:
		return ctx.ReplyBad("Invalid Arguments")
	}

	logFilePath := filepath.Join(logsDir, logName)

	if !fileExists(logFilePath) {
		return ctx.ReplyBad("Log not found")
	}

	f, err := os.Open(logFilePath)
	if err != nil {
		ctx.L().Error("opening log file", zap.Error(err))
		return ctx.ReplyFailure("Error opening log file")
	}
	b, err := io.ReadAll(f)
	if err != nil {
		ctx.L().Error("reading log file", zap.Error(err))
		return ctx.ReplyFailure("Error reading log file")
	}

	_ = ctx.ReplyAccept()
	return ctx.ReplyViaDM("```\n" + safeConvertString(b) + "\n```")
}
