package main

import (
	"errors"
	"fmt"
	"github.com/dghubble/sling"
	"github.com/gin-gonic/gin"
	"net/http"
	"strings"
	"time"
)

var traQClient *sling.Sling

type Map map[string]interface{}

// MessageCreatedPayload MESSAGE_CREATEDイベントペイロード
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

// BotEndPoint Botサーバーエンドポイント
func BotEndPoint(c *gin.Context) {
	// トークン検証
	if c.GetHeader("X-TRAQ-BOT-TOKEN") != config.VerificationToken {
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

// BotMessageReceived BOTのMESSAGE_CREATEDイベントハンドラ
func BotMessageReceived(p MessageCreatedPayload) {
	if p.Message.ChannelID != config.DevOpsChannelID {
		return // DevOpsチャンネル以外からのメッセージは無視
	}

	args := strings.Fields(p.Message.PlainText)
	c, ok := commandList[args[0]]
	if ok {
		c(p, args)
	}
}

// SendTRAQMessage traQにメッセージ送信
func SendTRAQMessage(channelID string, text string) error {
	req, err := traQClient.New().
		Post(fmt.Sprintf("api/1.0/channels/%s/messages", channelID)).
		BodyJSON(Map{"text": text}).
		Request()
	if err != nil {
		return err
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusCreated {
		return errors.New(res.Status)
	}
	return nil
}

// PushTRAQStamp traQのメッセージにスタンプを押す
func PushTRAQStamp(messageID, stampID string) error {
	req, err := traQClient.New().
		Post(fmt.Sprintf("api/1.0/messages/%s/stamps/%s", messageID, stampID)).
		Request()
	if err != nil {
		return err
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusNoContent {
		return errors.New(res.Status)
	}
	return nil
}

// cite traQのメッセージ引用形式を作る
func cite(messageId string) string {
	return fmt.Sprintf(`!{"raw":"","type":"message","id":"%s"}`, messageId)
}
