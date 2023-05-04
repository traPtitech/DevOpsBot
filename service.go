package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/kballard/go-shellquote"
	"go.uber.org/zap"
	"golang.org/x/crypto/ssh"
)

type Services map[string]*Service

func (sc *ServicesConfig) Compile() (Services, error) {
	ss := make(Services, len(sc.Services))
	for _, s := range sc.Services {
		// helpは予約済み
		if s.Name == "help" {
			return nil, errors.New("`help` cannot be used as service name")
		}
		for name, c := range s.Commands {
			// helpは予約済み
			if name == "help" {
				return nil, errors.New("`help` cannot be used as service command name")
			}
			c.Name = name
			c.service = s
		}
	}
	return ss, nil
}

// Execute Commandインターフェース実装
func (ss Services) Execute(ctx *Context) error {
	if len(ctx.Args) < 2 {
		return ctx.ReplyBad("Invalid Arguments")
	}
	// ctx.Args = service [name] [command]
	args := ctx.Args[1:]

	if args[0] == "help" {
		// サービス一覧表示
		return ctx.Reply(ss.MakeHelpMessage(), "")
	}

	s, ok := ss[args[0]]
	if !ok {
		// サービスが見つからない
		return ctx.ReplyBad(fmt.Sprintf("Unknown service: `%s`", args[0]))
	}
	return s.Execute(ctx)
}

// MakeHelpMessage service help用のメッセージを作成
func (ss Services) MakeHelpMessage() string {
	var sb strings.Builder
	sb.WriteString("## service\n")
	sb.WriteString("### usage:\n")
	sb.WriteString("`service [service_name] [command]`\n")
	sb.WriteString("### services:\n")
	for name, s := range ss {
		if len(s.Description) > 0 {
			sb.WriteString(fmt.Sprintf("+ `%s` - %s\n", name, s.Description))
		} else {
			sb.WriteString(fmt.Sprintf("+ `%s`\n", name))
		}
	}
	return sb.String()
}

// Service サービス
type Service struct {
	// Name サービス名
	Name string `yaml:"name"`
	// Description サービス説明
	Description string `yaml:"description"`
	// Host サービス稼働ホスト名
	//
	// このホスト上でコマンドが実行されます
	Host string `yaml:"host"`
	// SSHPort サービス稼働ホストのSSHポート番号
	SSHPort int `yaml:"sshPort"`
	// SSHUser サービス稼働ホストのSSHユーザー名
	SSHUser string `yaml:"sshUser"`
	// Operators コマンド実行可能なユーザーの名前のデフォルト配列
	//
	// 各コマンド設定においてオーバーライド可能
	Operators []string `yaml:"operators"`
	// Commands サービスコマンド
	Commands map[string]*ServiceCommand `yaml:"commands"`
}

// Execute Commandインターフェース実装
func (s *Service) Execute(ctx *Context) error {
	if len(ctx.Args) < 3 {
		return ctx.ReplyBad("Invalid Arguments")
	}
	// ctx.Args = service [name] [command]
	args := ctx.Args[2:]

	if args[0] == "help" {
		// サービスヘルプを表示
		return ctx.Reply(s.MakeHelpMessage(), "")
	}

	c, ok := s.Commands[args[0]]
	if !ok {
		// コマンドが見つからない
		return ctx.ReplyBad(fmt.Sprintf("Unknown command: `%s`", args[0]))
	}
	return c.Execute(ctx)
}

// MakeHelpMessage service [name] help用のメッセージを作成
func (s *Service) MakeHelpMessage() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("## service: %s\n", s.Name))
	sb.WriteString("### usage:\n")
	sb.WriteString(fmt.Sprintf("`service %s [command]`\n", s.Name))
	sb.WriteString("### commands:\n")
	for name, c := range s.Commands {
		sb.WriteString(fmt.Sprintf("+ `%s`\n", name))

		if len(s.Description) > 0 {
			sb.WriteString(fmt.Sprintf("  + %s\n", s.Description))
		}

		var quotedUsers []string
		for _, u := range c.GetOperators() {
			quotedUsers = append(quotedUsers, fmt.Sprintf("`%s`", u))
		}
		sb.WriteString(fmt.Sprintf("  + available users: %s\n", strings.Join(quotedUsers, ",")))
	}
	return sb.String()
}

// ServiceCommand サービスコマンド
type ServiceCommand struct {
	// Name コマンド名
	Name string `yaml:"-"`
	// Description コマンド説明
	Description string `yaml:"description"`
	// Host サービス稼働ホスト名
	//
	// この設定はサービス設定のHostをオーバーライドします
	Host string `yaml:"host"`
	// SSHPort サービス稼働ホストのSSHポート番号
	//
	// この設定はサービス設定のSSHPortをオーバーライドします
	SSHPort int `yaml:"sshPort"`
	// SSHUser サービス稼働ホストのSSHユーザー名
	//
	// この設定はサービス設定のSSHUserをオーバーライドします
	SSHUser string `yaml:"sshUser"`
	// Command 実行コマンド
	Command string `yaml:"command"`
	// CommandArgs 実行コマンド引数
	CommandArgs []string `yaml:"commandArgs"`
	// WorkingDirectory コマンド実行ディレクトリ
	WorkingDirectory string `yaml:"workingDir"`
	// Operators コマンド実行可能なユーザーの名前の配列
	//
	// この設定はサービス設定のOperatorsをオーバーライドします
	Operators []string `yaml:"operators"`
	// AllowConcurrency このコマンドの同時並列実行を許可するか
	AllowConcurrency bool `yaml:"allowConcurrency"`
	// AppendVariableArgs このコマンドの実行引数に、メッセージから追加で与えられた引数を追記するか
	AppendVariableArgs bool `yaml:"appendVariableArgs"`
	// PrintOutputOnMessage コマンド実行結果をメッセージとして送信するか
	PrintOutputOnMessage bool `yaml:"printOutputOnMessage"`

	service *Service   `yaml:"-"`
	running bool       `yaml:"-"`
	m       sync.Mutex `yaml:"-"`
}

func (sc *ServiceCommand) GetExecutionHost() string {
	if len(sc.Host) > 0 {
		return sc.Host
	}
	return sc.service.Host
}

func (sc *ServiceCommand) GetSSHPort() int {
	switch {
	case sc.SSHPort != 0:
		return sc.SSHPort
	case sc.service.SSHPort != 0:
		return sc.service.SSHPort
	default:
		return 22
	}
}

func (sc *ServiceCommand) GetSSHUser() string {
	switch {
	case len(sc.SSHUser) > 0:
		return sc.SSHUser
	case len(sc.service.SSHUser) > 0:
		return sc.service.SSHUser
	default:
		return config.Commands.Services.DefaultSSHUser
	}
}

func (sc *ServiceCommand) GetOperators() []string {
	if len(sc.Operators) > 0 {
		return sc.Operators
	}
	return sc.service.Operators
}

// CheckOperator nameユーザーがこのコマンドを実行可能かどうか
func (sc *ServiceCommand) CheckOperator(name string) bool {
	return StringArrayContains(sc.GetOperators(), name)
}

// Execute Commandインターフェース実装
func (sc *ServiceCommand) Execute(ctx *Context) error {
	// ctx.Args = service [name] [command]

	// オペレーター確認
	if !sc.CheckOperator(ctx.GetExecutor()) {
		return ctx.ReplyForbid()
	}

	if !sc.AllowConcurrency {
		// 同時実行ロック
		sc.m.Lock()
		if sc.running {
			// 既に実行中
			sc.m.Unlock()
			return nil
		}
		sc.running = true
		sc.m.Unlock()
	}

	_ = ctx.ReplyAccept()

	ctx.L().Info("shell command execution starts")
	_ = ctx.ReplyRunning()
	err := sc.execute(ctx)
	ctx.L().Info("shell command execution ends", zap.Error(err))

	sc.m.Lock()
	sc.running = false
	sc.m.Unlock()

	success := err == nil

	if !sc.PrintOutputOnMessage {
		if success {
			return ctx.ReplySuccess(fmt.Sprintf(":white_check_mark: Command execution was successful. \n log: `exec-log service %s %s %d` %s", sc.service.Name, sc.Name, ctx.P.EventTime.Unix(), cite(ctx.P.Message.ID)))
		}
		return ctx.ReplyFailure(fmt.Sprintf(":x: An error has occurred while executing command. \nPlease check the execution log. `exec-log service %s %s %d` %s", sc.service.Name, sc.Name, ctx.P.EventTime.Unix(), cite(ctx.P.Message.ID)))
	}

	logFile, err := sc.openLogFile(ctx)
	if err != nil {
		ctx.L().Error("failed to open log file", zap.Error(err))
		return ctx.ReplyFailure("An internal error has occurred")
	}
	defer logFile.Close()
	b, err := io.ReadAll(io.LimitReader(logFile, 1<<20)) // 1KBまでに抑えとく
	if err != nil {
		ctx.L().Error("failed to read log file", zap.Error(err))
		return ctx.ReplyFailure("An internal error has occurred")
	}

	var message strings.Builder
	if success {
		message.WriteString(":white_check_mark: Command execution was successful.\n")
	} else {
		message.WriteString(":x: An error has occurred while executing command.\n")
	}
	message.WriteString(fmt.Sprintf("```:exec-%s-%s-%d\n", sc.service.Name, sc.Name, ctx.P.EventTime.Unix()))
	message.Write(b)
	if !strings.HasSuffix(string(b), "\n") {
		message.WriteString("\n")
	}
	message.WriteString("```\n")
	message.WriteString(cite(ctx.P.Message.ID))

	if success {
		return ctx.ReplySuccess(message.String())
	}
	return ctx.ReplyFailure(message.String())
}

func (sc *ServiceCommand) execute(ctx *Context) error {
	// local or remote
	switch sc.GetExecutionHost() {
	case "", config.Commands.Services.LocalHostName:
		return sc.executeLocal(ctx)
	default:
		return sc.executeRemote(ctx)
	}
}

// executeRemote SSHで実行
func (sc *ServiceCommand) executeRemote(ctx *Context) error {
	// ログファイル生成
	logFile, err := sc.openLogFile(ctx)
	if err != nil {
		return err
	}
	defer logFile.Close()

	// 秘密鍵のパース
	key, err := ssh.ParsePrivateKey([]byte(config.Commands.Services.SSHPrivateKey))
	if err != nil {
		ctx.L().Error("failed to read private key for ssh", zap.Error(err))
		return err
	}

	// ssh接続
	host := sc.GetExecutionHost()
	conn, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", host, sc.GetSSHPort()), &ssh.ClientConfig{
		User: sc.GetSSHUser(),
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(key),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	})
	if err != nil {
		ctx.L().Error(fmt.Sprintf("failed to ssh to %s", host), zap.Error(err))
		return err
	}
	defer conn.Close()
	session, err := conn.NewSession()
	if err != nil {
		ctx.L().Error(fmt.Sprintf("failed to create ssh session to %s", host), zap.Error(err))
		return err
	}
	defer session.Close()

	// ログファイル設定
	session.Stdout = logFile
	session.Stderr = logFile

	// コマンド生成
	execCmd := append([]string{sc.Command}, sc.CommandArgs...)
	if sc.AppendVariableArgs {
		execCmd = append(execCmd, ctx.Args[3:]...)
	}

	cmdStr := fmt.Sprintf("%s && %s", shellquote.Join("cd", sc.WorkingDirectory), shellquote.Join(execCmd...))

	// コマンド実行
	if err = session.Start(cmdStr); err != nil {
		ctx.L().Error("failed to execute shell command", zap.String("cmd", cmdStr), zap.Error(err))
		return err
	}

	// 終了待機
	return session.Wait()
}

// executeLocal ローカルで実行
func (sc *ServiceCommand) executeLocal(ctx *Context) error {
	// ログファイル生成
	logFile, err := sc.openLogFile(ctx)
	if err != nil {
		return err
	}
	defer logFile.Close()

	// execコマンド生成
	args := make([]string, 0)
	args = append(args, sc.CommandArgs...)
	if sc.AppendVariableArgs {
		args = append(args, ctx.Args[3:]...)
	}

	cmd := exec.Command(sc.Command, args...)
	cmd.Dir = sc.WorkingDirectory

	// ログファイル設定
	cmd.Stdout = logFile
	cmd.Stderr = logFile

	// コマンド実行
	if err := cmd.Start(); err != nil {
		ctx.L().Error("failed to execute shell command", zap.Stringer("cmd", cmd), zap.Error(err))
		return err
	}

	// 終了待機
	return cmd.Wait()
}

// openLogFile ログファイルを開く
func (sc *ServiceCommand) openLogFile(ctx *Context) (*os.File, error) {
	logFilePath := filepath.Join(config.Commands.Services.LogsDir, sc.getLogFileName(ctx))
	logFile, err := os.OpenFile(logFilePath, os.O_RDWR|os.O_CREATE, 0600)
	if err != nil {
		ctx.L().Error("failed to open log file", zap.String("path", logFilePath), zap.Error(err))
		return nil, err
	}
	return logFile, nil
}

func (sc *ServiceCommand) getLogFileName(ctx *Context) string {
	return sc.getLogFileNameByUnixTime(ctx.P.EventTime.Unix())
}

func (sc *ServiceCommand) getLogFileNameByUnixTime(unix int64) string {
	return fmt.Sprintf("exec-%s-%s-%d", sc.service.Name, sc.Name, unix)
}
