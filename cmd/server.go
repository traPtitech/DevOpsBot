package cmd

import "github.com/spf13/cobra"

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "ConoHa server manipulation",
	RunE: func(cmd *cobra.Command, args []string) error {
		panic("implement me")
	},
}
