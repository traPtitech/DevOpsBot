package main

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
	"strings"
	"time"
)

type MessageCreatedPayload struct {
	EventTime time.Time `json:"eventTime"`
	Message   struct {
		ID   string `json:"id"`
		User struct {
			ID   string `json:"id"`
			Name string `json:"name"`
			Bot  bool   `json:"bot"`
		} `json:"user"`
		ChannelID string `json:"channelId"`
		Text      string `json:"text"`
		PlainText string `json:"plainText"`
		CreatedAt string `json:"createdAt"`
	} `json:"message"`
}

func BotEndPoint(c *gin.Context) {
	// トークン検証
	if c.GetHeader("X-TRAQ-BOT-TOKEN") != verificationToken {
		c.Status(http.StatusUnauthorized)
		return
	}

	event := c.GetHeader("X-TRAQ-BOT-EVENT")
	switch event {
	case "PING", "JOINED", "LEFT":
		c.Status(http.StatusNoContent)
	case "MESSAGE_CREATED":
		var payload MessageCreatedPayload
		if err := c.ShouldBindJSON(&payload); err != nil {
			c.Status(http.StatusBadRequest)
			return
		}
		go BotMessageReceived(payload)
		c.Status(http.StatusNoContent)
	default:
		c.Status(http.StatusBadRequest)
	}
}

func BotMessageReceived(p MessageCreatedPayload) {
	if p.Message.ChannelID != config.DevOpsChannelID {
		return // DevOpsチャンネル以外からのメッセージは無視
	}

	commands := strings.Fields(p.Message.PlainText)
	switch commands[0] {
	case "deploy": // デプロイコマンド
		if len(commands) != 2 {
			SendTRAQMessage(p.Message.ChannelID, fmt.Sprintf("不正なコマンドです%s", makeInlineMessage(p.Message.ID)))
			return
		}

		target, ok := config.Deploys[commands[1]]
		if !ok {
			// デプロイターゲットが見つからない
			SendTRAQMessage(p.Message.ChannelID, fmt.Sprintf("デプロイターゲットが見つかりません: %s %s", target, makeInlineMessage(p.Message.ID)))
			return
		}

		if !StringArrayContains(target.Operators, p.Message.User.Name) {
			// 許可されてない操作者
			SendTRAQMessage(p.Message.ChannelID, fmt.Sprintf("権限がありません%s", makeInlineMessage(p.Message.ID)))
			return
		}

		target.mx.Lock()
		isRunning := target.isRunning
		if !isRunning {
			target.isRunning = true
		}
		target.mx.Unlock()

		if isRunning {
			SendTRAQMessage(p.Message.ChannelID, fmt.Sprintf("現在実行中です%s", makeInlineMessage(p.Message.ID)))
			return
		}

		PushTRAQStamp(p.Message.ID, config.Stamps.ThumbsUp)

		DoDeploy(target)

		target.mx.Lock()
		isRunning = false
		target.mx.Unlock()

		SendTRAQMessage(p.Message.ChannelID, fmt.Sprintf("完了しました%s", makeInlineMessage(p.Message.ID)))
	}
}
