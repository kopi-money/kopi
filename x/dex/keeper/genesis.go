package keeper

import (
	"context"
	"github.com/kopi-money/kopi/x/dex/types"
)

//PrefixMovingLiquidity     = collections.NewPrefix(14)
//PrefixLiquidityAddressSum = collections.NewPrefix(15)

func (k Keeper) ExportGenesis(ctx context.Context) *types.GenesisState {
	genesis := types.DefaultGenesis()
	genesis.Params = k.GetParams(ctx)

	genesis.LiquidityEntries = k.exportLiquidityEntries(ctx)
	genesis.LiquidityPositions = k.exportLiquidityPositions(ctx)
	genesis.LiquidityAddressSums = k.exportLiquidityAddressSums(ctx)
	genesis.Orders = k.exportOrdersToGenesis(ctx)
	genesis.WalletTradeAmounts = k.exportTradeAmountsToGenesis(ctx)
	genesis.EpochShares = k.exportEpochSharesToGenesis(ctx)
	genesis.EpochLeftovers = k.exportEpochLeftoversToGenesis(ctx)
	genesis.EpochSharesSum, _ = k.epochSharesSum.Get(ctx)
	genesis.EpochStartTime, _ = k.epochStartTime.Get(ctx)
	genesis.MovingLiquidity = k.exportMovingLiquidity(ctx)

	genesis.LiquidityNextIndex, _ = k.liquidityEntriesNextIndex.Get(ctx)
	genesis.LiquidityPositionsNextIndex, _ = k.liquidityPositionNextIndex.Get(ctx)
	genesis.OrderNextIndex = k.GetOrderNextIndex(ctx)

	return genesis
}

func (k Keeper) ExportGenesisBytes(ctx context.Context) []byte {
	return k.cdc.MustMarshal(k.ExportGenesis(ctx))
}

func (k Keeper) InitGenesis(ctx context.Context, gs types.GenesisState) error {
	if err := k.SetParams(ctx, gs.Params); err != nil {
		return err
	}

	for _, liquidityEntry := range gs.LiquidityEntries {
		k.liquidityEntries.Set(ctx, liquidityEntry.Denom, liquidityEntry.Index, types.Liquidity{
			Index:         liquidityEntry.Index,
			Address:       liquidityEntry.Address,
			Amount:        liquidityEntry.Amount,
			PositionIndex: liquidityEntry.PositionIndex,
		})
	}

	for _, liquidityPosition := range gs.LiquidityPositions {
		k.liquidityPositions.Set(ctx, liquidityPosition.Address, liquidityPosition.PositionIndex, types.LiquidityPosition{
			AutoCompound: liquidityPosition.AutoCompound,
			CreatedAt:    liquidityPosition.CreatedAt,
		})
	}

	for _, liquidityAddressSum := range gs.LiquidityAddressSums {
		k.liquidityAddressSum.Set(ctx, liquidityAddressSum.Address, liquidityAddressSum.Denom, types.LiquiditySum{
			Sum: liquidityAddressSum.Sum,
		})
	}

	for _, order := range gs.Orders {
		k.orders.Set(ctx, order.Index, order)
	}

	for _, tradeAmount := range gs.WalletTradeAmounts {
		k.tradeAmounts.Set(ctx, tradeAmount.Address, types.WalletTradeAmount{
			Amount: tradeAmount.Amount,
		})
	}

	for _, epochShare := range gs.EpochShares {
		k.epochShares.Set(ctx, epochShare.Address, epochShare.PositionIndex, types.EpochShares{
			Shares: epochShare.Shares,
		})
	}

	for _, epochLeftovers := range gs.EpochLeftovers {
		k.epochLeftovers.Set(ctx, epochLeftovers.Address, epochLeftovers.PositionIndex, types.EpochLeftovers{
			Leftovers: epochLeftovers.Leftovers,
		})
	}

	k.epochSharesSum.Set(ctx, gs.EpochSharesSum)
	k.epochStartTime.Set(ctx, gs.EpochStartTime)

	for _, movingLiquidity := range gs.MovingLiquidity {
		k.movingLiquidity.Set(ctx, movingLiquidity.Denom, types.MovingLiquidity{
			Amount: movingLiquidity.Amount,
		})
	}

	k.liquidityEntriesNextIndex.Set(ctx, gs.LiquidityNextIndex)
	k.liquidityPositionNextIndex.Set(ctx, gs.LiquidityPositionsNextIndex)
	k.ordersNextIndex.Set(ctx, gs.OrderNextIndex)

	// this line is used by starport scaffolding # genesis/module/init

	return nil
}
