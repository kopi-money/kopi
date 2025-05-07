package keeper

import (
	"context"
	"fmt"
	"github.com/kopi-money/kopi/trading"
	"strconv"

	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/kopi-money/kopi/x/dex/types"
)

func (k msgServer) AddLiquidity(ctx context.Context, msg *types.MsgAddLiquidity) (*types.MsgAddLiquidityResponse, error) {
	amount, err := trading.ParseAmount(msg.Amount)
	if err != nil {
		return nil, fmt.Errorf("parse amount: %w", err)
	}

	if err = k.precheckTrade(ctx, msg.Creator, msg.Denom, &amount, false); err != nil {
		return nil, fmt.Errorf("validate message: %w", err)
	}

	acc, _ := sdk.AccAddressFromBech32(msg.Creator)
	if err = k.Keeper.AddLiquidityWithCompound(ctx, acc, msg.Denom, amount, msg.AutoCompound); err != nil {
		return nil, fmt.Errorf("add liquidity: %w", err)
	}

	return &types.MsgAddLiquidityResponse{}, nil
}

func (k msgServer) ChangePayout(ctx context.Context, msg *types.MsgChangePayout) (*types.Void, error) {
	liquidityShares, has := k.liquidityPositions.Get(ctx, msg.Creator, msg.PositionIndex)
	if !has {
		return nil, fmt.Errorf("find liquidity deposit for %s", msg.Creator)
	}

	liquidityShares.AutoCompound = msg.AutoCompound
	k.liquidityPositions.Set(ctx, msg.Creator, msg.PositionIndex, liquidityShares)

	return &types.Void{}, nil
}

func (k msgServer) RemoveAllLiquidityForDenom(ctx context.Context, msg *types.MsgRemoveAllLiquidityForDenom) (*types.MsgRemoveLiquidityResponse, error) {
	address, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return nil, types.ErrInvalidAddress
	}

	liq := k.GetLiquidityByAddress(ctx, msg.WithdrawDenom, msg.Creator)
	amount, err := k.RemoveLiquidityForAddress(ctx, address, msg.WithdrawDenom, liq, nil)
	if err != nil {
		return nil, err
	}

	if msg.WithdrawDenom != msg.PayoutDenom {
		if _, err = k.Sell(ctx, &types.MsgSell{
			Creator:        msg.Creator,
			DenomGiving:    msg.WithdrawDenom,
			DenomReceiving: msg.PayoutDenom,
			Amount:         amount.String(),
		}); err != nil {
			return nil, fmt.Errorf("sell liquidity for address: %w", err)
		}
	}

	return &types.MsgRemoveLiquidityResponse{}, nil
}

func (k Keeper) RemoveAllLiquidityForAddress(ctx context.Context, address, denom string) error {
	amount := math.ZeroInt()

	iterator := k.liquidityEntries.Iterator(ctx, nil, denom)
	for iterator.Valid() {
		liq := iterator.GetNext()
		if liq.Address == address {
			amount = amount.Add(liq.Amount)
			k.AddLiquidityAddressSum(ctx, liq.Address, denom, liq.Amount.Neg())
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
	amount, err := trading.ParseAmount(msg.Amount)
	if err != nil {
		return nil, fmt.Errorf("parse amount: %w", err)
	}

	if err = k.precheckTradeWithBalance(ctx, msg.Creator, msg.WithdrawDenom, &amount, false, false); err != nil {
		return nil, fmt.Errorf("error validating message: %w", err)
	}

	addedAmount := k.GetLiquidityByAddress(ctx, msg.WithdrawDenom, msg.Creator)
	if addedAmount.LT(amount) {
		return nil, fmt.Errorf("asked amount (%v) is bigger than added amount (%v)", amount.String(), addedAmount.String())
	}

	var positionIndex *uint64
	if msg.PositionIndex != "" {
		positionIndexInt, err := strconv.Atoi(msg.PositionIndex)
		if err != nil {
			return nil, fmt.Errorf("parse deposit index: %w", err)
		}

		_, has := k.liquidityPositions.Get(ctx, msg.Creator, uint64(positionIndexInt))
		if !has {
			return nil, fmt.Errorf("liquidity does not exist for given address and deposit index")
		}

		positionIndex64 := uint64(positionIndexInt)
		positionIndex = &positionIndex64
	}

	acc, _ := sdk.AccAddressFromBech32(msg.Creator)
	amount, err = k.RemoveLiquidityForAddress(ctx, acc, msg.WithdrawDenom, amount, positionIndex)
	if err != nil {
		return nil, fmt.Errorf("remove liquidity for address: %w", err)
	}

	if msg.WithdrawDenom != msg.PayoutDenom {
		if _, err = k.Sell(ctx, &types.MsgSell{
			Creator:        msg.Creator,
			DenomGiving:    msg.WithdrawDenom,
			DenomReceiving: msg.PayoutDenom,
			Amount:         amount.String(),
		}); err != nil {
			return nil, fmt.Errorf("sell liquidity for address: %w", err)
		}
	}

	return &types.MsgRemoveLiquidityResponse{}, nil
}

func (k Keeper) RemoveLiquidityForAddress(ctx context.Context, accAddr sdk.AccAddress, denom string, amount math.Int, positionIndex *uint64) (math.Int, error) {
	removed := math.ZeroInt()
	address := accAddr.String()

	iterator := k.liquidityEntries.Iterator(ctx, nil, denom)
	for iterator.Valid() {
		liq := iterator.GetNext()
		if liq.Address != address {
			continue
		}

		canUnlock, err := k.canUnlock(ctx, address, liq.PositionIndex)
		if err != nil {
			return math.Int{}, fmt.Errorf("can unlock: %w", err)
		}

		if !canUnlock {
			continue
		}

		if liq.Address == address && (positionIndex == nil || *positionIndex == liq.PositionIndex) {
			var amountRemovedForPosition math.Int

			if liq.Amount.GT(amount) {
				amountRemovedForPosition = amount
				liq.Amount = liq.Amount.Sub(amount)
				k.AddLiquidityAddressSum(ctx, liq.Address, denom, amount.Neg())
				k.SetLiquidity(ctx, denom, liq)
				amount = math.ZeroInt()
			} else {
				amountRemovedForPosition = liq.Amount
				amount = amount.Sub(liq.Amount)
				k.AddLiquidityAddressSum(ctx, liq.Address, denom, liq.Amount.Neg())
				k.RemoveLiquidity(ctx, denom, liq.Index)
			}

			amountUSD, _ := k.DenomKeeper.GetValueInUSD(ctx, denom, amountRemovedForPosition.ToLegacyDec())

			sdk.UnwrapSDKContext(ctx).EventManager().EmitEvent(
				sdk.NewEvent(
					"liquidity_removed",
					sdk.Attribute{Key: "denom", Value: denom},
					sdk.Attribute{Key: "amount", Value: amountRemovedForPosition.String()},
					sdk.Attribute{Key: "amount_usd", Value: amountUSD.String()},
					sdk.Attribute{Key: "address", Value: liq.Address},
					sdk.Attribute{Key: "index", Value: strconv.Itoa(int(liq.Index))},
					sdk.Attribute{Key: "position_index", Value: strconv.Itoa(int(liq.PositionIndex))},
				),
			)

			removed = removed.Add(amountRemovedForPosition)
		}

		if amount.IsZero() {
			break
		}
	}

	if amount.IsPositive() {
		return math.Int{}, types.ErrNotEnoughFunds
	}

	coins := sdk.NewCoins(sdk.NewCoin(denom, removed))
	if err := k.BankKeeper.SendCoinsFromModuleToAccount(ctx, types.PoolLiquidity, accAddr, coins); err != nil {
		return math.Int{}, fmt.Errorf("send coins from module to account: %w", err)
	}

	k.capMovingLiquidity(ctx, denom)
	return removed, nil
}

// only used in testnet to account for issue where module account has more liuqidity positions than there are funds in pool
func (k Keeper) dissolvePosition(ctx context.Context, address, denom string, amount math.Int) error {
	removed := math.ZeroInt()

	iterator := k.liquidityEntries.Iterator(ctx, nil, denom)
	for iterator.Valid() {
		liq := iterator.GetNext()

		if liq.Address == address {
			var amountRemovedForPosition math.Int

			if liq.Amount.GT(amount) {
				amountRemovedForPosition = amount
				liq.Amount = liq.Amount.Sub(amount)
				k.SetLiquidity(ctx, denom, liq)
				amount = math.ZeroInt()
			} else {
				amountRemovedForPosition = liq.Amount
				amount = amount.Sub(liq.Amount)
				k.RemoveLiquidity(ctx, denom, liq.Index)
			}

			removed = removed.Add(amountRemovedForPosition)
		}

		if amount.IsZero() {
			break
		}
	}

	if amount.IsPositive() {
		return types.ErrNotEnoughFunds
	}

	return nil
}
