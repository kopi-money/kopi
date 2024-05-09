package keeper

import (
	"context"
	"strconv"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/kopi-money/kopi/utils"
	"github.com/kopi-money/kopi/x/dex/types"
)

func (k msgServer) RemoveLiquidity(goCtx context.Context, msg *types.MsgRemoveLiquidity) (*types.MsgRemoveLiquidityResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	amount, err := parseAmount(msg.Amount)
	if err != nil {
		return nil, err
	}

	address, err := k.validateMsg(ctx, msg.Creator, msg.Denom, amount)
	if err != nil {
		return nil, err
	}

	if err = k.RemoveLiquidityForAddress(ctx, ctx.EventManager(), msg.Denom, address, amount); err != nil {
		return nil, err
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

func (k Keeper) RemoveLiquidityForModule(ctx context.Context, eventManager sdk.EventManagerI, denom, module string, amount math.Int) error {
	address := k.AccountKeeper.GetModuleAccount(ctx, module).GetAddress()
	removed, err := k.removeLiquidityForAddress(ctx, eventManager, denom, address.String(), amount)
	if err != nil {
		return err
	}

	coins := sdk.NewCoins(sdk.NewCoin(denom, removed))
	if err = k.BankKeeper.SendCoinsFromModuleToModule(ctx, types.PoolLiquidity, module, coins); err != nil {
		return err
	}

	return nil
}

func (k Keeper) RemoveLiquidityForAddress(ctx context.Context, eventManager sdk.EventManagerI, denom string, addr sdk.AccAddress, amount math.Int) error {
	removed, err := k.removeLiquidityForAddress(ctx, eventManager, denom, addr.String(), amount)
	if err != nil {
		return err
	}

	coins := sdk.NewCoins(sdk.NewCoin(denom, removed))
	if err = k.BankKeeper.SendCoinsFromModuleToAccount(ctx, types.PoolLiquidity, addr, coins); err != nil {
		return err
	}

	return nil
}

func (k Keeper) removeLiquidityForAddress(ctx context.Context, eventManager sdk.EventManagerI, denom, address string, amount math.Int) (math.Int, error) {
	removed := math.ZeroInt()

	iterator := k.LiquidityIterator(ctx, denom)
	for ; iterator.Valid(); iterator.Next() {
		var liq types.Liquidity
		k.cdc.MustUnmarshal(iterator.Value(), &liq)

		var amountRemoved math.Int
		if liq.Address == address && liq.Denom == denom {
			if liq.Amount.GT(amount) {
				amountRemoved = amount
				removed = removed.Add(amount)
				liq.Amount = liq.Amount.Sub(amount)
				k.SetLiquidity(ctx, &liq, amount.Neg())
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

	if amount.GT(math.ZeroInt()) {
		return removed, types.ErrNotEnoughFunds
	}

	if denom != utils.BaseCurrency {
		k.updatePair(ctx, denom)
	} else {
		k.updatePairs(ctx)
	}

	return removed, nil
}

func (k Keeper) removeAllLiquidityForAddress(ctx context.Context, eventManager sdk.EventManagerI, denom, address string) math.Int {
	removed := math.ZeroInt()

	iterator := k.LiquidityIterator(ctx, denom)
	for ; iterator.Valid(); iterator.Next() {
		var liq types.Liquidity
		k.cdc.MustUnmarshal(iterator.Value(), &liq)

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

	if denom != utils.BaseCurrency {
		k.updatePair(ctx, denom)
	} else {
		k.updatePairs(ctx)
	}

	return removed
}
