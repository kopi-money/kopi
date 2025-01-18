package keeper

import (
	"context"
	"fmt"

	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/kopi-money/kopi/x/dex/types"
)

func (k msgServer) AddLiquidity(ctx context.Context, msg *types.MsgAddLiquidity) (*types.MsgAddLiquidityResponse, error) {
	amount, err := ParseAmount(msg.Amount)
	if err != nil {
		return nil, fmt.Errorf("could not parse amount: %w", err)
	}

	if err = k.precheckTrade(ctx, msg.Creator, msg.Denom, &amount, false); err != nil {
		return nil, fmt.Errorf("could not validate message: %w", err)
	}

	acc, _ := sdk.AccAddressFromBech32(msg.Creator)
	if _, err = k.Keeper.AddLiquidity(ctx, acc, msg.Denom, amount); err != nil {
		return nil, fmt.Errorf("could not add liquidity: %w", err)
	}

	return &types.MsgAddLiquidityResponse{}, nil
}

func (k msgServer) RemoveAllLiquidityForDenom(goCtx context.Context, msg *types.MsgRemoveAllLiquidityForDenom) (*types.Void, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	address, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return nil, types.ErrInvalidAddress
	}

	liq := k.GetLiquidityByAddress(ctx, msg.Denom, msg.Creator)
	if err = k.RemoveLiquidityForAddress(ctx, address, msg.Denom, liq); err != nil {
		return nil, err
	}

	return &types.Void{}, nil
}

func (k Keeper) RemoveAllLiquidityForAddress(ctx context.Context, address, denom string) error {
	amount := math.ZeroInt()

	iterator := k.LiquidityIterator(ctx, denom)
	for iterator.Valid() {
		liq := iterator.GetNext()
		if liq.Address == address {
			amount = amount.Add(liq.Amount)
			k.RemoveLiquidity(ctx, denom, liq.Index)
		}
	}

	addr, err := sdk.AccAddressFromBech32(address)
	if err != nil {
		return fmt.Errorf("invalid address (%v): %w", address, err)
	}

	coins := sdk.NewCoins(sdk.NewCoin(denom, amount))
	if err = k.BankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, addr, coins); err != nil {
		return err
	}

	return nil
}

func (k msgServer) RemoveLiquidity(ctx context.Context, msg *types.MsgRemoveLiquidity) (*types.MsgRemoveLiquidityResponse, error) {
	amount, err := ParseAmount(msg.Amount)
	if err != nil {
		return nil, fmt.Errorf("could not parse amount: %w", err)
	}

	if err = k.precheckTradeWithBalance(ctx, msg.Creator, msg.Denom, &amount, false, false); err != nil {
		return nil, fmt.Errorf("error validating message: %w", err)
	}

	addedAmount := k.GetLiquidityByAddress(ctx, msg.Denom, msg.Creator)
	if addedAmount.LT(amount) {
		return nil, fmt.Errorf("asked amount (%v) is bigger than added amount (%v)", amount.String(), addedAmount.String())
	}

	acc, _ := sdk.AccAddressFromBech32(msg.Creator)
	if err = k.RemoveLiquidityForAddress(ctx, acc, msg.Denom, amount); err != nil {
		return nil, fmt.Errorf("could not remove liquidity for address: %w", err)
	}

	return &types.MsgRemoveLiquidityResponse{}, nil
}

func (k Keeper) RemoveLiquidityForAddress(ctx context.Context, accAddr sdk.AccAddress, denom string, amount math.Int) error {
	removed := math.ZeroInt()
	address := accAddr.String()

	iterator := k.LiquidityIterator(ctx, denom)
	for iterator.Valid() {
		liq := iterator.GetNext()

		if liq.Address == address {
			if liq.Amount.GT(amount) {
				removed = removed.Add(amount)
				liq.Amount = liq.Amount.Sub(amount)
				k.SetLiquidity(ctx, denom, liq)
				amount = math.ZeroInt()
			} else {
				removed = removed.Add(liq.Amount)
				amount = amount.Sub(liq.Amount)
				k.RemoveLiquidity(ctx, denom, liq.Index)
			}
		}

		if amount.IsZero() {
			break
		}
	}

	if amount.IsPositive() {
		return types.ErrNotEnoughFunds
	}

	coins := sdk.NewCoins(sdk.NewCoin(denom, removed))
	if err := k.BankKeeper.SendCoinsFromModuleToAccount(ctx, types.PoolLiquidity, accAddr, coins); err != nil {
		return fmt.Errorf("could not send coins from module to account: %w", err)
	}

	return nil
}
