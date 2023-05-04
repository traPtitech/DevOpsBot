package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/samber/lo"
)

type DeployCommand struct {
	instances map[string]*DeployCommandInstance
}

type DeployCommandInstance struct {
	commandFile string
	description string
	argsPrefix  []string
	operators   []string
}

func (dc *DeployConfig) Compile() (*DeployCommand, error) {
	cmd := &DeployCommand{
		instances: make(map[string]*DeployCommandInstance),
	}

	templates := make(map[string]string, len(dc.Templates)) // name to filename
	for _, tc := range dc.Templates {
		if tc.Name == "" {
			return nil, fmt.Errorf("template needs to have a name")
		}
		if _, ok := templates[tc.Name]; ok {
			return nil, fmt.Errorf("template %s conflict", tc.Name)
		}
		if tc.Command != "" && tc.CommandFile != "" {
			return nil, fmt.Errorf("template %s cannot have both command and commandFile set", tc.Name)
		}
		if tc.Command == "" && tc.CommandFile == "" {
			return nil, fmt.Errorf("template %s needs to have either command or commandFile", tc.Name)
		}

		filename := tc.CommandFile
		if filename == "" {
			f, err := os.CreateTemp("", "command-")
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

			filename = filepath.Join(os.TempDir(), f.Name())
		}
		templates[tc.Name] = filename
	}

	for _, cc := range dc.Commands {
		tmplFile, ok := templates[cc.TemplateRef]
		if !ok {
			return nil, fmt.Errorf("invalid template ref %s", cc.TemplateRef)
		}

		if _, ok = cmd.instances[cc.Name]; ok {
			return nil, fmt.Errorf("command name %s conflict", cc.Name)
		}
		cmd.instances[cc.Name] = &DeployCommandInstance{
			commandFile: tmplFile,
			description: cc.Description,
			argsPrefix:  cc.ArgsPrefix,
			operators:   cc.Operators,
		}
	}

	return cmd, nil
}

func (dc *DeployCommand) Execute(ctx *Context) error {
	// ctx.Args = deploy [name] [args...]

	if len(ctx.Args) <= 1 {
		return ctx.Reply(dc.MakeHelpMessage()...)
	}

	name := ctx.Args[1]
	c, ok := dc.instances[name]
	if !ok {
		return ctx.ReplyBad(fmt.Sprintf("unrecognized deploy subcommand %s", name))
	}

	// Check operator
	if len(c.operators) > 0 {
		if !lo.Contains(c.operators, ctx.GetExecutor()) {
			return ctx.ReplyForbid()
		}
	}

	// Run
	var args []string
	args = append(args, c.argsPrefix...)
	args = append(args, ctx.Args[2:]...)
	var buf bytes.Buffer
	cmd := exec.CommandContext(ctx, c.commandFile, args...)
	cmd.Stdout = &buf
	cmd.Stderr = &buf

	err := cmd.Run()
	if err != nil {
		return ctx.ReplyFailure(
			fmt.Sprintf("exec failed: %v", err),
			"```",
			safeConvertString(buf.Bytes()),
			"```",
		)
	}

	return ctx.ReplySuccess(
		"```",
		safeConvertString(buf.Bytes()),
		"```",
	)
}

func (dc *DeployCommand) MakeHelpMessage() []string {
	var lines []string
	lines = append(lines, "## deploy commands")
	for name, cmd := range dc.instances {
		lines = append(lines, fmt.Sprintf(
			"- `%sdeploy %s`%s",
			config.Prefix,
			name,
			lo.Ternary(cmd.description != "", " - "+cmd.description, ""),
		))
		if len(cmd.operators) > 0 {
			lines = append(lines, fmt.Sprintf("  - operators: %s", strings.Join(cmd.operators, ", ")))
		}
	}
	return lines
}
