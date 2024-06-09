package cache

import (
	"context"
	"cosmossdk.io/collections"
	storetypes "cosmossdk.io/store/types"
	"fmt"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/pkg/errors"
	"sync"
)

type ItemTransaction[V any] struct {
	txKey  string
	change Entry[V]
}

type ItemTransactions[V any] struct {
	sync.RWMutex

	transactions []*ItemTransaction[V]
}

func (mt *ItemTransactions[V]) Get(txKey string) *ItemTransaction[V] {
	if txKey == "" {
		return nil
	}

	return mt.get(txKey)
}

func (mt *ItemTransactions[V]) GetCreate(txKey string) *ItemTransaction[V] {
	if txKey == "" {
		return nil
	}

	itemTransaction := mt.get(txKey)
	if itemTransaction != nil {
		return itemTransaction
	}

	itemTransaction = &ItemTransaction[V]{txKey: txKey}
	mt.set(itemTransaction)

	return itemTransaction
}

func (mt *ItemTransactions[V]) get(key string) *ItemTransaction[V] {
	mt.RLock()
	defer mt.RUnlock()

	for _, itemTransaction := range mt.transactions {
		if itemTransaction.txKey == key {
			return itemTransaction
		}
	}

	return nil
}

func (mt *ItemTransactions[V]) set(itemTransaction *ItemTransaction[V]) {
	mt.Lock()
	defer mt.Unlock()

	for _, it := range mt.transactions {
		if itemTransaction == it {
			return
		}
	}

	mt.transactions = append(mt.transactions, itemTransaction)
}

func (mt *ItemTransactions[V]) remove(key string) {
	mt.Lock()
	defer mt.Unlock()

	index := -1
	for i, itemTransaction := range mt.transactions {
		if itemTransaction.txKey == key {
			index = i
			break
		}
	}

	if index != -1 {
		mt.transactions = append(mt.transactions[:index], mt.transactions[index+1:]...)
	}
}

type ItemCache[V any] struct {
	collection collections.Item[V]

	item *Entry[V]

	transactions *ItemTransactions[V]
	comparer     ValueComparer[V]
}

func NewItemCache[V any](collection collections.Item[V], caches *Caches, comparer ValueComparer[V]) *ItemCache[V] {
	ic := &ItemCache[V]{
		collection:   collection,
		transactions: &ItemTransactions[V]{},
		comparer:     comparer,
	}

	*caches = append(*caches, ic)
	return ic
}

func (it *ItemCache[V]) Initialize(goCtx context.Context) error {
	ctx := sdk.UnwrapSDKContext(goCtx)

	if it.item != nil {
		return nil
	}

	item := it.loadFromStorage(ctx, false)
	it.item = &item

	return nil
}

func (it *ItemCache[V]) Get(ctx context.Context) (V, bool) {
	txKey, hasTX := getTXKey(ctx)
	if hasTX {
		change := it.transactions.Get(txKey)
		if change != nil {
			return getEntry(ctx, change.change, true)
		}
	}

	if it.item != nil {
		return getEntry(ctx, *it.item, hasTX)
	}

	item := it.loadFromStorage(ctx, false)
	it.item = &item
	return getEntry(ctx, item, false)
}

func (it *ItemCache[V]) loadFromStorage(goCtx context.Context, preventGasConsumption bool) Entry[V] {
	ctx := sdk.UnwrapSDKContext(goCtx)

	if preventGasConsumption {
		ctx = ctx.WithGasMeter(storetypes.NewInfiniteGasMeter())
	}

	before := ctx.GasMeter().GasConsumed()
	value, err := it.collection.Get(ctx)
	after := ctx.GasMeter().GasConsumed()

	if err != nil {
		return Entry[V]{}
	}

	return Entry[V]{
		value:       &value,
		consumption: after - before,
	}
}

func (it *ItemCache[V]) Set(ctx context.Context, value V) {
	txKey, has := getTXKey(ctx)
	if !has {
		panic("calling set without initialized cache transaction")
	}

	itemTransaction := it.transactions.GetCreate(txKey)
	itemTransaction.change = Entry[V]{value: &value}
	it.transactions.set(itemTransaction)
}

func (it *ItemCache[V]) Remove(ctx context.Context) {
	txKey, has := getTXKey(ctx)
	if !has {
		panic("calling set without initialized cache transaction")
	}

	itemTransaction := it.transactions.GetCreate(txKey)
	itemTransaction.change = Entry[V]{}
}

func (it *ItemCache[V]) Rollback(ctx context.Context) {
	txKey, has := getTXKey(ctx)
	if !has {
		panic("committing without cache transaction")
	}

	it.transactions.remove(txKey)
}

func (it *ItemCache[V]) CommitToDB(ctx context.Context) error {
	txKey, has := getTXKey(ctx)
	if !has {
		panic("committing without cache transaction")
	}

	itemTransaction := it.transactions.Get(txKey)
	it.item = nil

	if itemTransaction != nil {
		if itemTransaction.change.value != nil {
			if err := it.collection.Set(ctx, *itemTransaction.change.value); err != nil {
				return err
			}
		} else {
			return it.collection.Remove(ctx)
		}
	}

	return nil
}

func (it *ItemCache[V]) CommitToCache(ctx context.Context) {
	txKey, has := getTXKey(ctx)
	if !has {
		panic("committing without cache transaction")
	}

	itemTransaction := it.transactions.Get(txKey)
	it.item = nil

	if itemTransaction != nil {
		//if itemTransaction.change.has {
		//	item := it.loadFromStorage(ctx, true)
		//	it.item = &item
		//}
	}
}

func (it *ItemCache[V]) ClearTransactions() {
	it.transactions.transactions = nil
}

func (ic *ItemCache[V]) CheckCache(ctx context.Context) error {
	cacheItem, cacheHas := ic.Get(ctx)
	collectionItem, err := ic.collection.Get(ctx)
	collectionHas := err == nil

	if cacheHas != collectionHas {
		return fmt.Errorf("collectionHas: %v, cacheHas: %v", collectionHas, cacheHas)
	}

	if !ic.comparer(cacheItem, collectionItem) {
		return errors.New("values don't match")
	}

	return nil
}

func ValueComparerUint64(v1, v2 uint64) bool {
	return v1 == v2
}
