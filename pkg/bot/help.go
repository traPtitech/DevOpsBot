package bot

import (
	"fmt"
	"strings"

	"github.com/traPtitech/DevOpsBot/pkg/config"
	"github.com/traPtitech/DevOpsBot/pkg/utils"
)

var _ command = (*HelpCommand)(nil)

type HelpCommand struct {
	root *RootCommand
}

func (h *HelpCommand) execute(ctx *Context) error {
	var lines []string
	lines = append(lines, fmt.Sprintf("## DevOpsBot v%s", utils.Version()))
	lines = append(lines, h.root.helpMessage(0)...)
	return ctx.Reply(lines...)
}

func (h *HelpCommand) helpMessage(indent int) []string {
	return []string{fmt.Sprintf(
		"%s- `%shelp` - Display help message.",
		strings.Repeat(" ", indent),
		config.C.Prefix,
	)}
}
