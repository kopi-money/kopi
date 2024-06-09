package cache

import (
	"context"
	"fmt"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"sync"
)

type transactions struct {
	sync.Mutex

	height int64
	count  int
}

var t = &transactions{}

func (t *transactions) createKey(height int64) string {
	t.Lock()
	defer t.Unlock()

	if height > t.height {
		t.height = height
		t.count = 0
	}

	t.count++
	return fmt.Sprintf("%v:%v", t.height, t.count)
}

type Cache interface {
	CheckCache(ctx context.Context) error
	ClearTransactions()
	CommitToCache(ctx context.Context)
	CommitToDB(ctx context.Context) error
	Initialize(ctx context.Context) error
	Rollback(ctx context.Context)
}

type Caches []Cache

func (c Caches) CheckCache(ctx context.Context) error {
	for _, cache := range c {
		if err := cache.CheckCache(ctx); err != nil {
			return err
		}
	}

	return nil
}

func (c Caches) Initialize(ctx context.Context) error {
	for _, cache := range c {
		if err := cache.Initialize(ctx); err != nil {
			return err
		}
	}

	return nil
}

func (c Caches) CommitToDB(ctx context.Context) error {
	for _, cache := range c {
		if err := cache.CommitToDB(ctx); err != nil {
			return err
		}
	}

	return nil
}

func (c Caches) CommitToCache(ctx context.Context) {
	for _, cache := range c {
		cache.CommitToCache(ctx)
	}
}

func (c Caches) Rollback(ctx context.Context) {
	for _, cache := range c {
		cache.Rollback(ctx)
	}
}

func (c Caches) ClearTransactions() {
	for _, cache := range c {
		cache.ClearTransactions()
	}
}

type TransactionFunction func(sdk.Context) error

func Transact(goCtx context.Context, keepers []Cache, f TransactionFunction) error {
	ctx := sdk.UnwrapSDKContext(goCtx)
	ctx = ctx.WithContext(NewCacheContext(ctx.Context(), ctx.BlockHeight()))

	err := f(ctx)
	for _, keeper := range keepers {
		if err == nil {
			keeper.CommitToCache(ctx)
			if err = keeper.CommitToDB(ctx); err != nil {
				return err
			}
		} else {
			keeper.Rollback(ctx)
		}

		keeper.ClearTransactions()
	}

	return nil
}
