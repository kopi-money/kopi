package keeper

import (
	"context"
	"github.com/kopi-money/kopi/x/dex/types"
)

func (k Keeper) ExportGenesis(ctx context.Context) *types.GenesisState {
	genesis := types.DefaultGenesis()
	genesis.Params = k.GetParams(ctx)

	lni, _ := k.GetLiquidityNextIndex(ctx)
	oni, _ := k.GetOrderNextIndex(ctx)

	genesis.LiquidityList = k.GetAllLiquidity(ctx)
	genesis.LiquidityNextIndex = lni.Next
	genesis.LiquidityPairList = k.GetAllLiquidityPair(ctx)
	genesis.LiquidityPairCount = k.GetLiquidityPairCount(ctx)
	genesis.RatioList = k.GetAllRatio(ctx)
	genesis.LiquiditySumList = k.GetAllLiquiditySum(ctx)
	genesis.OrderList = k.GetAllOrders(ctx)
	genesis.OrderNextIndex = oni.Next

	return genesis
}

func (k Keeper) ExportGenesisBytes(ctx context.Context) []byte {
	return k.cdc.MustMarshal(k.ExportGenesis(ctx))
}
