package handlers

import (
	crypto "github.com/tepleton/go-crypto"

	"github.com/tepleton/basecoin"
	"github.com/tepleton/basecoin/errors"
	"github.com/tepleton/basecoin/types"
)

// app name for auth
const (
	NameSigs = "sigs"
)

type SignedHandler struct {
	AllowMultiSig bool
}

func (_ SignedHandler) Name() string {
	return NameSigs
}

var _ basecoin.Middleware = SignedHandler{}

// Signed allows us to use txs.OneSig and txs.MultiSig (and others??)
type Signed interface {
	basecoin.TxLayer
	Signers() ([]crypto.PubKey, error)
}

func (h SignedHandler) CheckTx(ctx basecoin.Context, store types.KVStore, tx basecoin.Tx, next basecoin.Checker) (res basecoin.Result, err error) {
	sigs, tnext, err := getSigners(tx)
	if err != nil {
		return res, err
	}
	ctx2 := addSigners(ctx, sigs)
	return next.CheckTx(ctx2, store, tnext)
}

func (h SignedHandler) DeliverTx(ctx basecoin.Context, store types.KVStore, tx basecoin.Tx, next basecoin.Deliver) (res basecoin.Result, err error) {
	sigs, tnext, err := getSigners(tx)
	if err != nil {
		return res, err
	}
	ctx2 := addSigners(ctx, sigs)
	return next.DeliverTx(ctx2, store, tnext)
}

func addSigners(ctx basecoin.Context, sigs []crypto.PubKey) basecoin.Context {
	perms := make([]basecoin.Permission, len(sigs))
	for i, s := range sigs {
		perms[i] = basecoin.Permission{App: NameSigs, Address: s.Address()}
	}
	// add the signers to the context and continue
	return ctx.AddPermissions(perms...)
}

func getSigners(tx basecoin.Tx) ([]crypto.PubKey, basecoin.Tx, error) {
	stx, ok := tx.Unwrap().(Signed)
	if !ok {
		return nil, basecoin.Tx{}, errors.Unauthorized()
	}
	sig, err := stx.Signers()
	return sig, stx.Next(), err
}
