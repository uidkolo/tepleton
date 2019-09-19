package main

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/tepleton/tmlibs/cli"

	client "github.com/tepleton/basecoin/client/commands"
	"github.com/tepleton/basecoin/cmd/basecoin/commands"
	"github.com/tepleton/basecoin/docs/guide/counter/plugins/counter"
)

func main() {
	var RootCmd = &cobra.Command{
		Use:   "counter",
		Short: "demo plugin for basecoin",
	}

	// TODO: register the counter here
	commands.Handler = counter.NewHandler("mycoin")

	RootCmd.AddCommand(
		commands.InitCmd,
		commands.StartCmd,
		commands.UnsafeResetAllCmd,
		client.VersionCmd,
	)

	cmd := cli.PrepareMainCmd(RootCmd, "CT", os.ExpandEnv("$HOME/.counter"))
	cmd.Execute()
}
