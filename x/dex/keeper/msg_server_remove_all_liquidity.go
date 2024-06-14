package keeper

import (
	"context"
	"fmt"

	"cosmossdk.io/errors"
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/kopi-money/kopi/x/dex/types"
)

func (k msgServer) RemoveAllLiquidityForDenom(goCtx context.Context, msg *types.MsgRemoveAllLiquidityForDenom) (*types.Void, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	address, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return nil, types.ErrInvalidAddress
	}

	liq := k.GetLiquidityByAddress(ctx, msg.Denom, msg.Creator)
	if err = k.RemoveLiquidityForAddress(ctx, ctx.EventManager(), address, msg.Denom, liq); err != nil {
		return nil, err
	}

	return &types.Void{}, nil
}

func (k Keeper) RemoveAllLiquidityForAddress(ctx context.Context, address, denom string) error {
	amount := math.ZeroInt()

	iterator := k.LiquidityIterator(ctx, denom)
	for iterator.Valid() {
		liq := iterator.GetNext()
		if liq.Address == address && liq.Denom == denom {
			amount = amount.Add(liq.Amount)
			k.RemoveLiquidity(ctx, liq.Denom, liq.Index, liq.Amount)
		}
	}

	//k.updatePair(ctx, nil, denom)

	addr, err := sdk.AccAddressFromBech32(address)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("invalid address (%v)", address))
	}

	coins := sdk.NewCoins(sdk.NewCoin(denom, amount))
	if err = k.BankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, addr, coins); err != nil {
		return err
	}

	return nil
}
