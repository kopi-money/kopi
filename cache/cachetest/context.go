package cachetest

import (
	"context"
	"cosmossdk.io/core/store"
	storetypes "cosmossdk.io/store/types"
)

type StoreGetter interface {
	GetStore(string) (store.KVStore, bool)
}

type DummyCtx struct {
	context.Context

	stores   map[string]store.KVStore
	gasMeter storetypes.GasMeter
}

func (dc *DummyCtx) GasMeter() storetypes.GasMeter {
	return dc.gasMeter
}

func (dc *DummyCtx) WithGasMeter(gasMeter storetypes.GasMeter) context.Context {
	dc.gasMeter = gasMeter
	return dc
}

func (dc *DummyCtx) GetStore(key string) (store.KVStore, bool) {
	s, ok := dc.stores[key]
	return s, ok
}

func Deps() (store.KVStoreService, context.Context) {
	ctx := &DummyCtx{
		Context:  context.Background(),
		stores:   map[string]store.KVStore{},
		gasMeter: storetypes.NewInfiniteGasMeter(),
	}
	kv := KVStoreService(ctx)

	return kv, ctx
}
