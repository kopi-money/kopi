package keeper

import (
	"context"
	"cosmossdk.io/math"
	"fmt"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/kopi-money/kopi/x/dex/types"
	reservetypes "github.com/kopi-money/kopi/x/reserve/types"
)

func (k Keeper) liquidityShareExclusionAddresses(ctx context.Context) []string {
	return []string{
		k.AccountKeeper.GetModuleAccount(ctx, types.PoolReserve).GetAddress().String(),
		k.AccountKeeper.GetModuleAccount(ctx, types.PoolFeeIncome).GetAddress().String(),
		k.AccountKeeper.GetModuleAccount(ctx, reservetypes.ModuleName).GetAddress().String(),
	}
}

func (k Keeper) addressIsExcluded(ctx context.Context, address string) bool {
	for _, excludedAddress := range k.liquidityShareExclusionAddresses(ctx) {
		if excludedAddress == address {
			return true
		}
	}

	return false
}

// calcNetLiquidityValue returns the liquidity value in base denom added by users, i.e. excluding the protocol and fees
func (k Keeper) calcNetLiquidityValue(ctx context.Context) (math.LegacyDec, error) {
	excludeAddresses := []string{
		k.AccountKeeper.GetModuleAccount(ctx, types.PoolReserve).GetAddress().String(),
	}

	return k.calcLiquidityValueSum(ctx, excludeAddresses)
}

// calcLiquidityValueSum returns the sum of all added liquidity excluding the liquidity held by the reserve. This
// function is called when starting a new epoch.
func (k Keeper) calcLiquidityValueSum(ctx context.Context, excludeAddresses []string) (math.LegacyDec, error) {
	poolAcc := k.AccountKeeper.GetModuleAccount(ctx, types.PoolLiquidity)

	poolLiquidity := k.BankKeeper.SpendableCoins(ctx, poolAcc.GetAddress())
	for _, excludeAddress := range excludeAddresses {
		coins := k.getLiquidityForAddress(ctx, excludeAddress)
		poolLiquidity = poolLiquidity.Sub(coins...)
	}

	valueSum := math.LegacyZeroDec()
	for _, denom := range k.DenomKeeper.Denoms(ctx) {
		valueBase, err := k.DenomKeeper.GetValueInBase(ctx, denom, poolLiquidity.AmountOf(denom).ToLegacyDec())
		if err != nil {
			return math.LegacyDec{}, fmt.Errorf("convert to base (%v): %w", denom, err)
		}

		valueSum = valueSum.Add(valueBase)
	}

	return valueSum, nil
}

func (k Keeper) calculateLiquidityValueForPosition(ctx context.Context, positionIndex uint64) (math.LegacyDec, error) {
	valueSum := math.LegacyZeroDec()
	for _, denom := range k.DenomKeeper.Denoms(ctx) {
		value := k.GetLiquidityByPositionIndex(ctx, denom, positionIndex)
		valueBase, err := k.DenomKeeper.GetValueInBase(ctx, denom, value.ToLegacyDec())
		if err != nil {
			return math.LegacyDec{}, fmt.Errorf("convert to base (%v): %w", denom, err)
		}

		valueSum = valueSum.Add(valueBase)
	}

	return valueSum, nil
}

func (k Keeper) getBalance(ctx context.Context, moduleAcc string) sdk.Coins {
	acc := k.AccountKeeper.GetModuleAccount(ctx, moduleAcc)
	return k.BankKeeper.SpendableCoins(ctx, acc.GetAddress())
}

func (k Keeper) exportLiquidityPositions(ctx context.Context) (list []types.GenesisLiquidityPositions) {
	addresses, _ := k.liquidityPositions.OuterKeys(ctx)
	for _, address := range addresses {
		iterator := k.liquidityPositions.Iterator(ctx, nil, address)
		for iterator.Valid() {
			keyValue := iterator.GetNextKeyValue()

			list = append(list, types.GenesisLiquidityPositions{
				Address:       address,
				PositionIndex: keyValue.Key(),
				AutoCompound:  keyValue.Value().Value().AutoCompound,
				CreatedAt:     keyValue.Value().Value().CreatedAt,
			})
		}
	}

	return
}

func (k Keeper) exportLiquidityAddressSums(ctx context.Context) (list []types.GenesisLiquidityAddressSum) {
	addresses, _ := k.liquidityAddressSum.OuterKeys(ctx)
	for _, address := range addresses {
		iterator := k.liquidityAddressSum.Iterator(ctx, nil, address)
		for iterator.Valid() {
			keyValue := iterator.GetNextKeyValue()

			list = append(list, types.GenesisLiquidityAddressSum{
				Address: address,
				Denom:   keyValue.Key(),
				Sum:     keyValue.Value().Value().Sum,
			})
		}
	}

	return
}
