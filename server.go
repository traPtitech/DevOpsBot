package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/dghubble/sling"
	"github.com/samber/lo"
	"go.uber.org/zap"
	"golang.org/x/exp/slices"
)

type ServersCommand struct {
	instances map[string]*ServerInstance
}

func (sc *ServersConfig) Compile() (*ServersCommand, error) {
	cmd := &ServersCommand{
		instances: make(map[string]*ServerInstance, len(sc.Servers)),
	}

	for _, sic := range sc.Servers {
		// helpは予約済み
		if sic.Name == "help" {
			return nil, errors.New("`help` cannot be used as server name")
		}
		if sic.ServerID == "" {
			return nil, errors.New("serverID cannot be empty")
		}

		s := &ServerInstance{
			Name:        sic.Name,
			ServerID:    sic.ServerID,
			Description: sic.Description,
			Operators:   sic.Operators,
			Commands:    make(map[string]ServerCommand),
		}
		s.Commands["restart"] = &ServerRestartCommand{s}
		cmd.instances[s.Name] = s
	}
	return cmd, nil
}

// Execute Commandインターフェース実装
func (sc *ServersCommand) Execute(ctx *Context) error {
	if len(ctx.Args) < 2 {
		return ctx.Reply(sc.MakeHelpMessage()...)
	}
	// ctx.Args = server [server_name] restart [SOFT|HARD]
	args := ctx.Args[1:]

	if args[0] == "help" {
		// サーバー一覧表示
		return ctx.Reply(sc.MakeHelpMessage()...)
	}

	s, ok := sc.instances[args[0]]
	if !ok {
		// サーバーが見つからない
		return ctx.ReplyBad(fmt.Sprintf("Unknown server: `%s`", args[0]))
	}
	return s.Execute(ctx)
}

// MakeHelpMessage server help用のメッセージを作成
func (sc *ServersCommand) MakeHelpMessage() []string {
	var lines []string
	lines = append(lines, "## server commands")
	names := lo.Keys(sc.instances)
	slices.Sort(names)
	for _, name := range names {
		s := sc.instances[name]
		operators := strings.Join(lo.Map(s.Operators, func(s string, _ int) string { return `:@` + s + `:` }), "")
		lines = append(lines, fmt.Sprintf(
			"- `%sserver %s restart [SOFT|HARD]`%s (%s)",
			config.Prefix,
			name,
			lo.Ternary(s.Description != "", " - "+s.Description, ""),
			operators,
		))
	}
	return lines
}

type ServerInstance struct {
	Name        string
	ServerID    string
	Description string
	Operators   []string

	Commands map[string]ServerCommand
}

// Execute Commandインターフェース実装
func (s *ServerInstance) Execute(ctx *Context) error {
	if len(ctx.Args) < 3 {
		return ctx.ReplyBad("Invalid Arguments")
	}
	// ctx.Args = server [server_name] restart [SOFT|HARD]
	args := ctx.Args[2:]

	if args[0] == "help" {
		return ctx.Reply(s.MakeHelpMessage()...)
	}

	c, ok := s.Commands[args[0]]
	if !ok {
		// コマンドが見つからない
		return ctx.ReplyBad(fmt.Sprintf("Unknown command: `%s`", args[0]))
	}
	return c.Execute(ctx)
}

// MakeHelpMessage server [name] help用のメッセージを作成
func (s *ServerInstance) MakeHelpMessage() []string {
	var lines []string
	lines = append(lines, fmt.Sprintf("## server: %s", s.Name))
	lines = append(lines, "### usage:")
	lines = append(lines, fmt.Sprintf("`%sserver %s restart [SOFT|HARD]`", config.Prefix, s.Name))
	lines = append(lines, "### operators:")
	lines = append(lines, strings.Join(s.GetOperators(), ", "))
	return lines
}

type ServerCommand interface {
	Execute(ctx *Context) error
}

type ServerRestartCommand struct {
	server *ServerInstance
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

	if !lo.Contains([]string{"SOFT", "HARD"}, args[0]) {
		return ctx.ReplyBad(fmt.Sprintf("Unknown restart type: `%s`", args[0]))
	}

	_ = ctx.ReplyAccept()
	_ = ctx.ReplyRunning()

	token, err := getConohaAPIToken()
	if err != nil {
		ctx.L().Error("failed to get ConoHa API token", zap.Error(err))
		return ctx.ReplyFailure(":x: An error has occurred while getting ConoHa API token. Please retry after a while.")
	}

	req, err := sling.New().
		Base(config.Commands.Servers.Conoha.Origin.Compute).
		Post(fmt.Sprintf("v2/%s/servers/%s/action", config.Commands.Servers.Conoha.TenantID, sc.server.ServerID)).
		BodyJSON(Map{"reboot": Map{"type": args[0]}}).
		Set("Accept", "application/json").
		Set("X-Auth-Token", token).
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
		return ctx.ReplyFailure(":x: A network error has occurred while posing restart request to ConoHa API. Please retry after a while.")
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		ctx.L().Error("failed to read response body", zap.Error(err))
		return ctx.ReplyFailure("An internal error has occurred")
	}

	logStr := fmt.Sprintf(`Request
- URL: %s
- RestartType: %s

Response
- Header: %+v
- Body: %s
- Status: %s (Expected: 202)
`, req.URL.String(), args[0], resp.Header, string(respBody), resp.Status)

	ctx.L().Info(fmt.Sprintf("status code: %s", resp.Status))
	if resp.StatusCode != http.StatusAccepted {
		return ctx.ReplyFailure(fmt.Sprintf(":x: Incorrect status code received from ConoHa API.\n```\n%s\n```", logStr))
	}

	return ctx.ReplySuccess(fmt.Sprintf(":white_check_mark: Command execution was successful.\n```\n%s\n```", logStr))
}

func (s *ServerInstance) GetOperators() []string {
	return s.Operators
}

// CheckOperator nameユーザーがこのコマンドを実行可能かどうか
func (s *ServerInstance) CheckOperator(name string) bool {
	return lo.Contains(s.GetOperators(), name)
}

func getConohaAPIToken() (string, error) {
	type passwordCredentials struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	type auth struct {
		PasswordCredentials passwordCredentials `json:"passwordCredentials"`
		TenantId            string              `json:"tenantId"`
	}
	requestJson := struct {
		Auth auth `json:"auth"`
	}{
		Auth: auth{
			PasswordCredentials: passwordCredentials{
				Username: config.Commands.Servers.Conoha.Username,
				Password: config.Commands.Servers.Conoha.Password,
			},
			TenantId: config.Commands.Servers.Conoha.TenantID,
		},
	}

	req, err := sling.New().
		Base(config.Commands.Servers.Conoha.Origin.Identity).
		Post("v2.0/tokens").
		BodyJSON(requestJson).
		Set("Accept", "application/json").
		Request()
	if err != nil {
		return "", fmt.Errorf("failed to create authentication request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to post authentication request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("invalid status code: %s (expected: 200)", resp.Status)
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	var responseJson struct {
		Access struct {
			Token struct {
				Id string `json:"id"`
			} `json:"token"`
		} `json:"access"`
	}
	err = json.Unmarshal(respBody, &responseJson)
	if err != nil {
		return "", fmt.Errorf("failed to unmarshal response body: %w", err)
	}

	return responseJson.Access.Token.Id, nil
}
