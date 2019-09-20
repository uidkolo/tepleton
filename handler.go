package basecoin

import (
	wrsp "github.com/tepleton/wrsp/types"
	"github.com/tepleton/go-wire/data"
	"github.com/tepleton/tmlibs/log"

	"github.com/tepleton/basecoin/state"
)

// Handler is anything that processes a transaction
type Handler interface {
	// Checker verifies there are valid fees and estimates work
	Checker
	// Deliver performs the tx once it makes it in the block
	Deliver
	// InitStater sets state from the genesis file
	InitStater
	// InitValidater sets the initial validator set
	InitValidater
	// Named ensures there is a name for the item
	Named

	// TODO????
	// BeginBlock(store state.SimpleDB, hash []byte, header *wrsp.Header)
}

// Named ensures there is a name for the item
type Named interface {
	Name() string
}

// Checker verifies there are valid fees and estimates work
type Checker interface {
	CheckTx(ctx Context, store state.SimpleDB, tx Tx) (CheckResult, error)
}

// CheckerFunc (like http.HandlerFunc) is a shortcut for making wrapers
type CheckerFunc func(Context, state.SimpleDB, Tx) (CheckResult, error)

func (c CheckerFunc) CheckTx(ctx Context, store state.SimpleDB, tx Tx) (CheckResult, error) {
	return c(ctx, store, tx)
}

// Deliver performs the tx once it makes it in the block
type Deliver interface {
	DeliverTx(ctx Context, store state.SimpleDB, tx Tx) (DeliverResult, error)
}

// DeliverFunc (like http.HandlerFunc) is a shortcut for making wrapers
type DeliverFunc func(Context, state.SimpleDB, Tx) (DeliverResult, error)

func (c DeliverFunc) DeliverTx(ctx Context, store state.SimpleDB, tx Tx) (DeliverResult, error) {
	return c(ctx, store, tx)
}

// InitStater sets state from the genesis file
type InitStater interface {
	InitState(l log.Logger, store state.SimpleDB, module, key, value string) (string, error)
}

// InitStateFunc (like http.HandlerFunc) is a shortcut for making wrapers
type InitStateFunc func(log.Logger, state.SimpleDB, string, string, string) (string, error)

func (c InitStateFunc) InitState(l log.Logger, store state.SimpleDB, module, key, value string) (string, error) {
	return c(l, store, module, key, value)
}

// InitValidater sets the initial validator set
type InitValidater interface {
	InitValidate(log log.Logger, store state.SimpleDB, vals []*wrsp.Validator)
}

// InitValidateFunc (like http.HandlerFunc) is a shortcut for making wrapers
type InitValidateFunc func(log.Logger, state.SimpleDB, []*wrsp.Validator)

func (c InitValidateFunc) InitValidate(l log.Logger, store state.SimpleDB, vals []*wrsp.Validator) {
	c(l, store, vals)
}

//---------- results and some wrappers --------

type Dataer interface {
	GetData() data.Bytes
}

// CheckResult captures any non-error wrsp result
// to make sure people use error for error cases
type CheckResult struct {
	Data data.Bytes
	Log  string
	// GasAllocated is the maximum units of work we allow this tx to perform
	GasAllocated uint
	// GasPayment is the total fees for this tx (or other source of payment)
	GasPayment uint
}

// NewCheck sets the gas used and the response data but no more info
// these are the most common info needed to be set by the Handler
func NewCheck(gasAllocated uint, log string) CheckResult {
	return CheckResult{
		GasAllocated: gasAllocated,
		Log:          log,
	}
}

var _ Dataer = CheckResult{}

func (r CheckResult) ToWRSP() wrsp.Result {
	return wrsp.Result{
		Data: r.Data,
		Log:  r.Log,
	}
}

func (r CheckResult) GetData() data.Bytes {
	return r.Data
}

// DeliverResult captures any non-error wrsp result
// to make sure people use error for error cases
type DeliverResult struct {
	Data    data.Bytes
	Log     string
	Diff    []*wrsp.Validator
	GasUsed uint
}

var _ Dataer = DeliverResult{}

func (r DeliverResult) ToWRSP() wrsp.Result {
	return wrsp.Result{
		Data: r.Data,
		Log:  r.Log,
	}
}

func (r DeliverResult) GetData() data.Bytes {
	return r.Data
}

// placeholders
// holders
type NopCheck struct{}

func (_ NopCheck) CheckTx(Context, state.SimpleDB, Tx) (r CheckResult, e error) { return }

type NopDeliver struct{}

func (_ NopDeliver) DeliverTx(Context, state.SimpleDB, Tx) (r DeliverResult, e error) { return }

type NopInitState struct{}

func (_ NopInitState) InitState(log.Logger, state.SimpleDB, string, string, string) (string, error) {
	return "", nil
}

type NopInitValidate struct{}

func (_ NopInitValidate) InitValidate(log log.Logger, store state.SimpleDB, vals []*wrsp.Validator) {}
