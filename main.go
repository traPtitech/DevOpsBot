package main

import (
	"net/http"
	"time"

	"github.com/dghubble/sling"
	ginzap "github.com/gin-contrib/zap"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
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
		panic(err)
	}
	defer logger.Sync()

	// 設定ファイル読み込み
	config, err = LoadConfig(getEnvOrDefault("CONFIG_FILE", "./config.yml"))
	if err != nil {
		logger.Fatal("failed to load config", zap.Error(err))
	}
	commands["service"] = config.Services

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
