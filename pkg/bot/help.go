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
	args := ctx.Args()

	// Root usage
	if len(args) == 0 {
		lines = append(lines, fmt.Sprintf("## DevOpsBot v%s", utils.Version()))
		lines = append(lines, "")
		lines = append(lines, h.root.HelpMessage(0, true)...)
		lines = append(lines, "")
		lines = append(lines, fmt.Sprintf("Type `%shelp command-name` for more help", config.C.Prefix))
		return ctx.ReplySuccess(lines...)
	}

	// Specific command usage
	c, ok := h.root.getMatchingCommand(args)
	if !ok {
		lines = append(lines, fmt.Sprintf("Command `%s%s` not found, try `%shelp`?", config.C.Prefix, strings.Join(args, " "), config.C.Prefix))
		return ctx.ReplyBad(lines...)
	}

	lines = append(lines, fmt.Sprintf("## `%s%s` Usage", config.C.Prefix, strings.Join(args, " ")))
	lines = append(lines, "")
	lines = append(lines, c.HelpMessage(0, true)...)
	if c.HasSubcommands() {
		lines = append(lines, "")
		lines = append(lines, fmt.Sprintf("Type `%shelp command-name [sub-commands...]` for more help", config.C.Prefix))
	}
	return ctx.ReplySuccess(lines...)
}

func (h *HelpCommand) HasSubcommands() bool {
	return false
}

func (h *HelpCommand) GetSubcommand(_ string) (domain.Command, bool) {
	return nil, false
}

func (h *HelpCommand) HelpMessage(indent int, _ bool) []string {
	return []string{fmt.Sprintf(
		"%s- `%shelp` - Display help message.",
		strings.Repeat(" ", indent),
		config.C.Prefix,
	)}
}
