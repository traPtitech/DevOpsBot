package bot

import (
	"fmt"
	"strings"

	"github.com/traPtitech/DevOpsBot/pkg/config"
	"github.com/traPtitech/DevOpsBot/pkg/domain"
	"github.com/traPtitech/DevOpsBot/pkg/utils"
)

var _ domain.Command = (*HelpCommand)(nil)

type HelpCommand struct {
	root *RootCommand
}

func (h *HelpCommand) Execute(ctx domain.Context) error {
	var lines []string
	lines = append(lines, fmt.Sprintf("## DevOpsBot v%s", utils.Version()))
	lines = append(lines, h.root.HelpMessage(0)...)
	return ctx.ReplySuccess(lines...)
}

func (h *HelpCommand) HelpMessage(indent int) []string {
	return []string{fmt.Sprintf(
		"%s- `%shelp` - Display help message.",
		strings.Repeat(" ", indent),
		config.C.Prefix,
	)}
}
