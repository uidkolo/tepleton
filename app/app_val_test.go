package app

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	wrsp "github.com/tepleton/wrsp/types"
	wire "github.com/tepleton/go-wire"
	cmn "github.com/tepleton/tmlibs/common"
	"github.com/tepleton/tmlibs/log"

	"github.com/tepleton/tepleton-sdk/modules/base"
)

//-----------------------------------
// Test cases start here

func randPower() uint64 {
	return uint64(cmn.RandInt()%50 + 60)
}

func makeVal() *wrsp.Validator {
	return &wrsp.Validator{
		PubKey: cmn.RandBytes(10),
		Power:  randPower(),
	}
}

// withNewPower returns a copy of the validator with a different power
func withNewPower(val *wrsp.Validator) *wrsp.Validator {
	res := *val
	res.Power = randPower()
	return &res
}

func TestEndBlock(t *testing.T) {
	assert, require := assert.New(t), require.New(t)

	logger := log.NewNopLogger()
	handler := base.ValSetHandler{}
	app, err := NewBasecoin(handler, "", 0, logger)
	require.Nil(err, "%+v", err)

	val1 := makeVal()
	val2 := makeVal()
	val3 := makeVal()
	val1a := withNewPower(val1)
	val2a := withNewPower(val2)

	cases := [...]struct {
		changes  [][]*wrsp.Validator
		expected []*wrsp.Validator
	}{
		// Nothing in, nothing out, no crash
		0: {},
		// One in, one out, no problem
		1: {
			changes:  [][]*wrsp.Validator{{val1}},
			expected: []*wrsp.Validator{val1},
		},
		// Combine a few ones
		2: {
			changes:  [][]*wrsp.Validator{{val1}, {val2, val3}},
			expected: []*wrsp.Validator{val1, val2, val3},
		},
		// Make sure changes all to one validators are squished into one diff
		3: {
			changes:  [][]*wrsp.Validator{{val1}, {val2, val1a}, {val2a, val3}},
			expected: []*wrsp.Validator{val1a, val2a, val3},
		},
	}

	for i, tc := range cases {
		app.BeginBlock(wrsp.RequestBeginBlock{})
		for _, c := range tc.changes {
			tx := base.ValChangeTx{c}.Wrap()
			txBytes := wire.BinaryBytes(tx)
			res := app.DeliverTx(txBytes)
			require.True(res.IsOK(), "%#v", res)
		}
		diff := app.EndBlock(app.height)
		// TODO: don't care about order here...
		assert.Equal(tc.expected, diff.Diffs, "%d", i)
	}
}
