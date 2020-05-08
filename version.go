package main

import "fmt"

// VersionCommand `version`
type VersionCommand struct{}

func (v VersionCommand) Execute(ctx *Context) error {
	return ctx.ReplySuccess(fmt.Sprintf("DevOpsBot `v%s`", version))
}
