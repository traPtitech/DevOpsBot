package main

import (
	"fmt"
)

type HelpCommand struct{}

func (h *HelpCommand) Execute(ctx *Context) error {
	var lines []string
	lines = append(lines, fmt.Sprintf("## DevOpsBot v%s", version))
	lines = append(lines, fmt.Sprintf("- `%sdeploy` - Do deployments", config.Prefix))
	lines = append(lines, fmt.Sprintf("- `%sserver` - Server management", config.Prefix))
	lines = append(lines, fmt.Sprintf("- `%sexec-log` - Retrieve command logs", config.Prefix))
	lines = append(lines, fmt.Sprintf("- `%shelp` - This help", config.Prefix))
	return ctx.Reply(lines...)
}
