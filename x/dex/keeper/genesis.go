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
	genesis.RatioList = k.GetAllRatio(ctx)
	genesis.OrderNextIndex = oni.Next

	orderIterator := k.orders.Iterator(ctx, nil, nil)
	for orderIterator.Valid() {
		genesis.OrderList = append(genesis.OrderList, orderIterator.GetNext())
	}

	return genesis
}

func (k Keeper) ExportGenesisBytes(ctx context.Context) []byte {
	return k.cdc.MustMarshal(k.ExportGenesis(ctx))
}
