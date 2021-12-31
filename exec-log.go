package main

import (
	"fmt"
	"github.com/go-chi/chi/v5"
	"github.com/patrickmn/go-cache"
	"net/http"
	"path/filepath"
	"strconv"
)

// ExecLogCommand `exec-log [service|server] [command] [unix]`
type ExecLogCommand struct {
}

func (ec *ExecLogCommand) Execute(ctx *Context) error {
	// ctx.Args = exec-log [service|server] [command] [unix]
	if len(ctx.Args) != 4 {
		return ctx.ReplyBad("Invalid Arguments")
	}
	unix, err := strconv.ParseInt(ctx.Args[3], 10, 64)
	if err != nil {
		return ctx.ReplyBad("Invalid Arguments")
	}

	var logName string

	if service, ok := config.Services[ctx.Args[1]]; ok {
		c, ok := service.Commands[ctx.Args[2]]
		if !ok {
			// コマンドが見つからない
			return ctx.ReplyBad(fmt.Sprintf("Unknown command: `%s`", ctx.Args[2]))
		}

		// オペレーター確認
		if !c.CheckOperator(ctx.GetExecutor()) {
			return ctx.ReplyForbid()
		}

		logName = c.getLogFileNameByUnixTime(unix)
	} else if server, ok := config.Servers[ctx.Args[1]]; ok {
		c, ok := server.Commands[ctx.Args[2]]
		if !ok {
			// コマンドが見つからない
			return ctx.ReplyBad(fmt.Sprintf("Unknown command: `%s`", ctx.Args[2]))
		}

		// オペレーター確認
		if !server.CheckOperator(ctx.GetExecutor()) {
			return ctx.ReplyForbid()
		}

		logName = c.getLogFileNameByUnixTime(unix)
	} else {
		// サービスまたはサーバーが見つからない
		return ctx.ReplyBad(fmt.Sprintf("Unknown service: `%s`", ctx.Args[1]))
	}

	logFilePath := filepath.Join(config.LogsDir, logName)

	if !fileExists(logFilePath) {
		return ctx.ReplyBad("Log not found")
	}

	key := RandAlphaNumericString(30)
	logAccessUrls.Set(key, logName, cache.DefaultExpiration)
	_ = ctx.ReplyAccept()

	fileURL := fmt.Sprintf("%s/log/%s", config.DevOpsBotOrigin, key)
	return ctx.ReplyViaDM(fmt.Sprintf("[View](%s) [Download](%s?dl=1)\n\nThese URL is valid for 3 minutes.", fileURL, fileURL))
}

func GetLog(w http.ResponseWriter, r *http.Request) {
	key := chi.URLParam(r, "key")
	if len(key) == 0 {
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}
	shouldDownloadFile := r.URL.Query().Get("dl") == "1"

	logName, ok := logAccessUrls.Get(key)
	if !ok {
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}

	if shouldDownloadFile {
		w.Header().Set("content-disposition", fmt.Sprintf("attachment; filename=\"%s\"", logName.(string)))
	}

	http.ServeFile(w, r, filepath.Join(config.LogsDir, logName.(string)))
	return
}
