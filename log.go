package main

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/patrickmn/go-cache"
	"net/http"
	"path/filepath"
	"strconv"
)

func GetLog(c *gin.Context) {
	key := c.Param("key")
	if len(key) == 0 {
		c.Status(http.StatusNotFound)
		return
	}

	logName, ok := logAccessUrls.Get(key)
	if !ok {
		c.Status(http.StatusNotFound)
		return
	}

	c.FileAttachment(filepath.Join(config.LogsDir, logName.(string)), logName.(string))
	return
}

type ExecLogCommand struct {
}

func (ec *ExecLogCommand) Execute(ctx *Context) error {
	// ctx.Args = exec-log [service] [command] [unix]
	if len(ctx.Args) != 4 {
		return ctx.ReplyBad("Invalid Arguments")
	}
	unix, err := strconv.ParseInt(ctx.Args[3], 10, 64)
	if err != nil {
		return ctx.ReplyBad("Invalid Arguments")
	}

	s, ok := config.Services[ctx.Args[1]]
	if !ok {
		// サービスが見つからない
		return ctx.ReplyBad(fmt.Sprintf("Unknown service: `%s`", ctx.Args[1]))
	}
	c, ok := s.Commands[ctx.Args[2]]
	if !ok {
		// コマンドが見つからない
		return ctx.ReplyBad(fmt.Sprintf("Unknown command: `%s`", ctx.Args[2]))
	}

	// オペレーター確認
	if !c.CheckOperator(ctx.GetExecutor()) {
		return ctx.ReplyForbid()
	}

	logName := c.getLogFileNameByUnixTime(unix)
	logFilePath := filepath.Join(config.LogsDir, logName)

	if !fileExists(logFilePath) {
		return ctx.ReplyBad("Log not found")
	}

	key := RandAlphabetAndNumberString(30)
	logAccessUrls.Set(key, logName, cache.DefaultExpiration)
	_ = ctx.ReplyAccept()
	return ctx.ReplyViaDM(fmt.Sprintf("%s/log/%s\nThis URL is valid for 3 minutes.", config.DevOpsBotOrigin, key))
}
