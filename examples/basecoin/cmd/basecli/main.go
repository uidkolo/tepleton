package main

import (
	"os"

	"github.com/tepleton/tepleton-sdk/client"
	"github.com/tepleton/tepleton-sdk/client/keys"
	"github.com/tepleton/tepleton-sdk/client/lcd"
	"github.com/tepleton/tepleton-sdk/client/rpc"
	"github.com/tepleton/tepleton-sdk/client/tx"
	"github.com/tepleton/tepleton-sdk/examples/basecoin/app"
	"github.com/tepleton/tepleton-sdk/examples/basecoin/types"
	"github.com/tepleton/tepleton-sdk/version"
	authcmd "github.com/tepleton/tepleton-sdk/x/auth/client/cli"
	bankcmd "github.com/tepleton/tepleton-sdk/x/bank/client/cli"
	ibccmd "github.com/tepleton/tepleton-sdk/x/ibc/client/cli"
	stakecmd "github.com/tepleton/tepleton-sdk/x/stake/client/cli"
	"github.com/spf13/cobra"
	"github.com/tepleton/tepleton/libs/cli"
)

// rootCmd is the entry point for this binary
var (
	rootCmd = &cobra.Command{
		Use:   "basecli",
		Short: "Basecoin light-client",
	}
)

func main() {
	// disable sorting
	cobra.EnableCommandSorting = false

	// get the codec
	cdc := app.MakeCodec()

	// TODO: Setup keybase, viper object, etc. to be passed into
	// the below functions and eliminate global vars, like we do
	// with the cdc.

	// add standard rpc, and tx commands
	rpc.AddCommands(rootCmd)
	rootCmd.AddCommand(client.LineBreak)
	tx.AddCommands(rootCmd, cdc)
	rootCmd.AddCommand(client.LineBreak)

	// add query/post commands (custom to binary)
	rootCmd.AddCommand(
		client.GetCommands(
			stakecmd.GetCmdQueryValidator("stake", cdc),
			stakecmd.GetCmdQueryValidators("stake", cdc),
			stakecmd.GetCmdQueryDelegation("stake", cdc),
			stakecmd.GetCmdQueryDelegations("stake", cdc),
			authcmd.GetAccountCmd("acc", cdc, types.GetAccountDecoder(cdc)),
		)...)

	rootCmd.AddCommand(
		client.PostCommands(
			bankcmd.SendTxCmd(cdc),
			ibccmd.IBCTransferCmd(cdc),
			ibccmd.IBCRelayCmd(cdc),
			stakecmd.GetCmdCreateValidator(cdc),
			stakecmd.GetCmdEditValidator(cdc),
			stakecmd.GetCmdDelegate(cdc),
			stakecmd.GetCmdUnbond("stake", cdc),
		)...)

	// add proxy, version and key info
	rootCmd.AddCommand(
		client.LineBreak,
		lcd.ServeCommand(cdc),
		keys.Commands(),
		client.LineBreak,
		version.VersionCmd,
	)

	// prepare and add flags
	executor := cli.PrepareMainCmd(rootCmd, "BC", os.ExpandEnv("$HOME/.basecli"))
	err := executor.Execute()
	if err != nil {
		// Note: Handle with #870
		panic(err)
	}
}
