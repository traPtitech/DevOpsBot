package main

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/dghubble/sling"
	"go.uber.org/zap"
	"net/http"
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
	// TenantID テナントID
	TenantID string `yaml:"tenantId"`
	// ServerID サーバーID
	ServerID string `yaml:"serverId"`
	// Description サーバー説明
	Description string `yaml:"description"`
	// Operators コマンド実行可能なユーザーの名前のデフォルト配列
	Operators []string `yaml:"operators"`
}

// Execute Commandインターフェース実装
func (s Server) Execute(ctx *Context) error {
	if len(ctx.Args) < 3 {
		return ctx.ReplyBad("Invalid Arguments")
	}
	// ctx.Args = server [server_name] restart [SOFT|HARD]
	args := ctx.Args[2:]

	switch args[0] {
	case "help":
		return ctx.Reply(s.MakeHelpMessage(), "")
	case "restart":
		if !s.CheckOperator(ctx.GetExecutor()) {
			return ctx.ReplyForbid()
		}

		if len(ctx.Args) < 4 {
			return ctx.ReplyBad("Invalid Arguments")
		}
		if !StringArrayContains([]string{"SOFT", "HARD"}, args[1]) {
			return ctx.ReplyBad(fmt.Sprintf("Unknown restart type: `%s`", args[1]))
		}

		_ = ctx.ReplyAccept()
		_ = ctx.ReplyRunning()

		req, err := sling.New().Base(config.ConohaApiOrigin).
			Post(fmt.Sprintf("v2/%s/servers/%s/action", s.TenantID, s.ServerID)).
			Body(bytes.NewBufferString(fmt.Sprintf("{ \"reboot\": { \"type\": \"%s\" } }", args[1]))).
			Set("Accept", "application/json").
			Set("X-Auth-Token", config.ConohaApiToken).
			Request()
		if err != nil {
			ctx.L().Error("failed to create request", zap.Error(err))
			return ctx.ReplyFailure("An internal error has occurred")
		}

		ctx.L().Info(fmt.Sprintf("post request to %s starts", req.URL.String()))
		resp, err := http.DefaultClient.Do(req)
		defer resp.Body.Close()

		ctx.L().Info("post request ends")
		if err != nil {
			ctx.L().Error("failed to post request", zap.Error(err))
			return ctx.ReplyFailure(fmt.Sprintf(":x: An error has occurred while executing command. %s", cite(ctx.P.Message.ID)))
		}

		ctx.L().Info(fmt.Sprintf("status code: %s", resp.Status))
		if resp.StatusCode == http.StatusAccepted {
			return ctx.ReplySuccess(fmt.Sprintf(":white_check_mark: Command execution was successful. %s", cite(ctx.P.Message.ID)))
		}
		return ctx.ReplyFailure(fmt.Sprintf(":x: Incorrect status code was received from ConoHa API.\nstatus code: `%s` %s", resp.Status, cite(ctx.P.Message.ID)))
	default:
		return ctx.ReplyBad(fmt.Sprintf("Unknown command: `%s`", args[0]))
	}
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

func (s *Server) GetOperators() []string {
	return s.Operators
}

// CheckOperator nameユーザーがこのコマンドを実行可能かどうか
func (s *Server) CheckOperator(name string) bool {
	return StringArrayContains(s.GetOperators(), name)
}
