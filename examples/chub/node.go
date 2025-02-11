package main

import (
	"github.com/spf13/cobra"

	"github.com/tepleton/tepleton-sdk/baseapp"
)

const (
	flagWithTendermint = "with-tepleton"
)

var (
	initNodeCmd = &cobra.Command{
		Use:   "init <flags???>",
		Short: "Initialize full node",
		RunE:  todoNotImplemented,
	}

	resetNodeCmd = &cobra.Command{
		Use:   "unsafe_reset_all",
		Short: "Reset full node data (danger, must resync)",
		RunE:  todoNotImplemented,
	}
)

// NodeCommands registers a sub-tree of commands to interact with
// a local full-node.
//
// Accept an application it should start
func NodeCommands(node baseapp.BaseApp) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "node",
		Short: "Run the full node",
	}
	cmd.AddCommand(
		initNodeCmd,
		startNodeCmd(node),
		resetNodeCmd,
	)
	return cmd
}

func startNodeCmd(node baseapp.BaseApp) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "start",
		Short: "Run the full node",
		RunE:  todoNotImplemented,
	}
	cmd.Flags().Bool(flagWithTendermint, true, "run wrsp app embedded in-process with tepleton")
	return cmd
}
