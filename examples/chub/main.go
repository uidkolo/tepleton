package main

import (
	"errors"
	"os"

	"github.com/spf13/cobra"

	"github.com/tepleton/tmlibs/cli"

	"github.com/tepleton/tepleton-sdk/app"
)

// chubCmd is the entry point for this binary
var (
	chubCmd = &cobra.Command{
		Use:   "chub",
		Short: "tepleton Hub command-line tool",
		Run:   help,
	}

	lineBreak = &cobra.Command{Run: func(*cobra.Command, []string) {}}
)

func todoNotImplemented(_ *cobra.Command, _ []string) error {
	return errors.New("TODO: Command not yet implemented")
}

func help(cmd *cobra.Command, args []string) {
	cmd.Help()
}

func main() {
	// disable sorting
	cobra.EnableCommandSorting = false

	// TODO: set this to something real
	var node app.App

	// add commands
	// prepareClientCommands()

	chubCmd.AddCommand(
		nodeCommand(node),
		keyCommand(),
		// clientCmd,

		lineBreak,
		versionCmd,
	)

	// prepare and add flags
	// executor := cli.PrepareMainCmd(chubCmd, "CH", os.ExpandEnv("$HOME/.tepleton-chub"))
	executor := cli.PrepareBaseCmd(chubCmd, "CH", os.ExpandEnv("$HOME/.tepleton-chub"))
	executor.Execute()
}
