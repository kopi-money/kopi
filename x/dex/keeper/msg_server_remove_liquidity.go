package keeper

import (
	"context"
	"cosmossdk.io/errors"
	"strconv"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/kopi-money/kopi/x/dex/types"
)

func (k msgServer) RemoveLiquidity(goCtx context.Context, msg *types.MsgRemoveLiquidity) (*types.MsgRemoveLiquidityResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	amount, err := parseAmount(msg.Amount)
	if err != nil {
		return nil, errors.Wrap(err, "could not parse amount")
	}

	address, err := k.validateMsg(ctx, msg.Creator, msg.Denom, amount)
	if err != nil {
		return nil, errors.Wrap(err, "error validating message")
	}

	if err = k.RemoveLiquidityForAddress(ctx, ctx.EventManager(), address, msg.Denom, amount); err != nil {
		return nil, errors.Wrap(err, "could not remove liquidity for address")
	}

	return &types.MsgRemoveLiquidityResponse{}, nil
}

func (k Keeper) RemoveAllLiquidityForModule(ctx context.Context, eventManager sdk.EventManagerI, denom, module string) error {
	address := k.AccountKeeper.GetModuleAccount(ctx, module).GetAddress()
	removed := k.removeAllLiquidityForAddress(ctx, eventManager, denom, address.String())

	coins := sdk.NewCoins(sdk.NewCoin(denom, removed))
	if err := k.BankKeeper.SendCoinsFromModuleToModule(ctx, types.PoolLiquidity, module, coins); err != nil {
		return err
	}

	return nil
}

func (k Keeper) RemoveLiquidityForAddress(ctx context.Context, eventManager sdk.EventManagerI, accAddr sdk.AccAddress, denom string, amount math.Int) error {
	removed := math.ZeroInt()
	address := accAddr.String()

	iterator := k.LiquidityIterator(ctx, denom)
	for iterator.Valid() {
		liq := iterator.GetNext()

		var amountRemoved math.Int
		if liq.Address == address && liq.Denom == denom {
			if liq.Amount.GT(amount) {
				amountRemoved = amount
				removed = removed.Add(amount)
				liq.Amount = liq.Amount.Sub(amount)
				k.SetLiquidity(ctx, liq)
				amount = math.ZeroInt()
			} else {
				amountRemoved = liq.Amount
				removed = removed.Add(liq.Amount)
				amount = amount.Sub(liq.Amount)
				k.RemoveLiquidity(ctx, liq.Denom, liq.Index, liq.Amount)
			}

			eventManager.EmitEvent(
				sdk.NewEvent(
					"liquidity_removed",
					sdk.Attribute{Key: "index", Value: strconv.Itoa(int(liq.Index))},
					sdk.Attribute{Key: "denom", Value: denom},
					sdk.Attribute{Key: "amount", Value: amountRemoved.String()},
					sdk.Attribute{Key: "address", Value: address},
				),
			)
		}

		if amount.Equal(math.ZeroInt()) {
			break
		}
	}

	coins := sdk.NewCoins(sdk.NewCoin(denom, removed))
	if err := k.BankKeeper.SendCoinsFromModuleToAccount(ctx, types.PoolLiquidity, accAddr, coins); err != nil {
		return err
	}

	if amount.GT(math.ZeroInt()) {
		return types.ErrNotEnoughFunds
	}

	//if denom != utils.BaseCurrency {
	//	k.updatePair(ctx, nil, denom)
	//} else {
	//	k.updatePairs(ctx, nil)
	//}

	return nil
}

func (k Keeper) removeAllLiquidityForAddress(ctx context.Context, eventManager sdk.EventManagerI, denom, address string) math.Int {
	removed := math.ZeroInt()

	iterator := k.LiquidityIterator(ctx, denom)
	for iterator.Valid() {
		liq := iterator.GetNext()
		if liq.Address == address && liq.Denom == denom {
			k.RemoveLiquidity(ctx, liq.Denom, liq.Index, liq.Amount)
			removed = removed.Add(liq.Amount)
		}

		eventManager.EmitEvent(
			sdk.NewEvent(
				"liquidity_removed",
				sdk.Attribute{Key: "index", Value: strconv.Itoa(int(liq.Index))},
				sdk.Attribute{Key: "denom", Value: denom},
				sdk.Attribute{Key: "amount", Value: liq.Amount.String()},
				sdk.Attribute{Key: "address", Value: address},
			),
		)
	}

	//if denom != utils.BaseCurrency {
	//	k.updatePair(ctx, nil, denom)
	//} else {
	//	k.updatePairs(ctx, nil)
	//}

	return removed
}
