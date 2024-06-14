package dex

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/kopi-money/kopi/cache"

	"github.com/kopi-money/kopi/x/dex/keeper"
	"github.com/kopi-money/kopi/x/dex/types"
)

// InitGenesis initializes the module's state from a provided genesis state.
func InitGenesis(goCtx sdk.Context, k keeper.Keeper, genState types.GenesisState) {
	if err := cache.Transact(goCtx, func(ctx sdk.Context) error {
		// Set all the liquidity
		for _, elem := range genState.LiquidityList {
			k.SetLiquidity(ctx, elem)
		}

		for _, elem := range genState.RatioList {
			k.SetRatio(ctx, elem)
			//k.SetLiquidityPair(ctx, k.CreateLiquidityPair(ctx, elem))
		}

		// Set all the order
		for _, elem := range genState.OrderList {
			k.SetOrder(ctx, elem)
		}

		oni := types.OrderNextIndex{Next: genState.OrderNextIndex}
		k.SetOrderNextIndex(ctx, oni)
		// this line is used by starport scaffolding # genesis/module/init

		if err := k.SetParams(ctx, genState.Params); err != nil {
			return err
		}

		k.SetLiquidityEntryNextIndex(ctx, genState.LiquidityNextIndex)

		return nil
	}); err != nil {
		panic(err)
	}
}

// ExportGenesis returns the module's exported genesis.
func ExportGenesis(ctx sdk.Context, k keeper.Keeper) *types.GenesisState {
	// this line is used by starport scaffolding # genesis/module/export
	return k.ExportGenesis(ctx)
}
