package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/traPtitech/DevOpsBot/pkg/bot"
	"github.com/traPtitech/DevOpsBot/pkg/utils"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "DevOpsBot",
	Short: "A traQ bot for DevOps command execution",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		fmt.Printf("DevOpsBot v%s initializing\n", utils.Version())
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		return bot.Run()
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(serverCmd)

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	// rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.DevOpsBot.yaml)")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	// rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
