package main

import (
	"fmt"
	"github.com/patrickmn/go-cache"
	"net/http"
	"time"

	"github.com/dghubble/sling"
	ginzap "github.com/gin-contrib/zap"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

var (
	version       = "UNKNOWN"
	config        *Config
	logger        *zap.Logger
	logAccessUrls *cache.Cache
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
	commands["exec-log"] = &ExecLogCommand{}

	// traQクライアント初期化
	traQClient = sling.New().Base(config.TraqOrigin).Set("Authorization", "Bearer "+config.BotAccessToken)

	// アクセスキーマップ初期化
	logAccessUrls = cache.New(3*time.Minute, 5*time.Minute)

	// HTTPルーター初期化
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(ginzap.Ginzap(logger, time.RFC3339Nano, false))
	router.Use(ginzap.RecoveryWithZap(logger, true))

	router.POST("/_bot", BotEndPoint)
	router.GET("/health", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})
	router.GET("/log/:key", GetLog)

	// 起動
	if err := SendTRAQMessage(config.DevOpsChannelID, fmt.Sprintf("DevOpsBot `%s` is ready", version)); err != nil {
		logger.Fatal("failed to send starting message", zap.Error(err))
	}

	router.Run(config.BindAddr)
}
