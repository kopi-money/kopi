package keeper

import (
	"context"

	"cosmossdk.io/collections"
	"cosmossdk.io/math"
	"github.com/kopi-money/kopi/cache"
	"github.com/kopi-money/kopi/x/tokenfactory/types"
)

func (k Keeper) GetGenesisLiquidityPools(ctx context.Context) (pools []types.GenesisLiquidityPool) {
	iterator := k.liquidityPools.Iterator(ctx, nil)
	for iterator.Valid() {
		keyValue := iterator.GetNextKeyValue()
		pool := keyValue.Value().Value()

		pools = append(pools, types.GenesisLiquidityPool{
			FactoryDenom:       keyValue.Key(),
			KCoin:              pool.KCoin,
			UnlockInSeconds:    pool.UnlockInSeconds,
			PoolFee:            pool.PoolFee,
			FactoryDenomAmount: pool.FactoryDenomAmount,
			KCoinAmount:        pool.KCoinAmount,
			Shares:             k.getGenesisLiquidityShares(ctx, keyValue.Key()),
			Unlockings:         k.getPoolUnlockings(ctx, keyValue.Key()),
		})
	}

	return
}

func (k Keeper) getGenesisLiquidityShares(ctx context.Context, factoryDenom string) (list []*types.GenesisProviderShare) {
	iterator := k.LiquidityShareIterator(ctx, factoryDenom)
	for iterator.Valid() {
		keyValue := iterator.GetNextKeyValue()

		list = append(list, &types.GenesisProviderShare{
			Address: keyValue.Key(),
			Share:   keyValue.Value().Value().Share,
		})
	}

	return
}

func (k Keeper) SetGenesisLiquidityPool(ctx context.Context, pool types.GenesisLiquidityPool) {
	k.liquidityPools.Set(ctx, pool.FactoryDenom, types.LiquidityPool{
		KCoin:              pool.KCoin,
		UnlockInSeconds:    pool.UnlockInSeconds,
		PoolFee:            pool.PoolFee,
		FactoryDenomAmount: pool.FactoryDenomAmount,
		KCoinAmount:        pool.KCoinAmount,
	})

	for _, share := range pool.Shares {
		s := types.ProviderShare{Share: share.Share}
		k.liquidityProviderShares.Set(ctx, pool.FactoryDenom, share.Address, s)
	}

	for _, unlocking := range pool.Unlockings {
		k.liquidityUnlockings.Set(ctx, unlocking.Index, *unlocking)
	}
}

func (k Keeper) GetLiquidityPool(ctx context.Context, factoryDenomHash string) (types.LiquidityPool, bool) {
	return k.liquidityPools.Get(ctx, factoryDenomHash)
}

func (k Keeper) SetLiquidityPool(ctx context.Context, factoryDenomHash string, liquidityPool types.LiquidityPool) {
	k.liquidityPools.Set(ctx, factoryDenomHash, liquidityPool)
}

// getPoolRatio returns the ratio in the form of "One factory denom unit represents x kcoin denom units"
func getPoolRatio(pool types.LiquidityPool) math.LegacyDec {
	return pool.KCoinAmount.ToLegacyDec().Quo(pool.FactoryDenomAmount.ToLegacyDec())
}

func (k Keeper) LiquidityShareIterator(ctx context.Context, denom string) cache.Iterator[string, types.ProviderShare] {
	rng := collections.NewPrefixedPairRange[string, string](denom)
	return k.liquidityProviderShares.Iterator(ctx, rng, denom)
}

func (k Keeper) updateLiquidityShare(ctx context.Context, factoryDenom types.FactoryDenom, pool types.LiquidityPool, addedAmount math.Int, addedAddress string) error {
	var (
		iterator      = k.liquidityProviderShares.Iterator(ctx, nil, factoryDenom.FullName)
		keyValue      cache.KeyValue[string, cache.Entry[types.ProviderShare]]
		address       string
		providerShare types.ProviderShare
	)

	providers := make(map[string]math.LegacyDec)
	providers[addedAddress] = addedAmount.ToLegacyDec()
	sum := addedAmount.ToLegacyDec()

	for iterator.Valid() {
		keyValue = iterator.GetNextKeyValue()
		address = keyValue.Key()
		providerShare = *keyValue.Value().Value()

		amount := pool.FactoryDenomAmount.ToLegacyDec().Mul(providerShare.Share)
		sum = sum.Add(amount)

		if address == addedAddress {
			amount = amount.Add(addedAmount.ToLegacyDec())
			if amount.IsNegative() {
				return types.ErrNegativeLiquidity
			}
		}

		providers[address] = amount
	}

	for providerAddress, providerAmount := range providers {
		if providerAmount.IsZero() {
			k.liquidityProviderShares.Remove(ctx, factoryDenom.FullName, providerAddress)
		} else {
			k.liquidityProviderShares.Set(ctx, factoryDenom.FullName, providerAddress, types.ProviderShare{
				Share: providerAmount.Quo(sum),
			})
		}
	}

	return nil
}

func (k Keeper) getLiquidityShare(ctx context.Context, factoryDenom, address string) math.LegacyDec {
	share, has := k.liquidityProviderShares.Get(ctx, factoryDenom, address)
	if !has {
		return math.LegacyZeroDec()
	}

	return share.Share
}
