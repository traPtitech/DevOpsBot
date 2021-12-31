package main

import (
	"errors"
	"fmt"
	"github.com/dghubble/sling"
	"go.uber.org/zap"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

type Servers map[string]*Server

// UnmarshalYAML gopkg.in/yaml.v2.Unmarshaler 実装
func (ss *Servers) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var tmp map[string]*Server
	if err := unmarshal(&tmp); err != nil {
		return err
	}
	*ss = tmp
	for name, s := range *ss {
		// helpは予約済み
		if name == "help" {
			return errors.New("`help` cannot be used as server name")
		}
		s.Name = name
		s.Commands = map[string]ServerCommand{
			"restart": &ServerRestartCommand{s},
		}
	}
	return nil
}

// Execute Commandインターフェース実装
func (ss Servers) Execute(ctx *Context) error {
	if len(ctx.Args) < 2 {
		return ctx.ReplyBad("Invalid Arguments")
	}
	// ctx.Args = server [server_name] restart [SOFT|HARD]
	args := ctx.Args[1:]

	if args[0] == "help" {
		// サーバー一覧表示
		return ctx.Reply(ss.MakeHelpMessage(), "")
	}

	s, ok := ss[args[0]]
	if !ok {
		// サーバーが見つからない
		return ctx.ReplyBad(fmt.Sprintf("Unknown server: `%s`", args[0]))
	}
	return s.Execute(ctx)
}

// MakeHelpMessage server help用のメッセージを作成
func (ss Servers) MakeHelpMessage() string {
	var sb strings.Builder
	sb.WriteString("## server\n")
	sb.WriteString("### usage:\n")
	sb.WriteString("`server [server_name] restart [SOFT|HARD]`\n")
	sb.WriteString("### servers:\n")
	for name, s := range ss {
		if len(s.Description) > 0 {
			sb.WriteString(fmt.Sprintf("+ `%s` - %s\n", name, s.Description))
		} else {
			sb.WriteString(fmt.Sprintf("+ `%s`\n", name))
		}
	}
	return sb.String()
}

// Server サーバー
type Server struct {
	// Name サーバー名
	Name string `yaml:"-"`
	// ServerID サーバーID
	ServerID string `yaml:"serverId"`
	// Description サーバー説明
	Description string `yaml:"description"`
	// Operators コマンド実行可能なユーザーの名前のデフォルト配列
	Operators []string `yaml:"operators"`

	// Commands サーバーコマンド UnmarshalYAMLで追加
	Commands map[string]ServerCommand `yaml:"-"`
}

// Execute Commandインターフェース実装
func (s Server) Execute(ctx *Context) error {
	if len(ctx.Args) < 3 {
		return ctx.ReplyBad("Invalid Arguments")
	}
	// ctx.Args = server [server_name] restart [SOFT|HARD]
	args := ctx.Args[2:]

	if args[0] == "help" {
		return ctx.Reply(s.MakeHelpMessage(), "")
	}

	c, ok := s.Commands[args[0]]
	if !ok {
		// コマンドが見つからない
		return ctx.ReplyBad(fmt.Sprintf("Unknown command: `%s`", args[0]))
	}
	return c.Execute(ctx)
}

// MakeHelpMessage server [name] help用のメッセージを作成
func (s *Server) MakeHelpMessage() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("## server: %s\n", s.Name))
	sb.WriteString("### usage:\n")
	sb.WriteString(fmt.Sprintf("`server %s restart [SOFT|HARD]`\n", s.Name))
	sb.WriteString("### operators:\n")
	var quotedUsers []string
	for _, u := range s.GetOperators() {
		quotedUsers = append(quotedUsers, fmt.Sprintf("`%s`", u))
	}
	sb.WriteString(strings.Join(quotedUsers, ","))
	return sb.String()
}

type ServerCommand interface {
	Execute(ctx *Context) error
}

type ServerRestartCommand struct {
	server *Server
}

func (sc *ServerRestartCommand) Execute(ctx *Context) error {
	if !sc.server.CheckOperator(ctx.GetExecutor()) {
		return ctx.ReplyForbid()
	}

	if len(ctx.Args) < 4 {
		return ctx.ReplyBad("Invalid Arguments")
	}
	// ctx.Args = server [server_name] restart [SOFT|HARD]
	args := ctx.Args[3:]

	if !StringArrayContains([]string{"SOFT", "HARD"}, args[0]) {
		return ctx.ReplyBad(fmt.Sprintf("Unknown restart type: `%s`", args[0]))
	}

	_ = ctx.ReplyAccept()
	_ = ctx.ReplyRunning()

	req, err := sling.New().Base(config.ConohaApiOrigin).
		Post(fmt.Sprintf("v2/%s/servers/%s/action", config.TenantID, sc.server.ServerID)).
		BodyJSON(Map{"reboot": Map{"type": args[0]}}).
		Set("Content-Type", "application/json").
		Set("X-Auth-Token", config.ConohaApiToken).
		Request()
	if err != nil {
		ctx.L().Error("failed to create restart request", zap.String("URL", req.URL.String()), zap.Error(err))
		return ctx.ReplyFailure("An internal error has occurred")
	}

	ctx.L().Info("post restart request starts", zap.String("URL", req.URL.String()))
	resp, err := http.DefaultClient.Do(req)

	ctx.L().Info("post restart request ends")
	if err != nil {
		ctx.L().Error("failed to post restart request", zap.Error(err))
		return ctx.ReplyFailure(fmt.Sprintf(":x: An error has occurred while executing command. %s", cite(ctx.P.Message.ID)))
	}
	defer resp.Body.Close()

	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		ctx.L().Error("failed to read resp.Body", zap.Error(err))
		return ctx.ReplyFailure("An internal error has occurred")
	}

	logFile, err := sc.openLogFile(ctx)
	if err != nil {
		ctx.L().Error("failed to open log file", zap.Error(err))
		return ctx.ReplyFailure("An internal error has occurred")
	}
	defer logFile.Close()

	_, err = logFile.WriteString(fmt.Sprintf("Request\n- URL: %s\n- RestartType: %s\nResponse\n- Header: %+v\n- Body: %s\n- Status: %s (Expect: 202)\n", req.URL.String(), args[0], resp.Header, string(respBody), resp.Status))
	if err != nil {
		ctx.L().Error("failed to write log file", zap.Error(err))
		return ctx.ReplyFailure("An internal error has occurred")
	}

	ctx.L().Info(fmt.Sprintf("status code: %s", resp.Status))
	if resp.StatusCode == http.StatusAccepted {
		return ctx.ReplySuccess(fmt.Sprintf(":white_check_mark: Command execution was successful.\nPlease check the execution log. `exec-log %s %s %d` %s", sc.server.Name, "restart", ctx.P.EventTime.Unix(), cite(ctx.P.Message.ID)))
	}
	return ctx.ReplyFailure(fmt.Sprintf(":x: Incorrect status code was received from ConoHa API. Status code: `%s`\nPlease check the execution log. `exec-log %s %s %d` %s", resp.Status, sc.server.Name, "restart", ctx.P.EventTime.Unix(), cite(ctx.P.Message.ID)))
}

func (sc *ServerRestartCommand) openLogFile(ctx *Context) (*os.File, error) {
	logFilePath := filepath.Join(config.LogsDir, sc.getLogFileName(ctx))
	logFile, err := os.OpenFile(logFilePath, os.O_RDWR|os.O_CREATE, 0600)
	if err != nil {
		ctx.L().Error("failed to open log file", zap.String("path", logFilePath), zap.Error(err))
		return nil, err
	}
	return logFile, nil
}

func (sc *ServerRestartCommand) getLogFileName(ctx *Context) string {
	return sc.getLogFileNameByUnixTime(ctx.P.EventTime.Unix())
}

func (sc *ServerRestartCommand) getLogFileNameByUnixTime(unix int64) string {
	return fmt.Sprintf("exec-%s-%s-%d", sc.server.Name, "restart", unix)
}

func (s *Server) GetOperators() []string {
	return s.Operators
}

// CheckOperator nameユーザーがこのコマンドを実行可能かどうか
func (s *Server) CheckOperator(name string) bool {
	return StringArrayContains(s.GetOperators(), name)
}
