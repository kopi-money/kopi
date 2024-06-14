package cache

import (
	"context"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type CacheContext interface {
	context.Context
	getTXKey() *TXKey
}

type ValueContext interface {
	Value(key any) any
}

type Context struct {
	context.Context

	txKey *TXKey
}

func (c Context) Base() context.Context {
	return c.Context
}

func (c Context) getTXKey() *TXKey {
	return c.txKey
}

func NewCacheContext(baseContext context.Context, blockHeight int64, finalizing bool) Context {
	return Context{
		Context: baseContext,
		txKey:   TransactionHandler.createKey(blockHeight, finalizing),
	}
}

func getTXKey(goCtx context.Context) *TXKey {
	cacheCtx, ok := goCtx.(CacheContext)
	if ok {
		return cacheCtx.getTXKey()
	}

	baseCtx, ok := goCtx.(sdk.Context)
	if ok {
		cacheCtx, ok = baseCtx.Context().(CacheContext)
		if ok {
			return cacheCtx.getTXKey()
		}
	}

	valueCtx, ok := goCtx.(ValueContext)
	if ok {
		innerCtx := valueCtx.Value(sdk.SdkContextKey)
		if innerCtx != nil {
			baseCtx, ok = innerCtx.(sdk.Context)
			if ok {
				cacheCtx, ok = baseCtx.Context().(CacheContext)
				if ok {
					return cacheCtx.getTXKey()
				}
			}
		}
	}

	return nil
}
