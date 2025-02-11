package app

import (
	"encoding/json"

	wrsp "github.com/tepleton/tepleton/wrsp/types"
	cmn "github.com/tepleton/tepleton/libs/common"
	dbm "github.com/tepleton/tepleton/libs/db"
	"github.com/tepleton/tepleton/libs/log"
	tmtypes "github.com/tepleton/tepleton/types"

	bam "github.com/tepleton/tepleton-sdk/baseapp"
	sdk "github.com/tepleton/tepleton-sdk/types"
	"github.com/tepleton/tepleton-sdk/wire"
	"github.com/tepleton/tepleton-sdk/x/auth"
	"github.com/tepleton/tepleton-sdk/x/bank"
	"github.com/tepleton/tepleton-sdk/x/ibc"

	"github.com/tepleton/tepleton-sdk/examples/democoin/types"
	"github.com/tepleton/tepleton-sdk/examples/democoin/x/cool"
	"github.com/tepleton/tepleton-sdk/examples/democoin/x/pow"
	"github.com/tepleton/tepleton-sdk/examples/democoin/x/simplestake"
	"github.com/tepleton/tepleton-sdk/examples/democoin/x/sketchy"
)

const (
	appName = "DemocoinApp"
)

// Extended WRSP application
type DemocoinApp struct {
	*bam.BaseApp
	cdc *wire.Codec

	// keys to access the substores
	capKeyMainStore    *sdk.KVStoreKey
	capKeyAccountStore *sdk.KVStoreKey
	capKeyPowStore     *sdk.KVStoreKey
	capKeyIBCStore     *sdk.KVStoreKey
	capKeyStakingStore *sdk.KVStoreKey

	// keepers
	feeCollectionKeeper auth.FeeCollectionKeeper
	coinKeeper          bank.Keeper
	coolKeeper          cool.Keeper
	powKeeper           pow.Keeper
	ibcMapper           ibc.Mapper
	stakeKeeper         simplestake.Keeper

	// Manage getting and setting accounts
	accountMapper auth.AccountMapper
}

func NewDemocoinApp(logger log.Logger, db dbm.DB) *DemocoinApp {

	// Create app-level codec for txs and accounts.
	var cdc = MakeCodec()

	// Create your application object.
	var app = &DemocoinApp{
		BaseApp:            bam.NewBaseApp(appName, cdc, logger, db),
		cdc:                cdc,
		capKeyMainStore:    sdk.NewKVStoreKey("main"),
		capKeyAccountStore: sdk.NewKVStoreKey("acc"),
		capKeyPowStore:     sdk.NewKVStoreKey("pow"),
		capKeyIBCStore:     sdk.NewKVStoreKey("ibc"),
		capKeyStakingStore: sdk.NewKVStoreKey("stake"),
	}

	// Define the accountMapper.
	app.accountMapper = auth.NewAccountMapper(
		cdc,
		app.capKeyAccountStore, // target store
		&types.AppAccount{},    // prototype
	)

	// Add handlers.
	app.coinKeeper = bank.NewKeeper(app.accountMapper)
	app.coolKeeper = cool.NewKeeper(app.capKeyMainStore, app.coinKeeper, app.RegisterCodespace(cool.DefaultCodespace))
	app.powKeeper = pow.NewKeeper(app.capKeyPowStore, pow.NewConfig("pow", int64(1)), app.coinKeeper, app.RegisterCodespace(pow.DefaultCodespace))
	app.ibcMapper = ibc.NewMapper(app.cdc, app.capKeyIBCStore, app.RegisterCodespace(ibc.DefaultCodespace))
	app.stakeKeeper = simplestake.NewKeeper(app.capKeyStakingStore, app.coinKeeper, app.RegisterCodespace(simplestake.DefaultCodespace))
	app.Router().
		AddRoute("bank", bank.NewHandler(app.coinKeeper)).
		AddRoute("cool", cool.NewHandler(app.coolKeeper)).
		AddRoute("pow", app.powKeeper.Handler).
		AddRoute("sketchy", sketchy.NewHandler()).
		AddRoute("ibc", ibc.NewHandler(app.ibcMapper, app.coinKeeper)).
		AddRoute("simplestake", simplestake.NewHandler(app.stakeKeeper))

	// Initialize BaseApp.
	app.SetInitChainer(app.initChainerFn(app.coolKeeper, app.powKeeper))
	app.MountStoresIAVL(app.capKeyMainStore, app.capKeyAccountStore, app.capKeyPowStore, app.capKeyIBCStore, app.capKeyStakingStore)
	app.SetAnteHandler(auth.NewAnteHandler(app.accountMapper, app.feeCollectionKeeper))
	err := app.LoadLatestVersion(app.capKeyMainStore)
	if err != nil {
		cmn.Exit(err.Error())
	}
	return app
}

// custom tx codec
func MakeCodec() *wire.Codec {
	var cdc = wire.NewCodec()
	wire.RegisterCrypto(cdc) // Register crypto.
	sdk.RegisterWire(cdc)    // Register Msgs
	cool.RegisterWire(cdc)
	pow.RegisterWire(cdc)
	bank.RegisterWire(cdc)
	ibc.RegisterWire(cdc)
	simplestake.RegisterWire(cdc)

	// Register AppAccount
	cdc.RegisterInterface((*auth.Account)(nil), nil)
	cdc.RegisterConcrete(&types.AppAccount{}, "democoin/Account", nil)
	return cdc
}

// custom logic for democoin initialization
// nolint: unparam
func (app *DemocoinApp) initChainerFn(coolKeeper cool.Keeper, powKeeper pow.Keeper) sdk.InitChainer {
	return func(ctx sdk.Context, req wrsp.RequestInitChain) wrsp.ResponseInitChain {
		stateJSON := req.AppStateBytes

		genesisState := new(types.GenesisState)
		err := app.cdc.UnmarshalJSON(stateJSON, genesisState)
		if err != nil {
			panic(err) // TODO https://github.com/tepleton/tepleton-sdk/issues/468
			// return sdk.ErrGenesisParse("").TraceCause(err, "")
		}

		for _, gacc := range genesisState.Accounts {
			acc, err := gacc.ToAppAccount()
			if err != nil {
				panic(err) // TODO https://github.com/tepleton/tepleton-sdk/issues/468
				//	return sdk.ErrGenesisParse("").TraceCause(err, "")
			}
			app.accountMapper.SetAccount(ctx, acc)
		}

		// Application specific genesis handling
		err = cool.InitGenesis(ctx, app.coolKeeper, genesisState.CoolGenesis)
		if err != nil {
			panic(err) // TODO https://github.com/tepleton/tepleton-sdk/issues/468
			//	return sdk.ErrGenesisParse("").TraceCause(err, "")
		}

		err = pow.InitGenesis(ctx, app.powKeeper, genesisState.POWGenesis)
		if err != nil {
			panic(err) // TODO https://github.com/tepleton/tepleton-sdk/issues/468
			//	return sdk.ErrGenesisParse("").TraceCause(err, "")
		}

		return wrsp.ResponseInitChain{}
	}
}

// Custom logic for state export
func (app *DemocoinApp) ExportAppStateAndValidators() (appState json.RawMessage, validators []tmtypes.GenesisValidator, err error) {
	ctx := app.NewContext(true, wrsp.Header{})

	// iterate to get the accounts
	accounts := []*types.GenesisAccount{}
	appendAccount := func(acc auth.Account) (stop bool) {
		account := &types.GenesisAccount{
			Address: acc.GetAddress(),
			Coins:   acc.GetCoins(),
		}
		accounts = append(accounts, account)
		return false
	}
	app.accountMapper.IterateAccounts(ctx, appendAccount)

	genState := types.GenesisState{
		Accounts:    accounts,
		POWGenesis:  pow.WriteGenesis(ctx, app.powKeeper),
		CoolGenesis: cool.WriteGenesis(ctx, app.coolKeeper),
	}
	appState, err = wire.MarshalJSONIndent(app.cdc, genState)
	if err != nil {
		return nil, nil, err
	}
	return appState, validators, nil
}
