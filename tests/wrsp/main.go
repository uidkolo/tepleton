package main

import (
	"fmt"

	"github.com/tepleton/basecoin/app"
	"github.com/tepleton/basecoin/tests"
	"github.com/tepleton/basecoin/types"
	. "github.com/tepleton/go-common"
	"github.com/tepleton/go-wire"
	govtypes "github.com/tepleton/governmint/types"
	eyescli "github.com/tepleton/merkleeyes/client"
)

func main() {
	testSendTx()
	testGov()
}

func testSendTx() {
	eyesCli := eyescli.NewLocalClient()
	bcApp := app.NewBasecoin(eyesCli)
	fmt.Println(bcApp.Info())

	tPriv := tests.PrivAccountFromSecret("test")
	tPriv2 := tests.PrivAccountFromSecret("test2")

	// Seed Basecoin with account
	tAcc := tPriv.Account
	tAcc.Balance = 1000
	fmt.Println(bcApp.SetOption("base/chainID", "test_chain_id"))
	fmt.Println(bcApp.SetOption("base/account", string(wire.JSONBytes(tAcc))))

	// Construct a SendTx signature
	tx := &types.SendTx{
		Inputs: []types.TxInput{
			types.TxInput{
				Address:  tPriv.Account.PubKey.Address(),
				PubKey:   tPriv.Account.PubKey, // TODO is this needed?
				Amount:   1,
				Sequence: 1,
			},
		},
		Outputs: []types.TxOutput{
			types.TxOutput{
				Address: tPriv2.Account.PubKey.Address(),
				Amount:  1,
			},
		},
	}

	// Sign request
	signBytes := tx.SignBytes("test_chain_id")
	fmt.Printf("Sign bytes: %X\n", signBytes)
	sig := tPriv.PrivKey.Sign(signBytes)
	tx.Inputs[0].Signature = sig
	//fmt.Println("tx:", tx)
	fmt.Printf("Signed TX bytes: %X\n", wire.BinaryBytes(tx))

	// Write request
	txBytes := wire.BinaryBytes(tx)
	res := bcApp.AppendTx(txBytes)
	fmt.Println(res)
	if res.IsErr() {
		Exit(Fmt("Failed: %v", res.Error()))
	}
}

func testGov() {
	eyesCli := eyescli.NewLocalClient()
	bcApp := app.NewBasecoin(eyesCli)
	fmt.Println(bcApp.Info())

	tPriv := tests.PrivAccountFromSecret("test")

	// Seed Basecoin with admin using PrivAccount
	tAcc := tPriv.Account
	adminEntity := govtypes.Entity{
		ID:     "",
		PubKey: tAcc.PubKey,
	}
	log := bcApp.SetOption("gov/admin", string(wire.JSONBytes(adminEntity)))
	if log != "Success" {
		Exit(Fmt("Failed to set option: %v", log))
	}
	// TODO test proposals or something.
}
