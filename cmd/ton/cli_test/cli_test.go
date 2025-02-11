package clitest

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/tepleton/tepleton/crypto"
	cmn "github.com/tepleton/tepleton/libs/common"

	"github.com/tepleton/tepleton-sdk/client/keys"
	"github.com/tepleton/tepleton-sdk/cmd/ton/app"
	"github.com/tepleton/tepleton-sdk/server"
	"github.com/tepleton/tepleton-sdk/tests"
	sdk "github.com/tepleton/tepleton-sdk/types"
	"github.com/tepleton/tepleton-sdk/wire"
	"github.com/tepleton/tepleton-sdk/x/auth"
	"github.com/tepleton/tepleton-sdk/x/gov"
	"github.com/tepleton/tepleton-sdk/x/stake"
)

func TestGaiaCLISend(t *testing.T) {

	tests.ExecuteT(t, "tond unsafe_reset_all")
	pass := "1234567890"
	executeWrite(t, "toncli keys delete foo", pass)
	executeWrite(t, "toncli keys delete bar", pass)
	chainID := executeInit(t, "tond init -o --name=foo")
	executeWrite(t, "toncli keys add bar", pass)

	// get a free port, also setup some common flags
	servAddr, port, err := server.FreeTCPAddr()
	require.NoError(t, err)
	flags := fmt.Sprintf("--node=%v --chain-id=%v", servAddr, chainID)

	// start tond server
	proc := tests.GoExecuteTWithStdout(t, fmt.Sprintf("tond start --rpc.laddr=%v", servAddr))
	defer proc.Stop(false)
	tests.WaitForTMStart(port)
	tests.WaitForNextHeightTM(port)

	fooAddr, _ := executeGetAddrPK(t, "toncli keys show foo --output=json")
	fooCech, err := sdk.Bech32ifyAcc(fooAddr)
	require.NoError(t, err)
	barAddr, _ := executeGetAddrPK(t, "toncli keys show bar --output=json")
	barCech, err := sdk.Bech32ifyAcc(barAddr)
	require.NoError(t, err)

	fooAcc := executeGetAccount(t, fmt.Sprintf("toncli account %v %v", fooCech, flags))
	require.Equal(t, int64(50), fooAcc.GetCoins().AmountOf("steak").Int64())

	executeWrite(t, fmt.Sprintf("toncli send %v --amount=10steak --to=%v --name=foo", flags, barCech), pass)
	tests.WaitForNextHeightTM(port)

	barAcc := executeGetAccount(t, fmt.Sprintf("toncli account %v %v", barCech, flags))
	require.Equal(t, int64(10), barAcc.GetCoins().AmountOf("steak").Int64())
	fooAcc = executeGetAccount(t, fmt.Sprintf("toncli account %v %v", fooCech, flags))
	require.Equal(t, int64(40), fooAcc.GetCoins().AmountOf("steak").Int64())

	// test autosequencing
	executeWrite(t, fmt.Sprintf("toncli send %v --amount=10steak --to=%v --name=foo", flags, barCech), pass)
	tests.WaitForNextHeightTM(port)

	barAcc = executeGetAccount(t, fmt.Sprintf("toncli account %v %v", barCech, flags))
	require.Equal(t, int64(20), barAcc.GetCoins().AmountOf("steak").Int64())
	fooAcc = executeGetAccount(t, fmt.Sprintf("toncli account %v %v", fooCech, flags))
	require.Equal(t, int64(30), fooAcc.GetCoins().AmountOf("steak").Int64())

	// test memo
	executeWrite(t, fmt.Sprintf("toncli send %v --amount=10steak --to=%v --name=foo --memo 'testmemo'", flags, barCech), pass)
	tests.WaitForNextHeightTM(port)

	barAcc = executeGetAccount(t, fmt.Sprintf("toncli account %v %v", barCech, flags))
	require.Equal(t, int64(30), barAcc.GetCoins().AmountOf("steak").Int64())
	fooAcc = executeGetAccount(t, fmt.Sprintf("toncli account %v %v", fooCech, flags))
	require.Equal(t, int64(20), fooAcc.GetCoins().AmountOf("steak").Int64())
}

func TestGaiaCLICreateValidator(t *testing.T) {

	tests.ExecuteT(t, "tond unsafe_reset_all")
	pass := "1234567890"
	executeWrite(t, "toncli keys delete foo", pass)
	executeWrite(t, "toncli keys delete bar", pass)
	chainID := executeInit(t, "tond init -o --name=foo")
	executeWrite(t, "toncli keys add bar", pass)

	// get a free port, also setup some common flags
	servAddr, port, err := server.FreeTCPAddr()
	require.NoError(t, err)
	flags := fmt.Sprintf("--node=%v --chain-id=%v", servAddr, chainID)

	// start tond server
	proc := tests.GoExecuteTWithStdout(t, fmt.Sprintf("tond start --rpc.laddr=%v", servAddr))
	defer proc.Stop(false)
	tests.WaitForTMStart(port)
	tests.WaitForNextHeightTM(port)

	fooAddr, _ := executeGetAddrPK(t, "toncli keys show foo --output=json")
	fooCech, err := sdk.Bech32ifyAcc(fooAddr)
	require.NoError(t, err)
	barAddr, barPubKey := executeGetAddrPK(t, "toncli keys show bar --output=json")
	barCech, err := sdk.Bech32ifyAcc(barAddr)
	require.NoError(t, err)
	barCeshPubKey, err := sdk.Bech32ifyValPub(barPubKey)
	require.NoError(t, err)

	executeWrite(t, fmt.Sprintf("toncli send %v --amount=10steak --to=%v --name=foo", flags, barCech), pass)
	tests.WaitForNextHeightTM(port)

	barAcc := executeGetAccount(t, fmt.Sprintf("toncli account %v %v", barCech, flags))
	require.Equal(t, int64(10), barAcc.GetCoins().AmountOf("steak").Int64())
	fooAcc := executeGetAccount(t, fmt.Sprintf("toncli account %v %v", fooCech, flags))
	require.Equal(t, int64(40), fooAcc.GetCoins().AmountOf("steak").Int64())

	// create validator
	cvStr := fmt.Sprintf("toncli stake create-validator %v", flags)
	cvStr += fmt.Sprintf(" --name=%v", "bar")
	cvStr += fmt.Sprintf(" --address-validator=%v", barCech)
	cvStr += fmt.Sprintf(" --pubkey=%v", barCeshPubKey)
	cvStr += fmt.Sprintf(" --amount=%v", "2steak")
	cvStr += fmt.Sprintf(" --moniker=%v", "bar-vally")

	executeWrite(t, cvStr, pass)
	tests.WaitForNextHeightTM(port)

	barAcc = executeGetAccount(t, fmt.Sprintf("toncli account %v %v", barCech, flags))
	require.Equal(t, int64(8), barAcc.GetCoins().AmountOf("steak").Int64(), "%v", barAcc)

	validator := executeGetValidator(t, fmt.Sprintf("toncli stake validator %v --output=json %v", barCech, flags))
	require.Equal(t, validator.Owner, barAddr)
	require.Equal(t, "2/1", validator.PoolShares.Amount.String())

	// unbond a single share
	unbondStr := fmt.Sprintf("toncli stake unbond begin %v", flags)
	unbondStr += fmt.Sprintf(" --name=%v", "bar")
	unbondStr += fmt.Sprintf(" --address-validator=%v", barCech)
	unbondStr += fmt.Sprintf(" --address-delegator=%v", barCech)
	unbondStr += fmt.Sprintf(" --shares-amount=%v", "1")

	success := executeWrite(t, unbondStr, pass)
	require.True(t, success)
	tests.WaitForNextHeightTM(port)

	/* // this won't be what we expect because we've only started unbonding, haven't completed
	barAcc = executeGetAccount(t, fmt.Sprintf("toncli account %v %v", barCech, flags))
	require.Equal(t, int64(9), barAcc.GetCoins().AmountOf("steak").Int64(), "%v", barAcc)
	*/
	validator = executeGetValidator(t, fmt.Sprintf("toncli stake validator %v --output=json %v", barCech, flags))
	require.Equal(t, "1/1", validator.PoolShares.Amount.String())
}

func TestGaiaCLISubmitProposal(t *testing.T) {

	tests.ExecuteT(t, "tond unsafe_reset_all")
	pass := "1234567890"
	executeWrite(t, "toncli keys delete foo", pass)
	executeWrite(t, "toncli keys delete bar", pass)
	chainID := executeInit(t, "tond init -o --name=foo")
	executeWrite(t, "toncli keys add bar", pass)

	// get a free port, also setup some common flags
	servAddr, port, err := server.FreeTCPAddr()
	require.NoError(t, err)
	flags := fmt.Sprintf("--node=%v --chain-id=%v", servAddr, chainID)

	// start tond server
	proc := tests.GoExecuteTWithStdout(t, fmt.Sprintf("tond start --rpc.laddr=%v", servAddr))
	defer proc.Stop(false)
	tests.WaitForTMStart(port)
	tests.WaitForNextHeightTM(port)

	fooAddr, _ := executeGetAddrPK(t, "toncli keys show foo --output=json")
	fooCech, err := sdk.Bech32ifyAcc(fooAddr)
	require.NoError(t, err)

	fooAcc := executeGetAccount(t, fmt.Sprintf("toncli account %v %v", fooCech, flags))
	require.Equal(t, int64(50), fooAcc.GetCoins().AmountOf("steak").Int64())

	executeWrite(t, fmt.Sprintf("toncli gov submitproposal %v --proposer=%v --deposit=5steak --type=Text --title=Test --description=test --name=foo", flags, fooCech), pass)
	tests.WaitForNextHeightTM(port)

	fooAcc = executeGetAccount(t, fmt.Sprintf("toncli account %v %v", fooCech, flags))
	require.Equal(t, int64(45), fooAcc.GetCoins().AmountOf("steak").Int64())

	proposal1 := executeGetProposal(t, fmt.Sprintf("toncli gov query-proposal --proposalID=1 --output=json %v", flags))
	require.Equal(t, int64(1), proposal1.ProposalID)
	require.Equal(t, gov.StatusToString(gov.StatusDepositPeriod), proposal1.Status)

	executeWrite(t, fmt.Sprintf("toncli gov deposit %v --depositer=%v --deposit=10steak --proposalID=1 --name=foo", flags, fooCech), pass)
	tests.WaitForNextHeightTM(port)

	fooAcc = executeGetAccount(t, fmt.Sprintf("toncli account %v %v", fooCech, flags))
	require.Equal(t, int64(35), fooAcc.GetCoins().AmountOf("steak").Int64())
	proposal1 = executeGetProposal(t, fmt.Sprintf("toncli gov query-proposal --proposalID=1 --output=json %v", flags))
	require.Equal(t, int64(1), proposal1.ProposalID)
	require.Equal(t, gov.StatusToString(gov.StatusVotingPeriod), proposal1.Status)

	executeWrite(t, fmt.Sprintf("toncli gov vote %v --proposalID=1 --voter=%v --option=Yes --name=foo", flags, fooCech), pass)
	tests.WaitForNextHeightTM(port)

	vote := executeGetVote(t, fmt.Sprintf("toncli gov query-vote  --proposalID=1 --voter=%v --output=json %v", fooCech, flags))
	require.Equal(t, int64(1), vote.ProposalID)
	require.Equal(t, gov.VoteOptionToString(gov.OptionYes), vote.Option)
}

//___________________________________________________________________________________
// executors

func executeWrite(t *testing.T, cmdStr string, writes ...string) bool {
	proc := tests.GoExecuteT(t, cmdStr)

	for _, write := range writes {
		_, err := proc.StdinPipe.Write([]byte(write + "\n"))
		require.NoError(t, err)
	}
	stdout, stderr, err := proc.ReadAll()
	if err != nil {
		fmt.Println("Err on proc.ReadAll()", err, cmdStr)
	}
	// Log output.
	if len(stdout) > 0 {
		t.Log("Stdout:", cmn.Green(string(stdout)))
	}
	if len(stderr) > 0 {
		t.Log("Stderr:", cmn.Red(string(stderr)))
	}

	proc.Wait()
	return proc.ExitState.Success()
	//	bz := proc.StdoutBuffer.Bytes()
	//	fmt.Println("EXEC WRITE", string(bz))
}

func executeInit(t *testing.T, cmdStr string) (chainID string) {
	out := tests.ExecuteT(t, cmdStr)

	var initRes map[string]json.RawMessage
	err := json.Unmarshal([]byte(out), &initRes)
	require.NoError(t, err)

	err = json.Unmarshal(initRes["chain_id"], &chainID)
	require.NoError(t, err)

	return
}

func executeGetAddrPK(t *testing.T, cmdStr string) (sdk.Address, crypto.PubKey) {
	out := tests.ExecuteT(t, cmdStr)
	var ko keys.KeyOutput
	keys.UnmarshalJSON([]byte(out), &ko)

	address, err := sdk.GetAccAddressBech32(ko.Address)
	require.NoError(t, err)

	pk, err := sdk.GetAccPubKeyBech32(ko.PubKey)
	require.NoError(t, err)

	return address, pk
}

func executeGetAccount(t *testing.T, cmdStr string) auth.BaseAccount {
	out := tests.ExecuteT(t, cmdStr)
	var initRes map[string]json.RawMessage
	err := json.Unmarshal([]byte(out), &initRes)
	require.NoError(t, err, "out %v, err %v", out, err)
	value := initRes["value"]
	var acc auth.BaseAccount
	cdc := wire.NewCodec()
	wire.RegisterCrypto(cdc)
	err = cdc.UnmarshalJSON(value, &acc)
	require.NoError(t, err, "value %v, err %v", string(value), err)
	return acc
}

func executeGetValidator(t *testing.T, cmdStr string) stake.Validator {
	out := tests.ExecuteT(t, cmdStr)
	var validator stake.Validator
	cdc := app.MakeCodec()
	err := cdc.UnmarshalJSON([]byte(out), &validator)
	require.NoError(t, err, "out %v\n, err %v", out, err)
	return validator
}

func executeGetProposal(t *testing.T, cmdStr string) gov.ProposalRest {
	out := tests.ExecuteT(t, cmdStr)
	var proposal gov.ProposalRest
	cdc := app.MakeCodec()
	err := cdc.UnmarshalJSON([]byte(out), &proposal)
	require.NoError(t, err, "out %v\n, err %v", out, err)
	return proposal
}

func executeGetVote(t *testing.T, cmdStr string) gov.VoteRest {
	out := tests.ExecuteT(t, cmdStr)
	var vote gov.VoteRest
	cdc := app.MakeCodec()
	err := cdc.UnmarshalJSON([]byte(out), &vote)
	require.NoError(t, err, "out %v\n, err %v", out, err)
	return vote
}
