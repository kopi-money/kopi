package keeper

import (
	"context"
	"fmt"

	"github.com/kopi-money/kopi/trading"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/kopi-money/kopi/x/tokenfactory/types"
)

func (k msgServer) Buyback(ctx context.Context, msg *types.MsgBuyback) (*types.MsgBuybackResponse, error) {
	factoryDenom, has := k.GetDenomByFullName(ctx, msg.FullFactoryDenomName)
	if !has {
		return nil, types.ErrDenomDoesNotExist
	}

	pool, has := k.liquidityPools.Get(ctx, factoryDenom.FullName)
	if !has {
		return nil, types.ErrPoolDoesNotExist
	}

	amount, err := trading.ParseAmount(msg.BuybackAmount)
	if err != nil {
		return nil, err
	}

	tradeContext := types.TradeContext{
		Context: ctx,

		TradeAmount: amount,
		Callbacks:   trading.SellCallbacks(),

		Pool:           pool,
		DenomGiving:    pool.KCoin,
		DenomReceiving: factoryDenom.FullName,
		Creator:        msg.GetCreator(),
	}

	acc, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return nil, types.ErrInvalidAddress
	}

	res, err := k.Keeper.Trade(tradeContext, factoryDenom)
	if err != nil {
		return nil, err
	}

	amountReceivedNet, ok := math.NewIntFromString(res.AmountReceivedNet)
	if !ok {
		return nil, fmt.Errorf("invalid amount received from trade operation: %s", res.AmountReceivedNet)
	}

	if !amountReceivedNet.IsPositive() {
		return nil, fmt.Errorf("invalid trade result: non-positive amount %s", res.AmountReceivedNet)
	}

	coins := sdk.NewCoins(sdk.NewCoin(factoryDenom.FullName, amountReceivedNet))
	if err = k.BankKeeper.SendCoinsFromAccountToModule(ctx, acc, types.ModuleName, coins); err != nil {
		return nil, err
	}

	if err = k.BankKeeper.BurnCoins(ctx, types.ModuleName, coins); err != nil {
		return nil, err
	}

	sdk.UnwrapSDKContext(ctx).EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			"factory_denom_buyback",
			sdk.NewAttribute("factor_denom_full_name", factoryDenom.FullName),
			sdk.NewAttribute("buyback_amount", res.AmountGivenGross),
			sdk.NewAttribute("amount_burned", res.AmountReceivedNet),
		),
	})

	return &types.MsgBuybackResponse{
		BuybackAmount: res.AmountGivenGross,
		BurnedAmount:  res.AmountReceivedNet,
	}, nil
}
