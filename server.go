package main

import (
	"bytes"
	"errors"
	"fmt"
	"go.uber.org/zap"
	"net/http"
	"strings"
	"time"
)

type Servers map[string]*Server

// UnmarshalYAML gopkg.in/yaml.v2.Unmarshaler 実装
func (ss *Servers) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var tmp map[string]*Server
	if err := unmarshal(&tmp); err != nil {
		return err
	}
	*ss = tmp
	for name := range *ss {
		// helpは予約済み
		if name == "help" {
			return errors.New("`help` cannot be used as server name")
		}
	}
	return nil
}

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

type Server struct {
	// Name サーバー名
	Name string `yaml:"-"`
	// Description サーバー説明
	Description string `yaml:"description"`
	// Operators コマンド実行可能なユーザーの名前のデフォルト配列
	Operators []string `yaml:"operators"`
}

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
		if StringArrayContains([]string{"SOFT", "HARD"}, args[1]) {
			return ctx.ReplyBad(fmt.Sprintf("Unknown restart type: %s", args[1]))
		}

		_ = ctx.ReplyAccept()
		_ = ctx.ReplyRunning()

		// 環境変数の取得は main.go で最初に行う方が良いか
		apiURL, err := getEnvOrError("CONOHA_API_URL")
		if err != nil {
			ctx.L().Error("failed to get env CONOHA_API_URL", zap.Error(err))
			return ctx.ReplyBad("An internal error has occurred")
		}

		apiToken, err := getEnvOrError("CONOHA_API_TOKEN")
		if err != nil {
			ctx.L().Error("failed to get env CONOHA_API_TOKEN", zap.Error(err))
			return ctx.ReplyBad("An internal error has occurred")
		}

		// TODO: s.Name に応じて tenantID と serverID を環境変数から取得
		tenantID := "1"
		serverID := "1"

		url := fmt.Sprintf(fmt.Sprintf("%s/v2/%s/servers/%s/action", apiURL, tenantID, serverID))
		reqBody := bytes.NewBufferString(fmt.Sprintf("{ \"reboot\": { \"type\": \"%s\" } }", args[1]))

		ctx.L().Info(fmt.Sprintf("post request to %s starts", url))
		req, _ := http.NewRequest("POST", url, reqBody)
		req.Header.Set("Accept", "application/json")
		req.Header.Set("X-Auth-Token", apiToken)

		client := &http.Client{
			Timeout: 10 * time.Second,
		}
		resp, err := client.Do(req)
		defer resp.Body.Close()

		ctx.L().Info("post request ends")
		ctx.L().Info(fmt.Sprintf("status code: %d", resp.StatusCode))
		if err != nil {
			ctx.L().Error("failed to request", zap.Error(err))
		}

		success := err == nil && resp.StatusCode == 202
		if success {
			return ctx.ReplySuccess(fmt.Sprintf(":white_check_mark: Command execution was successful. %s", cite(ctx.P.Message.ID)))
		}
		return ctx.ReplyFailure(fmt.Sprintf(":x: An error has occurred while executing command. %s", cite(ctx.P.Message.ID)))
	default:
		return ctx.ReplyBad(fmt.Sprintf("Unknown command: `%s`", args[0]))
	}
}

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
	if len(s.Operators) > 0 {
		return s.Operators
	}
	return s.Operators
}

func (s *Server) CheckOperator(name string) bool {
	return StringArrayContains(s.GetOperators(), name)
}