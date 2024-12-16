package keeper

import (
	"context"
	"fmt"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/kopi-money/kopi/cache"
	"github.com/kopi-money/kopi/x/tokenfactory/types"
)

func (k Keeper) SetLiquidityUnlocking(ctx context.Context, unlocking types.LiquidityUnlocking) uint64 {
	if unlocking.Index == 0 {
		nextIndex, _ := k.liquidityUnlockingsNextIndex.Get(ctx)
		nextIndex += 1
		k.liquidityUnlockingsNextIndex.Set(ctx, nextIndex)
		unlocking.Index = nextIndex
	}

	k.liquidityUnlockings.Set(ctx, unlocking.Index, unlocking)
	return unlocking.Index
}

func (k Keeper) LiquidityUnlockingsIterator(ctx context.Context) cache.Iterator[uint64, types.LiquidityUnlocking] {
	return k.liquidityUnlockings.Iterator(ctx, nil)
}

func (k Keeper) GetUnlockings(ctx context.Context, factoryDenomHash, address string) (list []types.LiquidityUnlocking) {
	iterator := k.LiquidityUnlockingsIterator(ctx)
	for iterator.Valid() {
		unlocking := iterator.GetNext()
		if unlocking.FactoryDenomHash == factoryDenomHash && unlocking.Address == address {
			list = append(list, unlocking)
		}
	}

	return
}

func (k Keeper) HandleUnlockings(ctx context.Context, now time.Time) {
	iterator := k.LiquidityUnlockingsIterator(ctx)
	poolUnlockings := make(map[string]uint64)

	for iterator.Valid() {
		unlocking := iterator.GetNext()
		unlockSeconds := k.getPoolUnlocking(ctx, unlocking.FactoryDenomHash, &poolUnlockings)

		unlocksAt := unlocking.CreatedAt.Add(time.Duration(unlockSeconds) * time.Second)
		if now.Before(unlocksAt) {
			if err := k.HandleUnlocking(ctx, &unlocking); err != nil {
				k.Logger().Error(fmt.Sprintf("could not handle unlocking: %v", err))
			}
		}
	}
}

func (k Keeper) getPoolUnlockings(ctx context.Context, factoryDenomHash string) (unlockings []*types.LiquidityUnlocking) {
	iterator := k.liquidityUnlockings.Iterator(ctx, nil)
	for iterator.Valid() {
		unlocking := iterator.GetNext()

		if unlocking.FactoryDenomHash == factoryDenomHash {
			unlockings = append(unlockings, &unlocking)
		}
	}

	return unlockings
}

func (k Keeper) getPoolUnlocking(ctx context.Context, factoryDenomHash string, poolUnlockings *map[string]uint64) uint64 {
	unlockBlocks, has := (*poolUnlockings)[factoryDenomHash]
	if !has {
		pool, _ := k.liquidityPools.Get(ctx, factoryDenomHash)
		(*poolUnlockings)[factoryDenomHash] = pool.UnlockInSeconds
		unlockBlocks = pool.UnlockInSeconds
	}

	return unlockBlocks
}

func (k Keeper) HandleUnlocking(ctx context.Context, unlocking *types.LiquidityUnlocking) error {
	coins := sdk.NewCoins(
		sdk.NewCoin(unlocking.FactoryDenomHash, unlocking.FactoryDenomAmount),
		sdk.NewCoin(unlocking.KCoin, unlocking.KCoinAmount),
	)

	acc, _ := sdk.AccAddressFromBech32(unlocking.Address)
	if err := k.BankKeeper.SendCoinsFromModuleToAccount(ctx, types.PoolUnlocking, acc, coins); err != nil {
		return fmt.Errorf("could not send could from unlocking pool to account: %w", err)
	}

	k.liquidityUnlockings.Remove(ctx, unlocking.Index)
	return nil
}
