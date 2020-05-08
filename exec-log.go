package main

import (
	"fmt"
	"github.com/go-chi/chi"
	"github.com/patrickmn/go-cache"
	"net/http"
	"path/filepath"
	"strconv"
)

// ExecLogCommand `exec-log [service] [command] [unix]`
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

	key := RandAlphaNumericString(30)
	logAccessUrls.Set(key, logName, cache.DefaultExpiration)
	_ = ctx.ReplyAccept()
	return ctx.ReplyViaDM(fmt.Sprintf("%s/log/%s\nThis URL is valid for 3 minutes.", config.DevOpsBotOrigin, key))
}

func GetLog(w http.ResponseWriter, r *http.Request) {
	key := chi.URLParam(r, "key")
	if len(key) == 0 {
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}

	logName, ok := logAccessUrls.Get(key)
	if !ok {
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}

	w.Header().Set("content-disposition", fmt.Sprintf("attachment; filename=\"%s\"", logName.(string)))
	http.ServeFile(w, r, filepath.Join(config.LogsDir, logName.(string)))
	return
}
