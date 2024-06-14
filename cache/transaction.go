package cache

import (
	"context"
	"cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"sync"
)

type transactionHandler struct {
	muKey    sync.Mutex
	muCommit sync.Mutex

	caches Caches

	height     int64
	count      int
	finalizing bool
}

var TransactionHandler *transactionHandler

func init() {
	NewTranscationHandler()
}

func NewTranscationHandler() {
	TransactionHandler = &transactionHandler{
		muKey:      sync.Mutex{},
		muCommit:   sync.Mutex{},
		finalizing: false,
	}
}

func AddCache(cache Cache) {
	TransactionHandler.caches = append(TransactionHandler.caches, cache)
}

func AddCaches(caches Caches) {
	TransactionHandler.caches = append(TransactionHandler.caches, caches...)
}

func (t *transactionHandler) createKey(height int64, finalizing bool) *TXKey {
	t.muKey.Lock()
	defer t.muKey.Unlock()

	if height > t.height {
		t.height = height
		t.count = 0
	}

	if finalizing && t.finalizing {
		panic("multiple finalizing!!!")
	}

	t.count++
	if finalizing {
		t.muCommit.Lock()
		t.finalizing = true
	}

	return &TXKey{
		block:      t.height,
		index:      t.count,
		finalizing: finalizing,
	}
}

func (t *transactionHandler) Initialize(ctx context.Context) error {
	for _, cache := range t.caches {
		if err := cache.Initialize(ctx); err != nil {
			return err
		}
		cache.ClearTransactions()
	}

	return nil
}

func (t *transactionHandler) CommitToDB(ctx context.Context) error {
	for _, cache := range t.caches {
		if err := cache.CommitToDB(ctx); err != nil {
			return err
		}
	}

	return nil
}

func (t *transactionHandler) Rollback(ctx context.Context) {
	for _, cache := range t.caches {
		cache.Rollback(ctx)
	}
}

func (t *transactionHandler) CommitToCache(ctx context.Context) {
	for _, cache := range t.caches {
		cache.CommitToCache(ctx)
	}
}

func (t *transactionHandler) Clear(ctx context.Context) {
	for _, cache := range t.caches {
		cache.Clear(ctx)
	}

	t.clear(ctx)
}

func (t *transactionHandler) clear(ctx context.Context) {
	if getTXKey(ctx).finalizing {
		t.muCommit.Unlock()
		t.finalizing = false
	}
}

func (t *transactionHandler) ClearTransactions() {
	for _, cache := range t.caches {
		cache.ClearTransactions()
	}

	if t.finalizing {
		t.muCommit.Unlock()
		t.finalizing = false
	}
}

type TXKey struct {
	block      int64
	index      int
	finalizing bool
}

func (t TXKey) equals(other TXKey) bool {
	return t.block == other.block && t.index == other.index && t.finalizing == other.finalizing
}

type Cache interface {
	CheckCache(ctx context.Context) error
	Clear(ctx context.Context)
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

func (c Caches) Rollback(ctx context.Context) {
	for _, cache := range c {
		cache.Rollback(ctx)
	}
}

func (c Caches) CommitToCache(ctx context.Context) {
	for _, cache := range c {
		cache.CommitToCache(ctx)
	}
}

func (c Caches) Clear(ctx context.Context) {
	for _, cache := range c {
		cache.Clear(ctx)
	}
}

func (c Caches) ClearTransactions() {
	for _, cache := range c {
		cache.ClearTransactions()
	}
}

type TransactionFunction func(sdk.Context) error

func Transact(goCtx context.Context, f TransactionFunction) error {
	ctx := sdk.UnwrapSDKContext(goCtx)
	ctx = ctx.WithContext(NewCacheContext(ctx.Context(), ctx.BlockHeight(), true))
	defer TransactionHandler.Clear(ctx)

	err := f(ctx)

	if err != nil {
		TransactionHandler.Rollback(ctx)
		return errors.Wrap(err, "error in transaction function")
	} else {
		if err = TransactionHandler.CommitToDB(ctx); err != nil {
			return errors.Wrap(err, "could not commit to db")
		}

		TransactionHandler.CommitToCache(ctx)
	}

	return nil
}
