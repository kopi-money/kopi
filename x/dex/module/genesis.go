package dex

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/kopi-money/kopi/x/dex/keeper"
	"github.com/kopi-money/kopi/x/dex/types"
)

// InitGenesis initializes the module's state from a provided genesis state.
func InitGenesis(ctx sdk.Context, k keeper.Keeper, genState types.GenesisState) {
	// Set all the liquidity
	for _, elem := range genState.LiquidityList {
		k.SetLiquidity(ctx, elem, elem.Amount)
	}
	// Set all the liquidityPair
	for _, elem := range genState.LiquidityPairList {
		k.SetLiquidityPair(ctx, elem)
	}

	for _, elem := range genState.LiquidityPairList {
		k.SetRatio(ctx, types.Ratio{elem.Denom, k.PairRatio(ctx, elem.Denom)})
	}

	// Set liquidityPair count
	k.SetLiquidityPairCount(ctx, genState.LiquidityPairCount)

	// Set all the liquiditySum
	for _, elem := range genState.LiquiditySumList {
		k.SetLiquiditySum(ctx, elem)
	}

	// Set all the order
	for _, elem := range genState.OrderList {
		k.SetOrder(ctx, elem)
	}

	oni := types.OrderNextIndex{Next: genState.OrderNextIndex}
	k.SetOrderNextIndex(ctx, oni)
	// this line is used by starport scaffolding # genesis/module/init

	if err := k.SetParams(ctx, genState.Params); err != nil {
		panic(err)
	}

	lni := types.LiquidityNextIndex{Next: genState.LiquidityNextIndex}
	k.SetLiquidityNextIndex(ctx, lni)
}

// ExportGenesis returns the module's exported genesis.
func ExportGenesis(ctx sdk.Context, k keeper.Keeper) *types.GenesisState {
	// this line is used by starport scaffolding # genesis/module/export
	return k.ExportGenesis(ctx)
}
