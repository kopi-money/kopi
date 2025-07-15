package keeper

import (
	"context"
	"cosmossdk.io/math"
	"fmt"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/kopi-money/kopi/x/tokenfactory/types"
)

func (k msgServer) PayKCoinReward(ctx context.Context, msg *types.MsgPayKCoinReward) (*types.Void, error) {
	amount, ok := math.NewIntFromString(msg.Amount)
	if !ok {
		return nil, fmt.Errorf("invalid amount: %s", msg.Amount)
	}

	if !amount.IsPositive() {
		return nil, fmt.Errorf("invalid amount: %s", msg.Amount)
	}

	factoryDenom, has := k.GetDenomByFullName(ctx, msg.FullFactoryDenomName)
	if !has {
		return nil, types.ErrDenomDoesNotExist
	}

	pool, has := k.liquidityPools.Get(ctx, msg.FullFactoryDenomName)
	if !has {
		return nil, types.ErrPoolDoesNotExist
	}

	acc, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return nil, fmt.Errorf("invalid user address: %w", err)
	}

	if k.BankKeeper.SpendableCoin(ctx, acc, pool.KCoin).Amount.LT(amount) {
		return nil, types.ErrInsufficientFunds
	}

	coins := sdk.NewCoins(sdk.NewCoin(pool.KCoin, amount))
	if err = k.BankKeeper.SendCoinsFromAccountToModule(ctx, acc, types.PoolFactoryLiquidity, coins); err != nil {
		return nil, fmt.Errorf("send coins to Liquidity pool: %w", err)
	}

	pool.KCoinAmount = pool.KCoinAmount.Add(amount)
	k.liquidityPools.Set(ctx, factoryDenom.FullName, pool)

	return &types.Void{}, nil
}

func (k msgServer) PayFactoryReward(ctx context.Context, msg *types.MsgPayFactoryReward) (*types.Void, error) {
	amount, ok := math.NewIntFromString(msg.Amount)
	if !ok {
		return nil, fmt.Errorf("invalid amount: %s", msg.Amount)
	}

	if !amount.IsPositive() {
		return nil, fmt.Errorf("invalid amount: %s", msg.Amount)
	}

	factoryDenom, has := k.GetDenomByFullName(ctx, msg.FullFactoryDenomName)
	if !has {
		return nil, types.ErrDenomDoesNotExist
	}

	pool, has := k.liquidityPools.Get(ctx, msg.FullFactoryDenomName)
	if !has {
		return nil, types.ErrPoolDoesNotExist
	}

	acc, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return nil, fmt.Errorf("invalid user address: %w", err)
	}

	if k.BankKeeper.SpendableCoin(ctx, acc, factoryDenom.FullName).Amount.LT(amount) {
		return nil, types.ErrInsufficientFunds
	}

	coins := sdk.NewCoins(sdk.NewCoin(factoryDenom.FullName, amount))
	if err = k.BankKeeper.SendCoinsFromAccountToModule(ctx, acc, types.PoolFactoryLiquidity, coins); err != nil {
		return nil, fmt.Errorf("send coins to Liquidity pool: %w", err)
	}

	pool.FactoryDenomAmount = pool.FactoryDenomAmount.Add(amount)
	k.liquidityPools.Set(ctx, factoryDenom.FullName, pool)

	return &types.Void{}, nil
}
