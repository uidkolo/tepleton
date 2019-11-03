package lib

import (
	"fmt"
	"strconv"

	sdk "github.com/tepleton/tepleton-sdk/types"
	wire "github.com/tepleton/tepleton-sdk/wire"
)

type Mapper struct {
	key    sdk.StoreKey
	cdc    *wire.Codec
	prefix string
}

// ListMapper is a Mapper interface that provides list-like functions
// It panics when the element type cannot be (un/)marshalled by the codec

type ListMapper interface {
	// ListMapper dosen't check if an index is in bounds
	// The user should check Len() before doing any actions
	Len(sdk.Context) uint64

	Get(sdk.Context, uint64, interface{}) error

	// Setting element out of range will break length counting
	// Use Push() instead of Set() to append a new element
	Set(sdk.Context, uint64, interface{})

	// Other elements' indices are preserved after deletion
	// Panics when the index is out of range
	Delete(sdk.Context, uint64)

	// Push will increase the length when it is called
	// The other methods does not modify the length
	Push(sdk.Context, interface{})

	// Iterate*() is used to iterate over all existing elements in the list
	// Return true in the continuation to break
	// The second element of the continuation will indicate the position of the element
	// Using it with Get() will return the same one with the provided element

	// CONTRACT: No writes may happen within a domain while iterating over it.
	IterateRead(sdk.Context, interface{}, func(sdk.Context, uint64) bool)

	// IterateWrite() is safe to write over the domain
	IterateWrite(sdk.Context, interface{}, func(sdk.Context, uint64) bool)

	// Key for the length of the list
	LengthKey() []byte

	// Key for getting elements
	ElemKey(uint64) []byte
}

func NewListMapper(cdc *wire.Codec, key sdk.StoreKey, prefix string) ListMapper {
	return Mapper{
		key:    key,
		cdc:    cdc,
		prefix: prefix,
	}
}

func (lm Mapper) Len(ctx sdk.Context) uint64 {
	store := ctx.KVStore(lm.key)
	bz := store.Get(lm.LengthKey())
	if bz == nil {
		zero, err := lm.cdc.MarshalBinary(0)
		if err != nil {
			panic(err)
		}
		store.Set(lm.LengthKey(), zero)
		return 0
	}
	var res uint64
	if err := lm.cdc.UnmarshalBinary(bz, &res); err != nil {
		panic(err)
	}
	return res
}

func (lm Mapper) Get(ctx sdk.Context, index uint64, ptr interface{}) error {
	store := ctx.KVStore(lm.key)
	bz := store.Get(lm.ElemKey(index))
	return lm.cdc.UnmarshalBinary(bz, ptr)
}

func (lm Mapper) Set(ctx sdk.Context, index uint64, value interface{}) {
	store := ctx.KVStore(lm.key)
	bz, err := lm.cdc.MarshalBinary(value)
	if err != nil {
		panic(err)
	}
	store.Set(lm.ElemKey(index), bz)
}

func (lm Mapper) Delete(ctx sdk.Context, index uint64) {
	store := ctx.KVStore(lm.key)
	store.Delete(lm.ElemKey(index))
}

func (lm Mapper) Push(ctx sdk.Context, value interface{}) {
	length := lm.Len(ctx)
	lm.Set(ctx, length, value)

	store := ctx.KVStore(lm.key)
	store.Set(lm.LengthKey(), marshalUint64(lm.cdc, length+1))
}

func (lm Mapper) IterateRead(ctx sdk.Context, ptr interface{}, fn func(sdk.Context, uint64) bool) {
	store := ctx.KVStore(lm.key)
	start, end := subspace([]byte(fmt.Sprintf("%s/elem/", lm.prefix)))
	iter := store.Iterator(start, end)
	for ; iter.Valid(); iter.Next() {
		v := iter.Value()
		if err := lm.cdc.UnmarshalBinary(v, ptr); err != nil {
			panic(err)
		}
		s := string(iter.Key()[len(lm.prefix)+6:])
		index, err := strconv.ParseUint(s, 10, 64)
		if err != nil {
			panic(err)
		}
		if fn(ctx, index) {
			break
		}
	}

	iter.Close()
}

func (lm Mapper) IterateWrite(ctx sdk.Context, ptr interface{}, fn func(sdk.Context, uint64) bool) {
	length := lm.Len(ctx)

	for i := uint64(0); i < length; i++ {
		if err := lm.Get(ctx, i, ptr); err != nil {
			continue
		}
		if fn(ctx, i) {
			break
		}
	}
}

func (lm Mapper) LengthKey() []byte {
	return []byte(fmt.Sprintf("%s/length", lm.prefix))
}

func (lm Mapper) ElemKey(i uint64) []byte {
	return []byte(fmt.Sprintf("%s/elem/%020d", lm.prefix, i))
}

// QueueMapper is a Mapper interface that provides queue-like functions
// It panics when the element type cannot be (un/)marshalled by the codec

type QueueMapper interface {
	Push(sdk.Context, interface{})
	// Popping/Peeking on an empty queue will cause panic
	// The user should check IsEmpty() before doing any actions
	Peek(sdk.Context, interface{}) error
	Pop(sdk.Context)
	IsEmpty(sdk.Context) bool

	// Flush() removes elements it processed
	// Return true in the continuation to break
	// The interface{} is unmarshalled before the continuation is called
	// Starts from the top(head) of the queue
	// CONTRACT: Pop() or Push() should not be performed while flushing
	Flush(sdk.Context, interface{}, func(sdk.Context) bool)

	// Key for the index of top element
	TopKey() []byte
}

func NewQueueMapper(cdc *wire.Codec, key sdk.StoreKey, prefix string) QueueMapper {
	return Mapper{
		key:    key,
		cdc:    cdc,
		prefix: prefix,
	}
}

func (qm Mapper) getTop(store sdk.KVStore) (res uint64) {
	bz := store.Get(qm.TopKey())
	if bz == nil {
		store.Set(qm.TopKey(), marshalUint64(qm.cdc, 0))
		return 0
	}

	if err := qm.cdc.UnmarshalBinary(bz, &res); err != nil {
		panic(err)
	}

	return
}

func (qm Mapper) setTop(store sdk.KVStore, top uint64) {
	bz := marshalUint64(qm.cdc, top)
	store.Set(qm.TopKey(), bz)
}

func (qm Mapper) Peek(ctx sdk.Context, ptr interface{}) error {
	store := ctx.KVStore(qm.key)
	top := qm.getTop(store)
	return qm.Get(ctx, top, ptr)
}

func (qm Mapper) Pop(ctx sdk.Context) {
	store := ctx.KVStore(qm.key)
	top := qm.getTop(store)
	qm.Delete(ctx, top)
	qm.setTop(store, top+1)
}

func (qm Mapper) IsEmpty(ctx sdk.Context) bool {
	store := ctx.KVStore(qm.key)
	top := qm.getTop(store)
	length := qm.Len(ctx)
	return top >= length
}

func (qm Mapper) Flush(ctx sdk.Context, ptr interface{}, fn func(sdk.Context) bool) {
	store := ctx.KVStore(qm.key)
	top := qm.getTop(store)
	length := qm.Len(ctx)

	var i uint64
	for i = top; i < length; i++ {
		qm.Get(ctx, i, ptr)
		qm.Delete(ctx, i)
		if fn(ctx) {
			break
		}
	}

	qm.setTop(store, i)
}

func (qm Mapper) TopKey() []byte {
	return []byte(fmt.Sprintf("%s/top", qm.prefix))
}

func marshalUint64(cdc *wire.Codec, i uint64) []byte {
	bz, err := cdc.MarshalBinary(i)
	if err != nil {
		panic(err)
	}
	return bz
}

func subspace(prefix []byte) (start, end []byte) {
	end = make([]byte, len(prefix))
	copy(end, prefix)
	end[len(end)-1]++
	return prefix, end
}
