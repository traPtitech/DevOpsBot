package main

import (
	"github.com/dghubble/sling"
	"github.com/gin-gonic/gin"
	"log"
	"os"
	"os/exec"
)

var (
	verificationToken = os.Getenv("BOT_VERIFICATION_TOKEN")
	accessToken       = os.Getenv("BOT_ACCESS_TOKEN")
	configFile        = os.Getenv("CONFIG_FILE")
)

var config *Config

func main() {
	var err error

	// 設定ファイル読み込み
	config, err = LoadConfig(configFile)
	if err != nil {
		log.Fatal(err)
	}

	// traQクライアント初期化
	traQClient = sling.New().Base(config.TraqOrigin).Set("Authorization", "Bearer "+accessToken)

	// HTTPルーター初期化
	router := gin.Default()
	router.POST("/_bot", BotEndPoint)

	router.Run(":6666")
}

func DoDeploy(dc DeployConfig) error {
	cmd := exec.Command(dc.Command, dc.CommandArgs...)

	err := cmd.Run()
	if err != nil {
		return err
	}
	return nil
}
