package abi

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	wire "github.com/tepleton/go-wire"
	"github.com/tepleton/light-client/certifiers"
	"github.com/tepleton/tmlibs/log"

	"github.com/tepleton/basecoin"
	"github.com/tepleton/basecoin/errors"
	"github.com/tepleton/basecoin/modules/auth"
	"github.com/tepleton/basecoin/modules/coin"
	"github.com/tepleton/basecoin/stack"
	"github.com/tepleton/basecoin/state"
)

type checkErr func(error) bool

func noErr(err error) bool {
	return err == nil
}

// this tests registration without registrar permissions
func TestABIRegister(t *testing.T) {
	assert := assert.New(t)

	// the validators we use to make seeds
	keys := certifiers.GenValKeys(5)
	keys2 := certifiers.GenValKeys(7)
	appHash := []byte{0, 4, 7, 23}
	appHash2 := []byte{12, 34, 56, 78}

	// badSeed doesn't validate
	badSeed := genEmptySeed(keys2, "chain-2", 123, appHash, len(keys2))
	badSeed.Header.AppHash = appHash2

	cases := []struct {
		seed    certifiers.Seed
		checker checkErr
	}{
		{
			genEmptySeed(keys, "chain-1", 100, appHash, len(keys)),
			noErr,
		},
		{
			genEmptySeed(keys, "chain-1", 200, appHash, len(keys)),
			IsAlreadyRegisteredErr,
		},
		{
			badSeed,
			IsInvalidCommitErr,
		},
		{
			genEmptySeed(keys2, "chain-2", 123, appHash2, 5),
			noErr,
		},
	}

	ctx := stack.MockContext("hub", 50)
	store := state.NewMemKVStore()
	app := stack.New().Dispatch(stack.WrapHandler(NewHandler()))

	for i, tc := range cases {
		tx := RegisterChainTx{tc.seed}.Wrap()
		_, err := app.DeliverTx(ctx, store, tx)
		assert.True(tc.checker(err), "%d: %+v", i, err)
	}
}

// this tests permission controls on abi registration
func TestABIRegisterPermissions(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	// the validators we use to make seeds
	keys := certifiers.GenValKeys(4)
	appHash := []byte{0x17, 0x21, 0x5, 0x1e}

	foobar := basecoin.Actor{App: "foo", Address: []byte("bar")}
	baz := basecoin.Actor{App: "baz", Address: []byte("bar")}
	foobaz := basecoin.Actor{App: "foo", Address: []byte("baz")}

	cases := []struct {
		seed      certifiers.Seed
		registrar basecoin.Actor
		signer    basecoin.Actor
		checker   checkErr
	}{
		// no sig, no registrar
		{
			seed:    genEmptySeed(keys, "chain-1", 100, appHash, len(keys)),
			checker: noErr,
		},
		// sig, no registrar
		{
			seed:    genEmptySeed(keys, "chain-2", 100, appHash, len(keys)),
			signer:  foobaz,
			checker: noErr,
		},
		// registrar, no sig
		{
			seed:      genEmptySeed(keys, "chain-3", 100, appHash, len(keys)),
			registrar: foobar,
			checker:   errors.IsUnauthorizedErr,
		},
		// registrar, wrong sig
		{
			seed:      genEmptySeed(keys, "chain-4", 100, appHash, len(keys)),
			signer:    foobaz,
			registrar: foobar,
			checker:   errors.IsUnauthorizedErr,
		},
		// registrar, wrong sig
		{
			seed:      genEmptySeed(keys, "chain-5", 100, appHash, len(keys)),
			signer:    baz,
			registrar: foobar,
			checker:   errors.IsUnauthorizedErr,
		},
		// registrar, proper sig
		{
			seed:      genEmptySeed(keys, "chain-6", 100, appHash, len(keys)),
			signer:    foobar,
			registrar: foobar,
			checker:   noErr,
		},
	}

	store := state.NewMemKVStore()
	app := stack.New().Dispatch(stack.WrapHandler(NewHandler()))

	for i, tc := range cases {
		// set option specifies the registrar
		msg, err := json.Marshal(tc.registrar)
		require.Nil(err, "%+v", err)
		_, err = app.SetOption(log.NewNopLogger(), store,
			NameABI, OptionRegistrar, string(msg))
		require.Nil(err, "%+v", err)

		// add permissions to the context
		ctx := stack.MockContext("hub", 50).WithPermissions(tc.signer)
		tx := RegisterChainTx{tc.seed}.Wrap()
		_, err = app.DeliverTx(ctx, store, tx)
		assert.True(tc.checker(err), "%d: %+v", i, err)
	}
}

// this verifies that we can properly update the headers on the chain
func TestABIUpdate(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	// this is the root seed, that others are evaluated against
	keys := certifiers.GenValKeys(7)
	appHash := []byte{0, 4, 7, 23}
	start := 100 // initial height
	root := genEmptySeed(keys, "chain-1", 100, appHash, len(keys))

	keys2 := keys.Extend(2)
	keys3 := keys2.Extend(2)

	// create the app and register the root of trust (for chain-1)
	ctx := stack.MockContext("hub", 50)
	store := state.NewMemKVStore()
	app := stack.New().Dispatch(stack.WrapHandler(NewHandler()))
	tx := RegisterChainTx{root}.Wrap()
	_, err := app.DeliverTx(ctx, store, tx)
	require.Nil(err, "%+v", err)

	cases := []struct {
		seed    certifiers.Seed
		checker checkErr
	}{
		// same validator, higher up
		{
			genEmptySeed(keys, "chain-1", start+50, []byte{22}, len(keys)),
			noErr,
		},
		// same validator, between existing (not most recent)
		{
			genEmptySeed(keys, "chain-1", start+5, []byte{15, 43}, len(keys)),
			noErr,
		},
		// same validators, before root of trust
		{
			genEmptySeed(keys, "chain-1", start-8, []byte{11, 77}, len(keys)),
			IsHeaderNotFoundErr,
		},
		// insufficient signatures
		{
			genEmptySeed(keys, "chain-1", start+60, []byte{24}, len(keys)/2),
			IsInvalidCommitErr,
		},
		// unregistered chain
		{
			genEmptySeed(keys, "chain-2", start+60, []byte{24}, len(keys)/2),
			IsNotRegisteredErr,
		},
		// too much change (keys -> keys3)
		{
			genEmptySeed(keys3, "chain-1", start+100, []byte{22}, len(keys3)),
			IsInvalidCommitErr,
		},
		// legit update to validator set (keys -> keys2)
		{
			genEmptySeed(keys2, "chain-1", start+90, []byte{33}, len(keys2)),
			noErr,
		},
		// now impossible jump works (keys -> keys2 -> keys3)
		{
			genEmptySeed(keys3, "chain-1", start+100, []byte{44}, len(keys3)),
			noErr,
		},
	}

	for i, tc := range cases {
		tx := UpdateChainTx{tc.seed}.Wrap()
		_, err := app.DeliverTx(ctx, store, tx)
		assert.True(tc.checker(err), "%d: %+v", i, err)
	}
}

// try to create an abi packet and verify the number we get back
func TestABICreatePacket(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	// this is the root seed, that others are evaluated against
	keys := certifiers.GenValKeys(7)
	appHash := []byte{1, 2, 3, 4}
	start := 100 // initial height
	chainID := "tepleton-hub"
	root := genEmptySeed(keys, chainID, start, appHash, len(keys))

	// create the app and register the root of trust (for chain-1)
	ctx := stack.MockContext("hub", 50)
	store := state.NewMemKVStore()
	app := stack.New().Dispatch(stack.WrapHandler(NewHandler()))
	tx := RegisterChainTx{root}.Wrap()
	_, err := app.DeliverTx(ctx, store, tx)
	require.Nil(err, "%+v", err)

	// this is the tx we send, and the needed permission to send it
	raw := stack.NewRawTx([]byte{0xbe, 0xef})
	abiPerm := AllowABI(stack.NameOK)
	somePerm := basecoin.Actor{App: "some", Address: []byte("perm")}

	cases := []struct {
		dest     string
		abiPerms basecoin.Actors
		ctxPerms basecoin.Actors
		checker  checkErr
	}{
		// wrong chain -> error
		{
			dest:     "some-other-chain",
			ctxPerms: basecoin.Actors{abiPerm},
			checker:  IsNotRegisteredErr,
		},

		// no abi permission -> error
		{
			dest:    chainID,
			checker: IsNeedsABIPermissionErr,
		},

		// correct -> nice sequence
		{
			dest:     chainID,
			ctxPerms: basecoin.Actors{abiPerm},
			checker:  noErr,
		},

		// requesting invalid permissions -> error
		{
			dest:     chainID,
			abiPerms: basecoin.Actors{somePerm},
			ctxPerms: basecoin.Actors{abiPerm},
			checker:  IsCannotSetPermissionErr,
		},

		// requesting extra permissions when present
		{
			dest:     chainID,
			abiPerms: basecoin.Actors{somePerm},
			ctxPerms: basecoin.Actors{abiPerm, somePerm},
			checker:  noErr,
		},
	}

	for i, tc := range cases {
		tx := CreatePacketTx{
			DestChain:   tc.dest,
			Permissions: tc.abiPerms,
			Tx:          raw,
		}.Wrap()

		myCtx := ctx.WithPermissions(tc.ctxPerms...)
		_, err = app.DeliverTx(myCtx, store, tx)
		assert.True(tc.checker(err), "%d: %+v", i, err)
	}

	// query packet state - make sure both packets are properly writen
	p := stack.PrefixedStore(NameABI, store)
	q := OutputQueue(p, chainID)
	if assert.Equal(2, q.Size()) {
		expected := []struct {
			seq  uint64
			perm basecoin.Actors
		}{
			{0, nil},
			{1, basecoin.Actors{somePerm}},
		}

		for _, tc := range expected {
			var packet Packet
			err = wire.ReadBinaryBytes(q.Pop(), &packet)
			require.Nil(err, "%+v", err)
			assert.Equal(chainID, packet.DestChain)
			assert.EqualValues(tc.seq, packet.Sequence)
			assert.Equal(raw, packet.Tx)
			assert.Equal(len(tc.perm), len(packet.Permissions))
		}
	}
}

func TestABIPostPacket(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	otherID := "chain-1"
	ourID := "hub"
	start := 200

	// create the app and our chain
	app := stack.New().
		ABI(NewMiddleware()).
		Dispatch(
			stack.WrapHandler(NewHandler()),
			stack.WrapHandler(coin.NewHandler()),
		)
	ourChain := NewAppChain(app, ourID)

	// set up the other chain and register it with us
	otherChain := NewMockChain(otherID, 7)
	registerTx := otherChain.GetRegistrationTx(start).Wrap()
	_, err := ourChain.DeliverTx(registerTx)
	require.Nil(err, "%+v", err)

	// set up a rich guy on this chain
	wealth := coin.Coins{{"btc", 300}, {"eth", 2000}, {"ltc", 5000}}
	rich := coin.NewAccountWithKey(wealth)
	_, err = ourChain.SetOption("coin", "account", rich.MakeOption())
	require.Nil(err, "%+v", err)

	// sends money to another guy on a different chain, now other chain has credit
	buddy := basecoin.Actor{ChainID: otherID, App: auth.NameSigs, Address: []byte("dude")}
	outTx := coin.NewSendOneTx(rich.Actor(), buddy, wealth)
	_, err = ourChain.DeliverTx(outTx, rich.Actor())
	require.Nil(err, "%+v", err)

	// make sure the money moved to the other chain...
	cstore := ourChain.GetStore(coin.NameCoin)
	acct, err := coin.GetAccount(cstore, coin.ChainAddr(buddy))
	require.Nil(err, "%+v", err)
	require.Equal(wealth, acct.Coins)

	// these are the people for testing incoming abi from the other chain
	recipient := basecoin.Actor{ChainID: ourID, App: auth.NameSigs, Address: []byte("bar")}
	sender := basecoin.Actor{ChainID: otherID, App: auth.NameSigs, Address: []byte("foo")}
	coinTx := coin.NewSendOneTx(
		sender,
		recipient,
		coin.Coins{{"eth", 100}, {"ltc", 300}},
	)
	wrongCoin := coin.NewSendOneTx(sender, recipient, coin.Coins{{"missing", 20}})

	randomChain := NewMockChain("something-else", 4)
	pbad := NewPacket(coinTx, "something-else", 0)
	packetBad, _ := randomChain.MakePostPacket(pbad, 123)

	p0 := NewPacket(coinTx, ourID, 0, sender)
	packet0, update0 := otherChain.MakePostPacket(p0, start+5)
	require.Nil(ourChain.Update(update0))

	packet0badHeight := packet0
	packet0badHeight.FromChainHeight -= 2

	p1 := NewPacket(coinTx, ourID, 1, sender)
	packet1, update1 := otherChain.MakePostPacket(p1, start+25)
	require.Nil(ourChain.Update(update1))

	packet1badProof := packet1
	packet1badProof.Key = []byte("random-data")

	p2 := NewPacket(wrongCoin, ourID, 2, sender)
	packet2, update2 := otherChain.MakePostPacket(p2, start+50)
	require.Nil(ourChain.Update(update2))

	abiPerm := basecoin.Actors{AllowABI(coin.NameCoin)}
	cases := []struct {
		packet      PostPacketTx
		permissions basecoin.Actors
		checker     checkErr
	}{
		// bad chain -> error
		{packetBad, abiPerm, IsNotRegisteredErr},

		// invalid permissions -> error
		{packet0, nil, IsNeedsABIPermissionErr},

		// no matching header -> error
		{packet0badHeight, abiPerm, IsHeaderNotFoundErr},

		// out of order -> error
		{packet1, abiPerm, IsPacketOutOfOrderErr},

		// all good -> execute tx	}
		{packet0, abiPerm, noErr},

		// bad proof -> error
		{packet1badProof, abiPerm, IsInvalidProofErr},

		// all good -> execute tx }
		{packet1, abiPerm, noErr},

		// repeat -> error
		{packet0, abiPerm, IsPacketAlreadyExistsErr},

		// packet 2 attempts to spend money this chain doesn't have
		{packet2, abiPerm, coin.IsInsufficientFundsErr},
	}

	for i, tc := range cases {
		_, err := ourChain.DeliverTx(tc.packet.Wrap(), tc.permissions...)
		assert.True(tc.checker(err), "%d: %+v", i, err)
	}
}
