package main

import (
	"fmt"
	"github.com/dghubble/sling"
	ginzap "github.com/gin-contrib/zap"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"time"
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
	// execコマンド生成
	cmd := exec.Command(dc.Command, dc.CommandArgs...)

	// ログファイル生成
	logFilePath := filepath.Join(config.LogsDir, fmt.Sprintf("deploy-%s-%d", dc.Name, time.Now().Unix()))
	logFile, err := os.Create(logFilePath)
	if err != nil {
		logger.Error("failed to create log file", zap.String("path", logFilePath), zap.Error(err))
		return err
	}
	defer logFile.Close()

	cmd.Stdout = logFile
	cmd.Stderr = logFile

	// コマンド実行
	if err = cmd.Start(); err != nil {
		logger.Error("failed to execute command", zap.Stringer("cmd", cmd), zap.Error(err))
		return err
	}

	// 終了待機
	return cmd.Wait()
}
