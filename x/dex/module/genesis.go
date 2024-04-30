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
		k.SetRatio(ctx, types.Ratio{
			Denom: elem.Denom,
			Ratio: k.PairRatio(ctx, elem.Denom),
		})
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

	k.SetOrderNextIndex(ctx, genState.OrderNextIndex)
	// this line is used by starport scaffolding # genesis/module/init
	_ = k.SetParams(ctx, genState.Params)

	k.SetLiquidityNextIndex(ctx, genState.LiquidityNextIndex)
}

// ExportGenesis returns the module's exported genesis.
func ExportGenesis(ctx sdk.Context, k keeper.Keeper) *types.GenesisState {
	genesis := types.DefaultGenesis()
	genesis.Params = k.GetParams(ctx)

	genesis.LiquidityList = k.GetAllLiquidity(ctx)
	genesis.LiquidityNextIndex = k.GetLiquidityNextIndex(ctx)
	genesis.LiquidityPairList = k.GetAllLiquidityPair(ctx)
	genesis.LiquidityPairCount = k.GetLiquidityPairCount(ctx)
	genesis.RatioList = k.GetAllRatio(ctx)
	genesis.LiquiditySumList = k.GetAllLiquiditySum(ctx)
	genesis.OrderList = k.GetAllOrders(ctx)
	genesis.OrderNextIndex = k.GetOrderNextIndex(ctx)

	// this line is used by starport scaffolding # genesis/module/export

	return genesis
}
