package keeper

import (
	"context"
	"fmt"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"cosmossdk.io/collections"
	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/cache"
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
			CreatedAt:          pool.CreatedAt,
			PoolFee:            pool.PoolFee,
			FactoryDenomAmount: pool.FactoryDenomAmount,
			KCoinAmount:        pool.KCoinAmount,
			Shares:             k.getGenesisLiquidityShares(ctx, keyValue.Key()),
			Unlockings:         k.getPoolUnlockings(ctx, keyValue.Key()),
		})
	}

	return
}

func (k Keeper) getGenesisLiquidityShares(ctx context.Context, factoryDenom string) (list []types.GenesisProviderShare) {
	iterator := k.LiquidityShareIterator(ctx, factoryDenom)
	for iterator.Valid() {
		keyValue := iterator.GetNextKeyValue()

		list = append(list, types.GenesisProviderShare{
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
		CreatedAt:          pool.CreatedAt,
		PoolFee:            pool.PoolFee,
		FactoryDenomAmount: pool.FactoryDenomAmount,
		KCoinAmount:        pool.KCoinAmount,
	})

	for _, share := range pool.Shares {
		s := types.ProviderShare{Share: share.Share}
		k.liquidityProviderShares.Set(ctx, pool.FactoryDenom, share.Address, s)
	}

	for _, unlocking := range pool.Unlockings {
		k.liquidityUnlockings.Set(ctx, unlocking.Index, unlocking)
	}
}

func (k Keeper) GetLiquidityPool(ctx context.Context, factoryDenomHash string) (types.LiquidityPool, bool) {
	return k.liquidityPools.Get(ctx, factoryDenomHash)
}

func (k Keeper) SetLiquidityPool(ctx context.Context, factoryDenomHash string, liquidityPool types.LiquidityPool) {
	k.liquidityPools.Set(ctx, factoryDenomHash, liquidityPool)
}

func (k Keeper) LiquidityShareIterator(ctx context.Context, denom string) cache.Iterator[string, types.ProviderShare] {
	rng := collections.NewPrefixedPairRange[string, string](denom)
	return k.liquidityProviderShares.Iterator(ctx, rng, denom)
}

func (k Keeper) updateLiquidityShare(ctx context.Context, factoryDenom types.FactoryDenom, totalAmount, addedAmount math.LegacyDec, addedAddress string) error {
	var (
		iterator      = k.liquidityProviderShares.Iterator(ctx, nil, factoryDenom.FullName)
		sum           = addedAmount
		keyValue      cache.KeyValue[string, cache.Entry[types.ProviderShare]]
		address       string
		providerShare types.ProviderShare
	)

	providers := make(map[string]math.LegacyDec)
	providers[addedAddress] = addedAmount

	for iterator.Valid() {
		keyValue = iterator.GetNextKeyValue()
		address = keyValue.Key()
		providerShare = *keyValue.Value().Value()

		amount := totalAmount.Mul(providerShare.Share)
		sum = sum.Add(amount)

		if address == addedAddress {
			amount = amount.Add(addedAmount)
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
			if !sum.IsPositive() {
				return fmt.Errorf("sum is not positive")
			}

			k.liquidityProviderShares.Set(ctx, factoryDenom.FullName, providerAddress, types.ProviderShare{
				Share: providerAmount.Quo(sum), // C
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

func (k Keeper) getLiquidity(ctx context.Context, factoryDenom, address string) (math.Int, math.Int, error) {
	pool, has := k.liquidityPools.Get(ctx, factoryDenom)
	if !has {
		return math.Int{}, math.Int{}, types.ErrPoolDoesNotExist
	}

	share, has := k.liquidityProviderShares.Get(ctx, factoryDenom, address)
	if !has {
		return math.ZeroInt(), math.ZeroInt(), nil
	}

	amountKCoin := pool.KCoinAmount.ToLegacyDec().Mul(share.Share).TruncateInt()
	amountFactory := pool.FactoryDenomAmount.ToLegacyDec().Mul(share.Share).TruncateInt()
	return amountKCoin, amountFactory, nil
}

func (k Keeper) CheckPoolSizes(ctx context.Context) error {
	if sdk.UnwrapSDKContext(ctx).BlockHeight()%1000 != 0 {
		return nil
	}

	blocktime := sdk.UnwrapSDKContext(ctx).BlockTime()
	minimumSizes := k.DenomKeeper.MinimumFactoryPoolSizes(ctx)

	iterator := k.liquidityPools.Iterator(ctx, nil)
	for iterator.Valid() {
		keyValue := iterator.GetNextKeyValue()
		pool := keyValue.Value().Value()

		poolValue, err := pool.GetPoolValue()
		if err != nil {
			return fmt.Errorf("get pool value: %w", err)
		}

		if minimumSize, has := minimumSizes[pool.KCoin]; has {
			if poolValue.LT(minimumSize.ToLegacyDec()) {
				pool.ThresholdCrossed = nil
			} else if pool.ThresholdCrossed == nil {
				pool.ThresholdCrossed = &blocktime
			}
		}

		k.SetLiquidityPool(ctx, keyValue.Key(), *pool)
	}

	return nil
}
