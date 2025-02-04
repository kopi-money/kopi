package dex

import (
	"context"

	"github.com/cosmos/cosmos-sdk/cache"

	"github.com/kopi-money/kopi/x/dex/keeper"
	"github.com/kopi-money/kopi/x/dex/types"
)

// InitGenesis initializes the module's state from a provided genesis state.
func InitGenesis(ctx context.Context, k keeper.Keeper, genState types.GenesisState) {
	if err := cache.Transact(ctx, func(innerCtx context.Context) error {
		// Set all the liquidity
		for _, denomLiquidity := range genState.LiquidityList {
			for _, entry := range denomLiquidity.Entries {
				k.SetLiquidity(innerCtx, denomLiquidity.Denom, entry)
			}
		}

		// Set all the order
		for _, elem := range genState.OrderList {
			k.SetOrder(innerCtx, elem)
		}

		oni := types.OrderNextIndex{Next: genState.OrderNextIndex}
		k.SetOrderNextIndex(innerCtx, oni.Next)
		k.SetLiquidityEntryNextIndex(innerCtx, genState.LiquidityNextIndex)
		// this line is used by starport scaffolding # genesis/module/init

		if err := k.SetParams(innerCtx, genState.Params); err != nil {
			return err
		}

		return nil
	}); err != nil {
		panic(err)
	}
}

// ExportGenesis returns the module's exported genesis.
func ExportGenesis(ctx context.Context, k keeper.Keeper) *types.GenesisState {
	// this line is used by starport scaffolding # genesis/module/export
	return k.ExportGenesis(ctx)
}
