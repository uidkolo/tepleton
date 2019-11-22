package stake

import (
	"bytes"
	"encoding/hex"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"

	wrsp "github.com/tepleton/wrsp/types"
	crypto "github.com/tepleton/go-crypto"
	dbm "github.com/tepleton/tmlibs/db"
	"github.com/tepleton/tmlibs/log"

	"github.com/tepleton/tepleton-sdk/store"
	sdk "github.com/tepleton/tepleton-sdk/types"
	"github.com/tepleton/tepleton-sdk/wire"
	"github.com/tepleton/tepleton-sdk/x/auth"
	"github.com/tepleton/tepleton-sdk/x/bank"
)

// dummy addresses used for testing
var (
	addrs = []sdk.Address{
		testAddr("A58856F0FD53BF058B4909A21AEC019107BA6160", "tepletonaccaddr:5ky9du8a2wlstz6fpx3p4mqpjyrm5ctqyxjnwh"),
		testAddr("A58856F0FD53BF058B4909A21AEC019107BA6161", "tepletonaccaddr:5ky9du8a2wlstz6fpx3p4mqpjyrm5ctpesxxn9"),
		testAddr("A58856F0FD53BF058B4909A21AEC019107BA6162", "tepletonaccaddr:5ky9du8a2wlstz6fpx3p4mqpjyrm5ctzhrnsa6"),
		testAddr("A58856F0FD53BF058B4909A21AEC019107BA6163", "tepletonaccaddr:5ky9du8a2wlstz6fpx3p4mqpjyrm5ctr2489qg"),
		testAddr("A58856F0FD53BF058B4909A21AEC019107BA6164", "tepletonaccaddr:5ky9du8a2wlstz6fpx3p4mqpjyrm5ctytvs4pd"),
		testAddr("A58856F0FD53BF058B4909A21AEC019107BA6165", "tepletonaccaddr:5ky9du8a2wlstz6fpx3p4mqpjyrm5ct9k6yqul"),
		testAddr("A58856F0FD53BF058B4909A21AEC019107BA6166", "tepletonaccaddr:5ky9du8a2wlstz6fpx3p4mqpjyrm5ctxcf3kjq"),
		testAddr("A58856F0FD53BF058B4909A21AEC019107BA6167", "tepletonaccaddr:5ky9du8a2wlstz6fpx3p4mqpjyrm5ct89l9r0j"),
		testAddr("A58856F0FD53BF058B4909A21AEC019107BA6168", "tepletonaccaddr:5ky9du8a2wlstz6fpx3p4mqpjyrm5ctg6jkls2"),
		testAddr("A58856F0FD53BF058B4909A21AEC019107BA6169", "tepletonaccaddr:5ky9du8a2wlstz6fpx3p4mqpjyrm5ctf8yz2dc"),
	}

	// dummy pubkeys used for testing
	pks = []crypto.PubKey{
		newPubKey("0B485CFC0EECC619440448436F8FC9DF40566F2369E72400281454CB552AFB50"),
		newPubKey("0B485CFC0EECC619440448436F8FC9DF40566F2369E72400281454CB552AFB51"),
		newPubKey("0B485CFC0EECC619440448436F8FC9DF40566F2369E72400281454CB552AFB52"),
		newPubKey("0B485CFC0EECC619440448436F8FC9DF40566F2369E72400281454CB552AFB53"),
		newPubKey("0B485CFC0EECC619440448436F8FC9DF40566F2369E72400281454CB552AFB54"),
		newPubKey("0B485CFC0EECC619440448436F8FC9DF40566F2369E72400281454CB552AFB55"),
		newPubKey("0B485CFC0EECC619440448436F8FC9DF40566F2369E72400281454CB552AFB56"),
		newPubKey("0B485CFC0EECC619440448436F8FC9DF40566F2369E72400281454CB552AFB57"),
		newPubKey("0B485CFC0EECC619440448436F8FC9DF40566F2369E72400281454CB552AFB58"),
		newPubKey("0B485CFC0EECC619440448436F8FC9DF40566F2369E72400281454CB552AFB59"),
	}

	emptyAddr   sdk.Address
	emptyPubkey crypto.PubKey
)

//_______________________________________________________________________________________

// intended to be used with require/assert:  require.True(ValEq(...))
func ValEq(t *testing.T, exp, got Validator) (*testing.T, bool, string, Validator, Validator) {
	return t, exp.equal(got), "expected:\t%v\ngot:\t\t%v", exp, got
}

//_______________________________________________________________________________________

func makeTestCodec() *wire.Codec {
	var cdc = wire.NewCodec()

	// Register Msgs
	cdc.RegisterInterface((*sdk.Msg)(nil), nil)
	cdc.RegisterConcrete(bank.MsgSend{}, "test/stake/Send", nil)
	cdc.RegisterConcrete(bank.MsgIssue{}, "test/stake/Issue", nil)
	cdc.RegisterConcrete(MsgDeclareCandidacy{}, "test/stake/DeclareCandidacy", nil)
	cdc.RegisterConcrete(MsgEditCandidacy{}, "test/stake/EditCandidacy", nil)
	cdc.RegisterConcrete(MsgUnbond{}, "test/stake/Unbond", nil)

	// Register AppAccount
	cdc.RegisterInterface((*auth.Account)(nil), nil)
	cdc.RegisterConcrete(&auth.BaseAccount{}, "test/stake/Account", nil)
	wire.RegisterCrypto(cdc)

	return cdc
}

func paramsNoInflation() Params {
	return Params{
		InflationRateChange: sdk.ZeroRat(),
		InflationMax:        sdk.ZeroRat(),
		InflationMin:        sdk.ZeroRat(),
		GoalBonded:          sdk.NewRat(67, 100),
		MaxValidators:       100,
		BondDenom:           "steak",
	}
}

// hogpodge of all sorts of input required for testing
func createTestInput(t *testing.T, isCheckTx bool, initCoins int64) (sdk.Context, auth.AccountMapper, Keeper) {

	keyStake := sdk.NewKVStoreKey("stake")
	keyAcc := sdk.NewKVStoreKey("acc")

	db := dbm.NewMemDB()
	ms := store.NewCommitMultiStore(db)
	ms.MountStoreWithDB(keyStake, sdk.StoreTypeIAVL, db)
	ms.MountStoreWithDB(keyAcc, sdk.StoreTypeIAVL, db)
	err := ms.LoadLatestVersion()
	require.Nil(t, err)

	ctx := sdk.NewContext(ms, wrsp.Header{ChainID: "foochainid"}, isCheckTx, nil, log.NewNopLogger())
	cdc := makeTestCodec()
	accountMapper := auth.NewAccountMapper(
		cdc,                 // amino codec
		keyAcc,              // target store
		&auth.BaseAccount{}, // prototype
	)
	ck := bank.NewKeeper(accountMapper)
	keeper := NewKeeper(cdc, keyStake, ck, DefaultCodespace)
	keeper.setPool(ctx, initialPool())
	keeper.setNewParams(ctx, defaultParams())

	// fill all the addresses with some coins
	for _, addr := range addrs {
		ck.AddCoins(ctx, addr, sdk.Coins{
			{keeper.GetParams(ctx).BondDenom, initCoins},
		})
	}

	return ctx, accountMapper, keeper
}

func newPubKey(pk string) (res crypto.PubKey) {
	pkBytes, err := hex.DecodeString(pk)
	if err != nil {
		panic(err)
	}
	//res, err = crypto.PubKeyFromBytes(pkBytes)
	var pkEd crypto.PubKeyEd25519
	copy(pkEd[:], pkBytes[:])
	return pkEd
}

// for incode address generation
func testAddr(addr string, bech string) sdk.Address {

	res, err := sdk.GetAccAddressHex(addr)
	if err != nil {
		panic(err)
	}
	bechexpected, err := sdk.Bech32TepletonifyAcc(res)
	if err != nil {
		panic(err)
	}
	if bech != bechexpected {
		panic("Bech encoding doesn't match reference")
	}

	bechres, err := sdk.GetAccAddressBech32Tepleton(bech)
	if err != nil {
		panic(err)
	}
	if bytes.Compare(bechres, res) != 0 {
		panic("Bech decode and hex decode don't match")
	}

	return res
}

func createTestAddrs(numAddrs int) []sdk.Address {
	var addresses []sdk.Address
	var buffer bytes.Buffer

	//start at 10 to avoid changing 1 to 01, 2 to 02, etc
	for i := 10; i < numAddrs; i++ {
		numString := strconv.Itoa(i)
		buffer.WriteString("A58856F0FD53BF058B4909A21AEC019107BA61") //base address string

		buffer.WriteString(numString) //adding on final two digits to make addresses unique
		res, _ := sdk.GetAccAddressHex(buffer.String())
		bech, _ := sdk.Bech32TepletonifyAcc(res)
		addresses = append(addresses, testAddr(buffer.String(), bech))
		buffer.Reset()
	}
	return addresses
}

func createTestPubKeys(numPubKeys int) []crypto.PubKey {
	var publicKeys []crypto.PubKey
	var buffer bytes.Buffer

	//start at 10 to avoid changing 1 to 01, 2 to 02, etc
	for i := 10; i < numPubKeys; i++ {
		numString := strconv.Itoa(i)
		buffer.WriteString("0B485CFC0EECC619440448436F8FC9DF40566F2369E72400281454CB552AFB") //base pubkey string
		buffer.WriteString(numString)                                                        //adding on final two digits to make pubkeys unique
		publicKeys = append(publicKeys, newPubKey(buffer.String()))
		buffer.Reset()
	}
	return publicKeys
}
