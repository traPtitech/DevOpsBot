package main

import (
	"fmt"
	"go.uber.org/zap"
	"golang.org/x/crypto/ssh"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
)

// Services サービスマップ
type Services map[string]*Service

// UnmarshalYAML gopkg.in/yaml.v2.Unmarshaler 実装
func (ss *Services) UnmarshalYAML(unmarshal func(interface{}) error) error {
	if err := unmarshal(ss); err != nil {
		return err
	}
	for name, s := range *ss {
		s.Name = name
	}
	return nil
}

// Execute Commandインターフェース実装
func (ss Services) Execute(ctx *Context) error {
	if len(ctx.Args) < 2 {
		return ctx.ReplyBad("Invalid Arguments")
	}
	// ctx.Args = service [name] [command]
	args := ctx.Args[1:]

	s, ok := ss[args[0]]
	if !ok {
		// サービスが見つからない
		return ctx.ReplyBad(fmt.Sprintf("Unknown service: `%s`", args[0]))
	}
	return s.Execute(ctx)
}

// Service サービス
type Service struct {
	// Name サービス名
	Name string `yaml:"-"`
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

// UnmarshalYAML gopkg.in/yaml.v2.Unmarshaler 実装
func (s *Service) UnmarshalYAML(unmarshal func(interface{}) error) error {
	if err := unmarshal(s); err != nil {
		return err
	}
	for name, c := range s.Commands {
		c.Name = name
		c.service = s
	}
	return nil
}

// Execute Commandインターフェース実装
func (s *Service) Execute(ctx *Context) error {
	if len(ctx.Args) < 3 {
		return ctx.ReplyBad("Invalid Arguments")
	}
	// ctx.Args = service [name] [command]
	args := ctx.Args[2:]

	c, ok := s.Commands[args[0]]
	if !ok {
		// コマンドが見つからない
		return ctx.ReplyBad(fmt.Sprintf("Unknown command: `%s`", args[0]))
	}
	return c.Execute(ctx)
}

// ServiceCommand サービスコマンド
type ServiceCommand struct {
	// Name コマンド名
	Name string `yaml:"-"`
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
		return config.DefaultSSHUser
	}
}

// CheckOperator nameユーザーがこのコマンドを実行可能かどうか
func (sc *ServiceCommand) CheckOperator(name string) bool {
	if len(sc.Operators) > 0 {
		return StringArrayContains(sc.Operators, name)
	}
	return StringArrayContains(sc.service.Operators, name)
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
	err := sc.execute(ctx)
	ctx.L().Info("shell command execution ends", zap.Error(err))

	sc.m.Lock()
	sc.running = false
	sc.m.Unlock()

	if err != nil {
		return ctx.ReplyFailure(fmt.Sprintf("エラーが発生しました。詳しくはログを確認してください。%s", cite(ctx.P.Message.ID)))
	}
	return ctx.ReplySuccess(fmt.Sprintf("Command execution was successful: %s", cite(ctx.P.Message.ID)))
}

func (sc *ServiceCommand) execute(ctx *Context) error {
	// local or remote
	switch sc.GetExecutionHost() {
	case "", config.LocalHostName:
		return sc.executeLocal(ctx)
	default:
		return sc.executeRemote(ctx)
	}
}

// executeRemote SSHで実行
func (sc *ServiceCommand) executeRemote(ctx *Context) error {
	// ログファイル生成
	logFile, err := sc.makeLogFile(ctx)
	if err != nil {
		return err
	}
	defer logFile.Close()

	// 秘密鍵のパース
	key, err := ssh.ParsePrivateKey([]byte(config.SSHPrivateKey))
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
	cmdStr := fmt.Sprintf("cd %s; %s %s", sc.WorkingDirectory, sc.Command, strings.Join(sc.CommandArgs, " "))

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
	logFile, err := sc.makeLogFile(ctx)
	if err != nil {
		return err
	}
	defer logFile.Close()

	// execコマンド生成
	cmd := exec.Command(sc.Command, sc.CommandArgs...)
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

// makeLogFile ログファイル生成
func (sc *ServiceCommand) makeLogFile(ctx *Context) (*os.File, error) {
	logFilePath := filepath.Join(config.LogsDir, fmt.Sprintf("execution-%s-%s-%d", sc.service.Name, sc.Name, ctx.P.EventTime.Unix()))
	logFile, err := os.Create(logFilePath)
	if err != nil {
		ctx.L().Error("failed to create log file", zap.String("path", logFilePath), zap.Error(err))
		return nil, err
	}
	return logFile, nil
}
