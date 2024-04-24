package bot

import (
	"fmt"

	"github.com/traPtitech/DevOpsBot/pkg/config"
	"github.com/traPtitech/DevOpsBot/pkg/utils"
)

type HelpCommand struct{}

func (h *HelpCommand) Execute(ctx *Context) error {
	var lines []string
	lines = append(lines, fmt.Sprintf("## DevOpsBot v%s", utils.Version()))
	lines = append(lines, fmt.Sprintf("- `%sdeploy` - Do deployments", config.C.Prefix))
	lines = append(lines, fmt.Sprintf("- `%sserver` - Server management", config.C.Prefix))
	lines = append(lines, fmt.Sprintf("- `%shelp` - This help", config.C.Prefix))
	return ctx.Reply(lines...)
}
