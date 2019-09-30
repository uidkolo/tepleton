package app

import (
	"fmt"
	"os"

	bam "github.com/tepleton/tepleton-sdk/baseapp"
	sdk "github.com/tepleton/tepleton-sdk/types"
	"github.com/tepleton/wrsp/server"
	"github.com/tepleton/go-wire"
	cmn "github.com/tepleton/tmlibs/common"
)

const appName = "BasecoinApp"

type BasecoinApp struct {
	*bam.BaseApp
	cdc        *wire.Codec
	multiStore sdk.CommitMultiStore

	// The key to access the substores.
	capKeyMainStore *sdk.KVStoreKey
	capKeyIBCStore  *sdk.KVStoreKey

	// Object mappers :
	accountMapper sdk.AccountMapper
}

// TODO: This should take in more configuration options.
func NewBasecoinApp() *BasecoinApp {

	// Create and configure app.
	var app = &BasecoinApp{}
	app.initCapKeys() // ./init_capkeys.go
	app.initBaseApp() // ./init_baseapp.go
	app.initStores()  // ./init_stores.go
	app.initRoutes()  // ./init_routes.go

	// TODO: Load genesis
	// TODO: InitChain with validators
	// TODO: Set the genesis accounts

	app.loadStores()

	return app
}

func (app *BasecoinApp) RunForever() {

	// Start the WRSP server
	srv, err := server.NewServer("0.0.0.0:46658", "socket", app)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	srv.Start()

	// Wait forever
	cmn.TrapSignal(func() {
		// Cleanup
		srv.Stop()
	})

}

// Load the stores.
func (app *BasecoinApp) loadStores() {
	if err := app.LoadLatestVersion(app.capKeyMainStore); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
