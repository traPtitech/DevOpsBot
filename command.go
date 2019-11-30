package main

import (
	"fmt"
	"go.uber.org/zap"
)

var commandList = map[string]func(p MessageCreatedPayload, args []string){
	"deploy": commandDeploy,
}

// デプロイコマンド
// deploy [target]
func commandDeploy(p MessageCreatedPayload, args []string) {
	if len(args) != 2 {
		// 引数の数がおかしい
		PushTRAQStamp(p.Message.ID, config.Stamps.BadCommand)
		return
	}

	target, ok := config.Deploys[args[1]]
	if !ok {
		// デプロイターゲットが見つからない
		PushTRAQStamp(p.Message.ID, config.Stamps.BadCommand)
		return
	}

	if !StringArrayContains(target.Operators, p.Message.User.Name) {
		// 許可されてない操作者
		PushTRAQStamp(p.Message.ID, config.Stamps.Forbid)
		return
	}

	target.mx.Lock()
	isRunning := target.isRunning
	if !isRunning {
		target.isRunning = true
	}
	target.mx.Unlock()

	if isRunning {
		// 既に実行中
		return
	}

	PushTRAQStamp(p.Message.ID, config.Stamps.Accept)

	logger.Info("deploy starts", zap.String("target", target.Name))
	err := DoDeploy(target)
	logger.Info("deploy completes", zap.String("target", target.Name))

	target.mx.Lock()
	target.isRunning = false
	target.mx.Unlock()

	if err != nil {
		PushTRAQStamp(p.Message.ID, config.Stamps.Failure)
		SendTRAQMessage(p.Message.ChannelID, fmt.Sprintf("エラーが発生しました。詳しくはログを確認してください。%s", cite(p.Message.ID)))
	} else {
		PushTRAQStamp(p.Message.ID, config.Stamps.Success)
		SendTRAQMessage(p.Message.ChannelID, fmt.Sprintf("完了しました%s", cite(p.Message.ID)))
	}
}
