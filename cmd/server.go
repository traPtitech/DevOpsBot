package cmd

import (
	"github.com/spf13/cobra"

	"github.com/traPtitech/DevOpsBot/pkg/config"
	"github.com/traPtitech/DevOpsBot/pkg/server"
)

var serverCmd = &cobra.Command{
	Use:          "server",
	Short:        "ConoHa server manipulation",
	SilenceUsage: true, // Do not display command usage when RunE returns error
	RunE: func(cmd *cobra.Command, args []string) error {
		s, err := server.Compile(&config.C.Commands.Servers)
		if err != nil {
			return err
		}
		return s.Execute(args)
	},
}
