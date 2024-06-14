package cache

import (
	"context"
	"cosmossdk.io/collections"
	"cosmossdk.io/collections/codec"
	storetypes "cosmossdk.io/store/types"
	"fmt"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/pkg/errors"
	"sync"
)

type ItemTransaction[V any] struct {
	txKey TXKey

	previous *Entry[V]
	change   Entry[V]
}

type ItemTransactions[V any] struct {
	sync.RWMutex

	transactions []*ItemTransaction[V]
}

func (mt *ItemTransactions[V]) Get(txKey TXKey) *ItemTransaction[V] {
	return mt.get(txKey)
}

func (mt *ItemTransactions[V]) GetCreate(txKey TXKey) *ItemTransaction[V] {
	itemTransaction := mt.get(txKey)
	if itemTransaction != nil {
		return itemTransaction
	}

	itemTransaction = &ItemTransaction[V]{txKey: txKey}
	mt.set(itemTransaction)

	return itemTransaction
}

func (mt *ItemTransactions[V]) get(key TXKey) *ItemTransaction[V] {
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

func (mt *ItemTransactions[V]) remove(key TXKey) {
	mt.Lock()
	defer mt.Unlock()

	index := -1
	for i, itemTransaction := range mt.transactions {
		if itemTransaction.txKey.equals(key) {
			index = i
			break
		}
	}

	if index != -1 {
		mt.transactions = append(mt.transactions[:index], mt.transactions[index+1:]...)
	}
}

type ItemCache[V any] struct {
	vc codec.ValueCodec[V]

	collection collections.Item[V]

	item *Entry[V]

	transactions  *ItemTransactions[V]
	comparer      ValueComparer[V]
	name          string
	currentHeight int64
}

func NewItemCache[V any](sb *collections.SchemaBuilder, prefix []byte, name string, vc codec.ValueCodec[V], caches *Caches, comparer ValueComparer[V]) *ItemCache[V] {
	ic := &ItemCache[V]{
		collection: collections.NewItem(
			sb,
			prefix,
			name,
			vc,
		),
		vc:           vc,
		transactions: &ItemTransactions[V]{},
		comparer:     comparer,
		name:         name,
	}

	*caches = append(*caches, ic)
	return ic
}

func (ic *ItemCache[V]) Initialize(ctx context.Context) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	ic.currentHeight = sdkCtx.BlockHeight()

	if ic.item != nil {
		return nil
	}

	item, has := ic.loadFromStorage(sdkCtx)
	if has {
		ic.item = &item
	}

	return nil
}

func (ic *ItemCache[V]) Get(ctx context.Context) (V, bool) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	txKey := getTXKey(ctx)
	if txKey != nil {
		change := ic.transactions.Get(*txKey)
		if change != nil {
			return getEntry(ctx, change.change)
		}
	}

	if ic.item == nil || sdkCtx.BlockHeight() != ic.currentHeight {
		item, has := ic.loadFromStorage(ctx)
		if has {
			return *item.value, true
		} else {
			var v V
			return v, false
		}
	}

	if ic.item != nil {
		return getEntry(ctx, *ic.item)
	}

	item, has := ic.loadFromStorage(ctx)
	if !has {
		var v V
		return v, false
	}

	ic.item = &item
	return getEntry(ctx, item)
}

func (ic *ItemCache[V]) loadFromStorage(goCtx context.Context) (Entry[V], bool) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	gasMeter := ctx.GasMeter()
	ctx = ctx.WithGasMeter(storetypes.NewInfiniteGasMeter())
	value, err := ic.collection.Get(ctx)
	ctx = ctx.WithGasMeter(gasMeter)

	if err != nil {
		return Entry[V]{}, false
	}

	return Entry[V]{
		value: &value,
		cost:  CalculateReadCostItem(ic.vc, value),
	}, true
}

func (ic *ItemCache[V]) Set(goCtx context.Context, value V) {
	txKey := getTXKey(goCtx)
	if txKey == nil {
		panic("calling Set without initialized cache transaction")
	}

	itemTransaction := ic.transactions.GetCreate(*txKey)
	if itemTransaction.previous == nil {
		itemTransaction.previous = ic.item
	}

	itemTransaction.change = Entry[V]{value: &value}
	ic.transactions.set(itemTransaction)
}

func (ic *ItemCache[V]) Remove(goCtx context.Context) {
	txKey := getTXKey(goCtx)
	if txKey == nil {
		panic("calling Remove without initialized cache transaction")
	}

	itemTransaction := ic.transactions.GetCreate(*txKey)
	if itemTransaction.previous == nil {
		itemTransaction.previous = ic.item
	}

	itemTransaction.change = Entry[V]{}
}

func (ic *ItemCache[V]) Clear(ctx context.Context) {
	txKey := getTXKey(ctx)
	if txKey == nil {
		panic("Clear without cache transaction")
	}

	ic.transactions.remove(*txKey)
}

func (ic *ItemCache[V]) CommitToDB(goCtx context.Context) error {
	txKey := getTXKey(goCtx)
	if txKey == nil {
		panic("CommitToDB without cache transaction")
	}

	itemTransaction := ic.transactions.Get(*txKey)
	if itemTransaction != nil {
		if itemTransaction.change.value != nil {
			if err := ic.collection.Set(goCtx, *itemTransaction.change.value); err != nil {
				return err
			}
		} else {
			if err := ic.collection.Remove(goCtx); err != nil {
				return err
			}
		}
	}

	return nil
}

func (ic *ItemCache[V]) Rollback(goCtx context.Context) {
	txKey := getTXKey(goCtx)
	if txKey == nil {
		panic("Rollback without cache transaction")
	}

	itemTransaction := ic.transactions.Get(*txKey)
	if itemTransaction != nil {
		ctx := sdk.UnwrapSDKContext(goCtx)
		gasMeter := ctx.GasMeter()
		ctx = ctx.WithGasMeter(storetypes.NewInfiniteGasMeter())

		if itemTransaction.previous != nil && itemTransaction.previous.value != nil {
			_ = ic.collection.Set(ctx, *itemTransaction.previous.value)
		} else {
			_ = ic.collection.Remove(ctx)
		}

		ctx = ctx.WithGasMeter(gasMeter)
	}
}

func (ic *ItemCache[V]) CommitToCache(ctx context.Context) {
	txKey := getTXKey(ctx)
	if txKey == nil {
		panic("CommitToCache without cache transaction")
	}

	itemTransaction := ic.transactions.Get(*txKey)
	if itemTransaction != nil {
		if itemTransaction.change.value != nil {
			itemTransaction.change.cost = CalculateReadCostItem(ic.vc, *itemTransaction.change.value)
			ic.item = &itemTransaction.change
		} else {
			ic.item = nil
		}

	}

	ic.transactions.remove(*txKey)
}

func (ic *ItemCache[V]) ClearTransactions() {
	ic.transactions.transactions = nil
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
