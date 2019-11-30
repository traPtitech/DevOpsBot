package main

import (
	"errors"
	"fmt"
	"github.com/dghubble/sling"
	"net/http"
)

var traQClient *sling.Sling

type Map map[string]interface{}

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

func makeInlineMessage(messageId string) string {
	return fmt.Sprintf(`!{"type":"message","id":"%s"}`, messageId)
}
