package cache

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

type Height interface {
	BlockHeight() int64
}

type CacheContext interface {
	context.Context
	getTXKey() *TXKey
}

type ValueContext interface {
	Value(key any) any
}

func NewCacheContext(baseContext context.Context, blockHeight int64, finalizing bool) context.Context {
	return context.WithValue(baseContext, "cache-tx-key", TransactionHandler.createKey(blockHeight, finalizing))
}

func getTXKey(ctx context.Context) *TXKey {
	txKey, ok := ctx.Value("cache-tx-key").(*TXKey)
	if ok {
		return txKey
	}

	return nil
}

func getCurrentHeight(ctx context.Context) int64 {
	innerCtx := ctx.Value(sdk.SdkContextKey)
	if innerCtx != nil {
		baseCtx, ok := innerCtx.(sdk.Context)
		if ok {
			return baseCtx.BlockHeight()
		}
	}

	return 0
}
