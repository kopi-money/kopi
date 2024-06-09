package keeper

import (
	"context"
	"github.com/kopi-money/kopi/x/dex/types"
)

func (k Keeper) ExportGenesis(ctx context.Context) *types.GenesisState {
	genesis := types.DefaultGenesis()
	genesis.Params = k.GetParams(ctx)

	oni, _ := k.GetOrderNextIndex(ctx)

	genesis.LiquidityList = k.GetAllLiquidity(ctx)
	genesis.LiquidityNextIndex, _ = k.liquidityEntriesNextIndex.Get(ctx)
	genesis.LiquidityPairList = k.GetAllLiquidityPair(ctx)
	genesis.RatioList = k.GetAllRatio(ctx)
	genesis.OrderNextIndex = oni.Next

	liquiditySumIterator := k.liquiditySums.Iterator(ctx)
	for liquiditySumIterator.Valid() {
		genesis.LiquiditySumList = append(genesis.LiquiditySumList, liquiditySumIterator.GetNext())
	}

	orderIterator := k.orders.Iterator(ctx)
	for orderIterator.Valid() {
		genesis.OrderList = append(genesis.OrderList, orderIterator.GetNext())
	}

	return genesis
}

func (k Keeper) ExportGenesisBytes(ctx context.Context) []byte {
	return k.cdc.MustMarshal(k.ExportGenesis(ctx))
}
