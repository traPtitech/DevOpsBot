package bot

import (
	"bytes"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"slices"
	"strings"

	"github.com/samber/lo"

	"github.com/traPtitech/DevOpsBot/pkg/config"
	"github.com/traPtitech/DevOpsBot/pkg/domain"
	"github.com/traPtitech/DevOpsBot/pkg/utils"
)

var (
	_ domain.Command = (*RootCommand)(nil)
	_ domain.Command = (*CommandInstance)(nil)
)

type RootCommand struct {
	cmds map[string]domain.Command
}

type CommandInstance struct {
	leadingMatcher []string
	name           string
	description    string
	allowArgs      bool
	argsSyntax     string
	argsPrefix     []string
	operators      []string

	commandFile string
	subCommands map[string]domain.Command
}

func Compile() (*RootCommand, error) {
	cmd := &RootCommand{
		cmds: make(map[string]domain.Command),
	}

	// Compile templates
	templates := make(map[string]string, len(config.C.Templates)) // template name to filename
	for _, tc := range config.C.Templates {
		if tc.Name == "" {
			return nil, fmt.Errorf("template needs to have a name")
		}
		if _, ok := templates[tc.Name]; ok {
			return nil, fmt.Errorf("template %s conflict", tc.Name)
		}
		if tc.Command != "" && tc.ExecFile != "" {
			return nil, fmt.Errorf("template %s cannot have both command and execFile set", tc.Name)
		}
		if tc.Command == "" && tc.ExecFile == "" {
			return nil, fmt.Errorf("template %s needs to have either command or execFile", tc.Name)
		}

		filename := tc.ExecFile
		if filename == "" {
			// Create command file with that content if specified by 'command'
			f, err := os.CreateTemp(config.C.TmpDir, "command-")
			if err != nil {
				return nil, fmt.Errorf("creating command file: %w", err)
			}
			err = f.Chmod(0755)
			if err != nil {
				return nil, fmt.Errorf("changing file permission: %w", err)
			}
			_, err = f.WriteString(tc.Command)
			if err != nil {
				return nil, fmt.Errorf("writing command to file: %w", err)
			}
			err = f.Close()
			if err != nil {
				return nil, fmt.Errorf("closing command file: %w", err)
			}

			filename = f.Name()
		}
		templates[tc.Name] = filename
	}

	var err error
	cmd.cmds, err = compileCommands(templates, config.C.Commands, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("compiling root command: %w", err)
	}

	// Add intrinsic help command
	if _, ok := cmd.cmds["help"]; ok {
		return nil, fmt.Errorf("`help` command is an intrinsic command and cannot be overridden")
	}
	cmd.cmds["help"] = &HelpCommand{root: cmd}

	return cmd, nil
}

func compileCommands(templates map[string]string, cc []*config.CommandConfig, leadingMatcher []string, parentOperators []string) (map[string]domain.Command, error) {
	cmds := make(map[string]domain.Command)

	for _, ci := range cc {
		// Validate
		if ci.Name == "" {
			return nil, fmt.Errorf("command needs a name")
		}
		if _, ok := cmds[ci.Name]; ok {
			return nil, fmt.Errorf("command name %s conflict", ci.Name)
		}
		if ci.TemplateRef == "" && len(ci.SubCommands) == 0 {
			return nil, fmt.Errorf("no self command or sub-commands defined")
		}
		operators := ci.Operators // If the parent allows everyone, this command's configuration is used
		if len(parentOperators) > 0 {
			// Take intersection with parent operators config, if parent has set one
			if len(operators) == 0 {
				operators = parentOperators // This command allows everyone, just inherit the parent operators
			} else {
				operators = lo.Intersect(operators, parentOperators)
				// Ensure the intersection is not empty
				if len(operators) == 0 {
					return nil, fmt.Errorf(
						"there will be no operators for command %s! Make sure to write all operators to parent commands which have operators set",
						ci.Name)
				}
				// Display warning if the command's operator was narrowed from definition
				if len(operators) < len(ci.Operators) {
					slog.Warn(fmt.Sprintf(
						"Compiling command \"%s\": number of operators was narrowed from %d to %d. Make sure to write all operators to parent commands which have operators set.",
						strings.Join(append(utils.Copy(leadingMatcher), ci.Name), " "), len(ci.Operators), len(operators)))
				}
			}
		}

		// Create a command instance
		cmd := &CommandInstance{
			leadingMatcher: utils.Copy(leadingMatcher),
			name:           ci.Name,
			description:    ci.Description,
			allowArgs:      ci.AllowArgs,
			argsSyntax:     ci.ArgsSyntax,
			argsPrefix:     ci.ArgsPrefix,
			operators:      operators,
			subCommands:    make(map[string]domain.Command),
		}

		// Command (self)
		if ci.TemplateRef != "" {
			tmplFile, ok := templates[ci.TemplateRef]
			if !ok {
				return nil, fmt.Errorf("invalid template ref %s", ci.TemplateRef)
			}
			cmd.commandFile = tmplFile
		}

		// Sub-commands, if any
		var err error
		cmd.subCommands, err = compileCommands(templates, ci.SubCommands, append(leadingMatcher, ci.Name), operators)
		if err != nil {
			return nil, fmt.Errorf("compiling sub-commands of %s: %w", ci.Name, err)
		}

		cmds[ci.Name] = cmd
	}

	return cmds, nil
}

func (dc *RootCommand) Execute(ctx domain.Context) error {
	slog.Info("Executing command", "args", ctx.Args(), "executor", ctx.Executor())
	name := ctx.Args()[0]

	c, ok := dc.cmds[name]
	if !ok {
		return ctx.ReplyBad(fmt.Sprintf("Unrecognized command `%s`, try /help", name))
	}

	ctx = ctx.ShiftArgs() // Cut matching args
	return c.Execute(ctx)
}

func (dc *RootCommand) HasSubcommands() bool {
	return len(dc.cmds) > 0
}

func (dc *RootCommand) GetSubcommand(verb string) (domain.Command, bool) {
	c, ok := dc.cmds[verb]
	return c, ok
}

func (dc *RootCommand) getMatchingCommand(args []string) (domain.Command, bool) {
	cur, ok := dc.GetSubcommand(args[0])
	if !ok {
		return nil, false
	}
	for _, arg := range args[1:] {
		cur, ok = cur.GetSubcommand(arg)
		if !ok {
			return nil, false
		}
	}
	return cur, true
}

func (dc *RootCommand) HelpMessage(_ int, _ bool) []string {
	var lines []string
	names := lo.Keys(dc.cmds)
	slices.Sort(names)
	for _, name := range names {
		cmd := dc.cmds[name]
		lines = append(lines, cmd.HelpMessage(0, false)...)
	}
	return lines
}

func (c *CommandInstance) Execute(ctx domain.Context) error {
	// If this command has permitted operators defined, check operator
	if len(c.operators) > 0 {
		if !lo.Contains(c.operators, ctx.Executor()) {
			// User is not allowed to execute this command (or any subcommand)
			return ctx.ReplyForbid(fmt.Sprintf("You do not have permission to execute this command (`%s`).", c.matcher()))
		}
	}

	// Check if any sub-commands match
	if len(ctx.Args()) > 0 {
		subVerb := ctx.Args()[0]
		subCmd, ok := c.subCommands[subVerb]
		if ok {
			// A sub-command match
			ctx = ctx.ShiftArgs() // Cut matching args
			return subCmd.Execute(ctx)
		}

		if c.commandFile == "" {
			// Sub-commands do not match, and self-command is not defined
			return ctx.ReplyBad(fmt.Sprintf("Unrecognized sub-command `%s`, try `%shelp`", subVerb, config.C.Prefix))
		}
	}

	// Self-command is not defined - error
	if c.commandFile == "" {
		if len(c.subCommands) > 0 {
			// If this command has sub-commands, display help
			var lines []string
			lines = append(lines, fmt.Sprintf("## `%s` Usage", c.matcher()))
			lines = append(lines, "")
			lines = append(lines, c.HelpMessage(0, true)...)
			return ctx.ReplyBad(lines...)
		} else {
			// Otherwise, just error
			return ctx.ReplyBad(fmt.Sprintf("Command `%s` has no use, maybe the bot is badly configured?", c.matcher()))
		}
	}

	// Validate run command arguments (self)
	if !c.allowArgs && len(ctx.Args()) > 0 {
		return ctx.ReplyBad(fmt.Sprintf(
			"Command `%s` cannot have extra arguments (you supplied `%s`)\nTry setting allowArgs: true in config to allow extra arguments",
			c.matcher(),
			strings.Join(ctx.Args(), " "),
		))
	}

	// Run command (self)
	_ = ctx.ReplyRunning()

	var args []string
	args = append(args, c.argsPrefix...)
	if c.allowArgs {
		args = append(args, ctx.Args()...)
	}
	var buf bytes.Buffer
	cmd := exec.CommandContext(ctx, c.commandFile, args...)
	cmd.Stdout = &buf
	cmd.Stderr = &buf

	logLimit := ctx.MessageLimit() - 100 /* margin */

	err := cmd.Run()
	if err != nil {
		return ctx.ReplyFailure(
			fmt.Sprintf(":%s: exec failed: %v", ctx.StampNames().Failure, err),
			"```",
			utils.LimitLog(utils.SafeConvertString(buf.Bytes()), logLimit),
			"```",
		)
	}

	var replyMessage []string
	if buf.Len() > 0 {
		replyMessage = append(replyMessage, fmt.Sprintf(":%s:", ctx.StampNames().Success))
		replyMessage = append(replyMessage, "```")
		replyMessage = append(replyMessage, utils.LimitLog(utils.SafeConvertString(buf.Bytes()), logLimit))
		replyMessage = append(replyMessage, "```")
	} else {
		replyMessage = append(replyMessage, "*No output*")
	}
	return ctx.ReplySuccess(replyMessage...)
}

func (c *CommandInstance) HasSubcommands() bool {
	return len(c.subCommands) > 0
}

func (c *CommandInstance) GetSubcommand(verb string) (domain.Command, bool) {
	sub, ok := c.subCommands[verb]
	return sub, ok
}

func (c *CommandInstance) HelpMessage(indent int, formatSub bool) []string {
	var lines []string

	// Command (self) usage
	var operators string
	if config.C.Mode == "traq" {
		operators = strings.Join(
			lo.Map(c.operators, func(s string, _ int) string { return `:@` + s + `:` }),
			"",
		)
	} else {
		operators = fmt.Sprintf("%d operator%s", len(c.operators), lo.Ternary(len(c.operators) == 1, "", "s"))
	}
	if len(c.operators) == 0 {
		operators = "everyone"
	}

	var subCommandsNum string
	if len(c.subCommands) > 0 {
		subCommandsNum = fmt.Sprintf(", %d sub-command%s", len(c.subCommands), lo.Ternary(len(c.subCommands) == 1, "", "s"))
	}

	syntax := c.matcher()
	if c.argsSyntax != "" {
		syntax += " " + c.argsSyntax
	}

	lines = append(lines, fmt.Sprintf(
		"%s- `%s`%s (%s%s)",
		strings.Repeat(" ", indent),
		syntax,
		lo.Ternary(c.description != "", " - "+c.description, ""),
		operators,
		subCommandsNum,
	))

	// Sub-commands usage
	if formatSub {
		subVerbs := lo.Keys(c.subCommands)
		slices.Sort(subVerbs)
		for _, subVerb := range subVerbs {
			subCmd := c.subCommands[subVerb]
			lines = append(lines, subCmd.HelpMessage(indent+2, false)...)
		}
	}

	return lines
}

func (c *CommandInstance) matcher() string {
	return config.C.Prefix + strings.Join(append(c.leadingMatcher, c.name), " ")
}
