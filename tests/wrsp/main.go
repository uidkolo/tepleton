package main

import (
	"fmt"

	"github.com/tepleton/basecoin/app"
	"github.com/tepleton/basecoin/tests"
	"github.com/tepleton/basecoin/types"
	. "github.com/tepleton/go-common"
	"github.com/tepleton/go-wire"
	eyescli "github.com/tepleton/merkleeyes/client"
)

func main() {
	testSendTx()
	testSequence()
}

func testSendTx() {
	eyesCli := eyescli.NewLocalClient()
	chainID := "test_chain_id"
	bcApp := app.NewBasecoin(eyesCli)
	bcApp.SetOption("base/chainID", chainID)
	fmt.Println(bcApp.Info())

	test1PrivAcc := tests.PrivAccountFromSecret("test1")
	test2PrivAcc := tests.PrivAccountFromSecret("test2")

	// Seed Basecoin with account
	test1Acc := test1PrivAcc.Account
	test1Acc.Balance = types.Coins{{"", 1000}}
	fmt.Println(bcApp.SetOption("base/account", string(wire.JSONBytes(test1Acc))))

	res := bcApp.Commit()
	if res.IsErr() {
		Exit(Fmt("Failed Commit: %v", res.Error()))
	}

	// Construct a SendTx signature
	tx := &types.SendTx{
		Fee: 0,
		Gas: 0,
		Inputs: []types.TxInput{
			types.TxInput{
				Address:  test1PrivAcc.Account.PubKey.Address(),
				PubKey:   test1PrivAcc.Account.PubKey, // TODO is this needed?
				Coins:    types.Coins{{"", 1}},
				Sequence: 1,
			},
		},
		Outputs: []types.TxOutput{
			types.TxOutput{
				Address: test2PrivAcc.Account.PubKey.Address(),
				Coins:   types.Coins{{"", 1}},
			},
		},
	}

	// Sign request
	signBytes := tx.SignBytes(chainID)
	fmt.Printf("Sign bytes: %X\n", signBytes)
	sig := test1PrivAcc.PrivKey.Sign(signBytes)
	tx.Inputs[0].Signature = sig
	//fmt.Println("tx:", tx)
	fmt.Printf("Signed TX bytes: %X\n", wire.BinaryBytes(struct{ types.Tx }{tx}))

	// Write request
	txBytes := wire.BinaryBytes(struct{ types.Tx }{tx})
	res = bcApp.AppendTx(txBytes)
	fmt.Println(res)
	if res.IsErr() {
		Exit(Fmt("Failed: %v", res.Error()))
	}
}

func testSequence() {
	eyesCli := eyescli.NewLocalClient()
	chainID := "test_chain_id"
	bcApp := app.NewBasecoin(eyesCli)
	bcApp.SetOption("base/chainID", chainID)
	fmt.Println(bcApp.Info())

	// Get the test account
	test1PrivAcc := tests.PrivAccountFromSecret("test1")
	test1Acc := test1PrivAcc.Account
	test1Acc.Balance = types.Coins{{"", 1 << 53}}
	fmt.Println(bcApp.SetOption("base/account", string(wire.JSONBytes(test1Acc))))

	res := bcApp.Commit()
	if res.IsErr() {
		Exit(Fmt("Failed Commit: %v", res.Error()))
	}

	sequence := int(1)
	// Make a bunch of PrivAccounts
	privAccounts := tests.RandAccounts(1000, 1000000, 0)
	privAccountSequences := make(map[string]int)
	// Send coins to each account

	for i := 0; i < len(privAccounts); i++ {
		privAccount := privAccounts[i]

		//Generate txInputs with or without public key
		tempTxInputs := types.TxInput{
			Address:  test1Acc.PubKey.Address(),
			PubKey:   test1Acc.PubKey, // TODO is this needed?
			Coins:    types.Coins{{"", 1000002}},
			Sequence: sequence,
		}

		if sequence > 1 {
			tempTxInputs = types.TxInput{
				Address:  test1Acc.PubKey.Address(),
				Coins:    types.Coins{{"", 1000002}},
				Sequence: sequence,
			}
		}

		tx := &types.SendTx{
			Fee:    2,
			Gas:    2,
			Inputs: []types.TxInput{tempTxInputs},
			Outputs: []types.TxOutput{
				types.TxOutput{
					Address: privAccount.Account.PubKey.Address(),
					Coins:   types.Coins{{"", 1000000}},
				},
			},
		}
		sequence += 1

		// Sign request
		signBytes := tx.SignBytes(chainID)
		sig := test1PrivAcc.PrivKey.Sign(signBytes)
		tx.Inputs[0].Signature = sig
		// fmt.Printf("ADDR: %X -> %X\n", tx.Inputs[0].Address, tx.Outputs[0].Address)

		// Write request
		txBytes := wire.BinaryBytes(struct{ types.Tx }{tx})
		res := bcApp.AppendTx(txBytes)
		if res.IsErr() {
			Exit("AppendTx error: " + res.Error())
		}

	}

	fmt.Println("-------------------- RANDOM SENDS --------------------")

	res = bcApp.Commit()
	if res.IsErr() {
		Exit(Fmt("Failed Commit: %v", res.Error()))
	}

	// Now send coins between these accounts
	for i := 0; i < 10000; i++ {
		randA := RandInt() % len(privAccounts)
		randB := RandInt() % len(privAccounts)
		if randA == randB {
			continue
		}

		privAccountA := privAccounts[randA]
		privAccountASequence := privAccountSequences[privAccountA.Account.PubKey.KeyString()]
		privAccountSequences[privAccountA.Account.PubKey.KeyString()] = privAccountASequence + 1
		privAccountB := privAccounts[randB]

		tx := &types.SendTx{
			Fee: 2,
			Gas: 2,
			Inputs: []types.TxInput{
				types.TxInput{
					Address:  privAccountA.Account.PubKey.Address(),
					PubKey:   privAccountA.Account.PubKey,
					Coins:    types.Coins{{"", 3}},
					Sequence: privAccountASequence + 1,
				},
			},
			Outputs: []types.TxOutput{
				types.TxOutput{
					Address: privAccountB.Account.PubKey.Address(),
					Coins:   types.Coins{{"", 1}},
				},
			},
		}

		// Sign request
		signBytes := tx.SignBytes(chainID)
		sig := privAccountA.PrivKey.Sign(signBytes)
		tx.Inputs[0].Signature = sig
		// fmt.Printf("ADDR: %X -> %X\n", tx.Inputs[0].Address, tx.Outputs[0].Address)

		// Write request
		txBytes := wire.BinaryBytes(struct{ types.Tx }{tx})
		res := bcApp.AppendTx(txBytes)
		if res.IsErr() {
			Exit("AppendTx error: " + res.Error())
		}
	}
}
