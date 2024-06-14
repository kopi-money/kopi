package cachetest

import (
	"context"
	"cosmossdk.io/core/store"
)

func KVStoreService(ctx *DummyCtx) store.KVStoreService {
	ctx.stores["test"] = newMemDB()
	return kvStoreService{
		moduleName: "test",
	}
}

type kvStoreService struct {
	moduleName string
}

type BaseGetter interface {
	Base() context.Context
}

func (k kvStoreService) OpenKVStore(goCtx context.Context) store.KVStore {
	ctx, ok := goCtx.(BaseGetter)
	if !ok {
		panic("could not get base")
	}

	kv, ok := ctx.Base().(StoreGetter)
	if !ok {
		panic("could not get store")
	}

	s, _ := kv.GetStore(k.moduleName)
	return s
}
