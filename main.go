package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/dghubble/sling"
	ginzap "github.com/gin-contrib/zap"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"golang.org/x/crypto/ssh"
)

var (
	config *Config
	logger *zap.Logger
)

func main() {
	var err error

	// ロガー初期化
	logger, err = zap.NewProduction()
	if err != nil {
		log.Fatal(err)
	}
	defer logger.Sync()

	// 設定ファイル読み込み
	configFile := os.Getenv("CONFIG_FILE")
	if configFile == "" {
		configFile = "./config.yml"
	}
	config, err = LoadConfig(configFile)
	if err != nil {
		logger.Fatal("failed to load config", zap.Error(err))
	}

	// traQクライアント初期化
	traQClient = sling.New().Base(config.TraqOrigin).Set("Authorization", "Bearer "+config.BotAccessToken)

	// HTTPルーター初期化
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(ginzap.Ginzap(logger, time.RFC3339Nano, false))
	router.Use(ginzap.RecoveryWithZap(logger, true))

	router.POST("/_bot", BotEndPoint)
	router.GET("/health", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	router.Run(config.BindAddr)
}

func DoDeploy(dc *DeployConfig) error {
	// ログファイル生成
	logFilePath := filepath.Join(config.LogsDir, fmt.Sprintf("deploy-%s-%d", dc.Name, time.Now().Unix()))
	logFile, err := os.Create(logFilePath)
	if err != nil {
		logger.Error("failed to create log file", zap.String("path", logFilePath), zap.Error(err))
		return err
	}
	defer logFile.Close()

	if dc.Host == config.DeployerHost {
		return DoDeployLocal(dc, logFile)
	}
	return DoDeployRemote(dc, logFile)
}

func DoDeployLocal(dc *DeployConfig, logFile *os.File) error {
	// execコマンド生成
	cmd := exec.Command(dc.Command, dc.CommandArgs...)
	cmd.Dir = dc.WorkingDirectory

	// ログファイル設定
	cmd.Stdout = logFile
	cmd.Stderr = logFile

	// コマンド実行
	if err := cmd.Start(); err != nil {
		logger.Error("failed to execute command", zap.Stringer("cmd", cmd), zap.Error(err))
		return err
	}

	// 終了待機
	return cmd.Wait()
}

func DoDeployRemote(dc *DeployConfig, logFile *os.File) error {
	// 秘密鍵のパース
	key, err := ssh.ParsePrivateKey([]byte(config.DeployerPrivateKey))
	if err != nil {
		logger.Error("failed to read private key for ssh", zap.Error(err))
		return err
	}

	// ssh用の設定
	config := &ssh.ClientConfig{
		User: config.DeployerUserName,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(key),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	// ssh接続
	conn, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", dc.Host, 22), config)
	if err != nil {
		logger.Error("failed to ssh to "+dc.Host, zap.Error(err))
		return err
	}
	defer conn.Close()
	session, err := conn.NewSession()
	defer session.Close()
	if err != nil {
		logger.Error("failed to create ssh session to "+dc.Host, zap.Error(err))
		return err
	}

	// ログファイル設定
	session.Stdout = logFile
	session.Stderr = logFile

	// コマンド生成
	cmdStr := "cd " + dc.WorkingDirectory + "; " + dc.Command + " " + strings.Join(dc.CommandArgs, " ")

	// コマンド実行
	if err = session.Start(cmdStr); err != nil {
		logger.Error("failed to execute command: "+cmdStr, zap.Error(err))
		return err
	}

	// 終了待機
	return session.Wait()
}
