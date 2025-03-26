// Package server provides ConoHa server related operations
package server

import (
	"errors"
	"fmt"
)

type ServersCommand struct {
	sub *subCommand
}

func Compile() (*ServersCommand, error) {
	cmd := &ServersCommand{}

	s := &subCommand{
		Commands: make(map[string]command),
	}
	s.Commands["restart"] = &restartCommand{s}
	cmd.sub = s

	return cmd, nil
}

func (sc *ServersCommand) Execute(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("invalid arguments, expected server id")
	}

	// args == [server id] restart [SOFT|HARD]
	serverID := args[0]
	return sc.sub.Execute(serverID, args[1:])
}

type subCommand struct {
	Commands map[string]command
}

func (i *subCommand) Execute(serverID string, args []string) error {
	if len(args) < 1 {
		return errors.New("invalid arguments, expected server action verb (supported: restart)")
	}

	// args == restart [SOFT|HARD]
	verb := args[0]
	c, ok := i.Commands[verb]
	if !ok {
		return fmt.Errorf("unknown command: `%s`", verb)
	}
	return c.Execute(serverID, args[1:])
}

type command interface {
	Execute(serverID string, args []string) error
}
