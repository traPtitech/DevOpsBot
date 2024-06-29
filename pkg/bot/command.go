package bot

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"slices"
	"strings"

	"github.com/samber/lo"

	"github.com/traPtitech/DevOpsBot/pkg/config"
	"github.com/traPtitech/DevOpsBot/pkg/utils"
)

// command コマンドインターフェース
type command interface {
	execute(ctx Context) error
	helpMessage(indent int) []string
}

var (
	_ command = (*RootCommand)(nil)
	_ command = (*CommandInstance)(nil)
)

type RootCommand struct {
	cmds map[string]command
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
	subCommands map[string]command
}

func Compile() (*RootCommand, error) {
	cmd := &RootCommand{
		cmds: make(map[string]command),
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
	cmd.cmds, err = compileCommands(templates, config.C.Commands, nil)
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

func compileCommands(templates map[string]string, cc []*config.CommandConfig, leadingMatcher []string) (map[string]command, error) {
	cmds := make(map[string]command)

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

		// Create a command instance
		cmd := &CommandInstance{
			leadingMatcher: utils.Copy(leadingMatcher),
			name:           ci.Name,
			description:    ci.Description,
			allowArgs:      ci.AllowArgs,
			argsSyntax:     ci.ArgsSyntax,
			argsPrefix:     ci.ArgsPrefix,
			operators:      ci.Operators,
			subCommands:    make(map[string]command),
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
		cmd.subCommands, err = compileCommands(templates, ci.SubCommands, append(leadingMatcher, ci.Name))
		if err != nil {
			return nil, fmt.Errorf("compiling sub-commands of %s: %w", ci.Name, err)
		}

		cmds[ci.Name] = cmd
	}

	return cmds, nil
}

func (dc *RootCommand) execute(ctx Context) error {
	name := ctx.Args()[0]

	c, ok := dc.cmds[name]
	if !ok {
		return ctx.ReplyBad(fmt.Sprintf("Unrecognized command `%s`, try /help", name))
	}

	ctx = ctx.ShiftArgs() // Cut matching args
	return c.execute(ctx)
}

func (c *CommandInstance) execute(ctx Context) error {
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
			return subCmd.execute(ctx)
		}

		if c.commandFile == "" {
			// Sub-commands do not match, and self-command is not defined
			return ctx.ReplyBad(fmt.Sprintf("Unrecognized sub-command `%s`, try /help", subVerb))
		}
	}

	// Self-command is not defined - error
	if c.commandFile == "" {
		if len(c.subCommands) > 0 {
			// If this command has sub-commands, display help
			var lines []string
			lines = append(lines, fmt.Sprintf("## `%s` Usage", c.matcher()))
			lines = append(lines, c.helpMessage(0)...)
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

	const logLimit = 9900

	err := cmd.Run()
	if err != nil {
		return ctx.ReplyFailure(
			fmt.Sprintf("exec failed: %v", err),
			"```",
			utils.LimitLog(utils.SafeConvertString(buf.Bytes()), logLimit),
			"```",
		)
	}

	return ctx.ReplySuccess(
		"```",
		utils.LimitLog(utils.SafeConvertString(buf.Bytes()), logLimit),
		"```",
	)
}

func (dc *RootCommand) helpMessage(_ int) []string {
	var lines []string
	names := lo.Keys(dc.cmds)
	slices.Sort(names)
	for _, name := range names {
		cmd := dc.cmds[name]
		lines = append(lines, cmd.helpMessage(0)...)
	}
	return lines
}

func (c *CommandInstance) helpMessage(indent int) []string {
	var lines []string

	// Command (self) usage
	operators := strings.Join(
		lo.Map(c.operators, func(s string, _ int) string { return `:@` + s + `:` }),
		"",
	)
	if operators == "" {
		operators = "everyone"
	}

	syntax := c.matcher()
	if c.argsSyntax != "" {
		syntax += " " + c.argsSyntax
	}

	lines = append(lines, fmt.Sprintf(
		"%s- `%s`%s (%s)",
		strings.Repeat(" ", indent),
		syntax,
		lo.Ternary(c.description != "", " - "+c.description, ""),
		operators,
	))

	// Sub-commands usage
	subVerbs := lo.Keys(c.subCommands)
	slices.Sort(subVerbs)
	for _, subVerb := range subVerbs {
		subCmd := c.subCommands[subVerb]
		lines = append(lines, subCmd.helpMessage(indent+2)...)
	}

	return lines
}

func (c *CommandInstance) matcher() string {
	return config.C.Prefix + strings.Join(append(c.leadingMatcher, c.name), " ")
}
