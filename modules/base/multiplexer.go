package base

import (
	"strings"

	wire "github.com/tepleton/go-wire"
	"github.com/tepleton/go-wire/data"

	"github.com/tepleton/basecoin"
	"github.com/tepleton/basecoin/stack"
	"github.com/tepleton/basecoin/state"
)

//nolint
const (
	NameMultiplexer = "mplx"
)

// Multiplexer grabs a MultiTx and sends them sequentially down the line
type Multiplexer struct {
	stack.PassOption
}

// Name of the module - fulfills Middleware interface
func (Multiplexer) Name() string {
	return NameMultiplexer
}

var _ stack.Middleware = Multiplexer{}

// CheckTx splits the input tx and checks them all - fulfills Middlware interface
func (Multiplexer) CheckTx(ctx basecoin.Context, store state.SimpleDB, tx basecoin.Tx, next basecoin.Checker) (res basecoin.Result, err error) {
	if mtx, ok := tx.Unwrap().(*MultiTx); ok {
		return runAll(ctx, store, mtx.Txs, next.CheckTx)
	}
	return next.CheckTx(ctx, store, tx)
}

// DeliverTx splits the input tx and checks them all - fulfills Middlware interface
func (Multiplexer) DeliverTx(ctx basecoin.Context, store state.SimpleDB, tx basecoin.Tx, next basecoin.Deliver) (res basecoin.Result, err error) {
	if mtx, ok := tx.Unwrap().(*MultiTx); ok {
		return runAll(ctx, store, mtx.Txs, next.DeliverTx)
	}
	return next.DeliverTx(ctx, store, tx)
}

func runAll(ctx basecoin.Context, store state.SimpleDB, txs []basecoin.Tx, next basecoin.CheckerFunc) (res basecoin.Result, err error) {
	// store all results, unless anything errors
	rs := make([]basecoin.Result, len(txs))
	for i, stx := range txs {
		rs[i], err = next(ctx, store, stx)
		if err != nil {
			return
		}
	}
	// now combine the results into one...
	return combine(rs), nil
}

// combines all data bytes as a go-wire array.
// joins all log messages with \n
func combine(all []basecoin.Result) basecoin.Result {
	datas := make([]data.Bytes, len(all))
	logs := make([]string, len(all))
	for i, r := range all {
		datas[i] = r.Data
		logs[i] = r.Log
	}
	return basecoin.Result{
		Data: wire.BinaryBytes(datas),
		Log:  strings.Join(logs, "\n"),
	}
}
