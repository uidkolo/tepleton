package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/tepleton/tmlibs/cli"

	"github.com/tepleton/tepleton-sdk/baseapp"
	"github.com/tepleton/tepleton-sdk/server"
	"github.com/tepleton/tepleton-sdk/version"
)

// tondCmd is the entry point for this binary
var (
	tondCmd = &cobra.Command{
		Use:   "tond",
		Short: "Gaia Daemon (server)",
	}
)

// defaultOptions sets up the app_options for the
// default genesis file
func defaultOptions(args []string) (json.RawMessage, error) {
	addr, secret, err := server.GenerateCoinKey()
	if err != nil {
		return nil, err
	}
	fmt.Println("Secret phrase to access coins:")
	fmt.Println(secret)

	opts := fmt.Sprintf(`{
      "accounts": [{
        "address": "%s",
        "coins": [
          {
            "denom": "mycoin",
            "amount": 9007199254740992
          }
        ]
      }]
    }`, addr)
	return json.RawMessage(opts), nil
}

func main() {
	// TODO: set this to something real
	var app *baseapp.BaseApp

	tondCmd.AddCommand(
		server.InitCmd(defaultOptions),
		server.StartCmd(app, app.Logger),
		server.UnsafeResetAllCmd(app.Logger),
		version.VersionCmd,
	)

	// prepare and add flags
	executor := cli.PrepareBaseCmd(tondCmd, "GA", os.ExpandEnv("$HOME/.tond"))
	executor.Execute()
}
