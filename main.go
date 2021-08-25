package main

import (
	"fmt"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/patrickmn/go-cache"
	"net/http"
	"time"

	"github.com/dghubble/sling"
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
	commands["version"] = &VersionCommand{}

	// traQクライアント初期化
	traQClient = sling.New().Base(config.TraqOrigin).Set("Authorization", "Bearer "+config.BotAccessToken)

	// アクセスキーマップ初期化
	logAccessUrls = cache.New(3*time.Minute, 5*time.Minute)

	// HTTPルーター初期化
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.RequestLogger(&AccessLoggingFormatter{l: logger.Named("http")}))
	r.Use(middleware.Recoverer)
	r.Use(middleware.Heartbeat("/ping"))
	r.Post("/_bot", BotEndPoint)
	r.Get("/log/{key}", GetLog)

	// 起動
	if err := SendTRAQMessage(config.DevOpsChannelID, fmt.Sprintf("DevOpsBot `v%s` is ready", version)); err != nil {
		logger.Fatal("failed to send starting message", zap.Error(err))
	}

	logger.Info(fmt.Sprintf("DevOpsBot `v%s` is ready", version))
	http.ListenAndServe(config.BindAddr, r)
}
