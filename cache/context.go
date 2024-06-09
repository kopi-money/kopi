package cache

import (
	"context"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type CacheContext interface {
	getTXKey() string
}

type Context struct {
	context.Context
	txKey string
}

func (c Context) getTXKey() string {
	return c.txKey
}

func NewCacheContext(baseContext context.Context, blockHeight int64) Context {
	return Context{
		Context: baseContext,
		txKey:   t.createKey(blockHeight),
	}
}

func getTXKey(goCtx context.Context) (string, bool) {
	sdkCtx := sdk.UnwrapSDKContext(goCtx)
	cacheCtx, is := sdkCtx.Context().(CacheContext)
	if !is {
		return "", false
	}

	return cacheCtx.getTXKey(), true
}
