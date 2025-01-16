package keeper

import (
	"context"
	"fmt"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/kopi-money/kopi/x/tokenfactory/types"
)

func (k msgServer) CreatePool(ctx context.Context, msg *types.MsgCreatePool) (*types.Void, error) {
	factoryDenom, has := k.GetDenomByFullName(ctx, msg.FullFactoryDenomName)
	if !has {
		return nil, types.ErrDenomDoesNotExists
	}

	if factoryDenom.Admin != msg.Creator {
		return nil, types.ErrIncorrectAdmin
	}

	pool, has := k.liquidityPools.Get(ctx, factoryDenom.FullName)
	if has {
		return nil, types.ErrPoolAlreadyExists
	}

	if msg.KCoin == "" {
		return nil, types.ErrEmptyKCoin
	}

	if !k.DenomKeeper.IsKCoin(ctx, msg.KCoin) {
		return nil, types.ErrNoKCoin
	}

	amountFactory, ok := math.NewIntFromString(msg.FactoryDenomAmount)
	if !ok {
		return nil, fmt.Errorf("invalid factory denom amount: %v", msg.FactoryDenomAmount)
	}

	kCoinAmount, ok := math.NewIntFromString(msg.KCoinAmount)
	if !ok {
		return nil, fmt.Errorf("invalid other kcoin amount: %v", msg.KCoinAmount)
	}

	if kCoinAmount.LT(k.getMinimumPoolSize(ctx)) {
		return nil, types.ErrAmountBelowMinimum
	}

	poolFee, err := math.LegacyNewDecFromStr(msg.PoolFee)
	if err != nil {
		return nil, types.ErrInvalidFeeFormat
	}

	if poolFee.GT(math.LegacyOneDec()) {
		return nil, types.ErrPoolFeeToLarge
	}

	if int64(msg.UnlockInSeconds) < k.GetParams(ctx).MinimumUnlockInSeconds {
		return nil, types.ErrUnlockTooShort
	}

	pool = types.LiquidityPool{
		FactoryDenomAmount: amountFactory,
		KCoin:              msg.KCoin,
		KCoinAmount:        kCoinAmount,
		PoolFee:            poolFee,
		UnlockInSeconds:    msg.UnlockInSeconds,
		CreatedAt:          sdk.UnwrapSDKContext(ctx).BlockHeight(),
	}

	k.liquidityPools.Set(ctx, factoryDenom.FullName, pool)

	coins := sdk.NewCoins(
		sdk.NewCoin(factoryDenom.FullName, amountFactory),
		sdk.NewCoin(msg.KCoin, kCoinAmount),
	)

	adminAcc, _ := sdk.AccAddressFromBech32(msg.Creator)
	if err = k.BankKeeper.SendCoinsFromAccountToModule(ctx, adminAcc, types.PoolFactoryLiquidity, coins); err != nil {
		return nil, err
	}

	provider := types.ProviderShare{Share: math.LegacyOneDec()}
	k.liquidityProviderShares.Set(ctx, factoryDenom.FullName, msg.Creator, provider)

	return &types.Void{}, nil
}

func (k Keeper) getLiquidityForAddress(ctx context.Context, fullName, amount string) (types.FactoryDenom, types.LiquidityPool, math.Int, math.Int, error) {
	factoryDenom, has := k.GetDenomByFullName(ctx, fullName)
	if !has {
		return types.FactoryDenom{}, types.LiquidityPool{}, math.Int{}, math.Int{}, types.ErrDenomDoesNotExists
	}

	pool, has := k.liquidityPools.Get(ctx, factoryDenom.FullName)
	if !has {
		return types.FactoryDenom{}, types.LiquidityPool{}, math.Int{}, math.Int{}, types.ErrPoolDoesNotExist
	}

	amountFactory, ok := math.NewIntFromString(amount)
	if !ok {
		return types.FactoryDenom{}, types.LiquidityPool{}, math.Int{}, math.Int{}, fmt.Errorf("invalid factory denom amount: %v", amount)
	}

	poolRatio, err := getPoolRatio(pool)
	if err != nil {
		return types.FactoryDenom{}, types.LiquidityPool{}, math.Int{}, math.Int{}, err
	}

	amountOtherDenom := amountFactory.ToLegacyDec().Mul(poolRatio).TruncateInt()

	return factoryDenom, pool, amountFactory, amountOtherDenom, nil
}

func (k msgServer) AddLiquidity(ctx context.Context, msg *types.MsgAddLiquidity) (*types.Void, error) {
	factoryDenom, pool, amountFactory, amountKCoin, err := k.getLiquidityForAddress(ctx, msg.FullFactoryDenomName, msg.FactoryDenomAmount)
	if err != nil {
		return nil, fmt.Errorf("could not check liquidity: %w", err)
	}

	acc, _ := sdk.AccAddressFromBech32(msg.Creator)
	coins := sdk.NewCoins(
		sdk.NewCoin(factoryDenom.FullName, amountFactory),
		sdk.NewCoin(pool.KCoin, amountKCoin),
	)

	if err = k.BankKeeper.SendCoinsFromAccountToModule(ctx, acc, types.PoolFactoryLiquidity, coins); err != nil {
		return nil, fmt.Errorf("could not send coins to Liquidity pool: %w", err)
	}

	_ = k.updateLiquidityShare(ctx, factoryDenom, pool, amountFactory, msg.Creator)

	pool.FactoryDenomAmount = pool.FactoryDenomAmount.Add(amountFactory)
	pool.KCoinAmount = pool.KCoinAmount.Add(amountKCoin)
	k.liquidityPools.Set(ctx, factoryDenom.FullName, pool)

	return &types.Void{}, nil
}

func (k msgServer) UnlockLiquidity(ctx context.Context, msg *types.MsgUnlockLiquidity) (*types.Void, error) {
	factoryDenom, pool, amountFactory, amountKCoin, err := k.getLiquidityForAddress(ctx, msg.FullFactoryDenomName, msg.FactoryDenomAmount)
	if err != nil {
		return nil, fmt.Errorf("could not check liquidity: %w", err)
	}

	providerShare := k.getLiquidityShare(ctx, msg.FullFactoryDenomName, msg.Creator)
	factoryAmountShare := pool.FactoryDenomAmount.ToLegacyDec().Mul(providerShare).TruncateInt()
	if factoryAmountShare.LT(amountFactory) {
		return nil, types.ErrAmountTooLarge
	}

	pool.FactoryDenomAmount = pool.FactoryDenomAmount.Sub(amountFactory)
	pool.KCoinAmount = pool.KCoinAmount.Sub(amountKCoin)
	k.liquidityPools.Set(ctx, factoryDenom.FullName, pool)

	if pool.FactoryDenomAmount.IsNegative() || pool.KCoinAmount.IsNegative() {
		return nil, types.ErrNegativeLiquidity
	}

	coins := sdk.NewCoins(
		sdk.NewCoin(factoryDenom.FullName, amountFactory),
		sdk.NewCoin(pool.KCoin, amountKCoin),
	)

	if err = k.BankKeeper.SendCoinsFromModuleToModule(ctx, types.PoolFactoryLiquidity, types.PoolUnlocking, coins); err != nil {
		return nil, fmt.Errorf("could not send coins from module to module: %w", err)
	}

	if err = k.updateLiquidityShare(ctx, factoryDenom, pool, amountFactory, msg.Creator); err != nil {
		return nil, fmt.Errorf("could not update liquidity share: %w", err)
	}

	now := sdk.UnwrapSDKContext(ctx).BlockTime()
	k.SetLiquidityUnlocking(ctx, types.LiquidityUnlocking{
		Index:              0,
		Address:            msg.Creator,
		FactoryDenomHash:   factoryDenom.FullName,
		FactoryDenomAmount: amountFactory,
		KCoin:              pool.KCoin,
		KCoinAmount:        amountKCoin,
		CreatedAt:          &now,
	})

	return &types.Void{}, nil
}

func (k msgServer) UpdateLiquidityPoolSettings(ctx context.Context, msg *types.MsgUpdateLiquidityPoolSettings) (*types.Void, error) {
	factoryDenom, has := k.GetDenomByFullName(ctx, msg.FullFactoryDenomName)
	if !has {
		return nil, types.ErrDenomDoesNotExists
	}

	if factoryDenom.Admin != msg.Creator {
		return nil, types.ErrIncorrectAdmin
	}

	pool, has := k.liquidityPools.Get(ctx, factoryDenom.FullName)
	if !has {
		return nil, types.ErrPoolDoesNotExist
	}

	var err error
	pool.PoolFee, err = math.LegacyNewDecFromStr(msg.PoolFee)
	if err != nil {
		return nil, types.ErrInvalidFeeFormat
	}

	if pool.PoolFee.IsNegative() {
		return nil, types.ErrInvalidNegativeFee
	}

	if pool.PoolFee.GT(math.LegacyOneDec()) {
		return nil, types.ErrPoolFeeToLarge
	}

	if pool.UnlockInSeconds > msg.UnlockInSeconds {
		return nil, types.ErrShorterUnlockPeriod
	}

	pool.UnlockInSeconds = msg.UnlockInSeconds

	k.SetLiquidityPool(ctx, factoryDenom.FullName, pool)
	return &types.Void{}, nil
}

func (k msgServer) DissolvePool(ctx context.Context, msg *types.MsgDissolvePool) (*types.Void, error) {
	factoryDenom, has := k.GetDenomByFullName(ctx, msg.FullFactoryDenomName)
	if !has {
		return nil, types.ErrDenomDoesNotExists
	}

	if factoryDenom.Admin != msg.Creator {
		return nil, types.ErrIncorrectAdmin
	}

	pool, has := k.liquidityPools.Get(ctx, factoryDenom.FullName)
	if !has {
		return nil, types.ErrPoolDoesNotExist
	}

	if err := k.payoutLiquidityProviders(ctx, factoryDenom, pool); err != nil {
		return nil, fmt.Errorf("could not payout liquidity Providers: %w", err)
	}

	k.liquidityPools.Remove(ctx, factoryDenom.FullName)

	return &types.Void{}, nil
}

func (k Keeper) payoutLiquidityUnlockins(ctx context.Context, factoryDenom types.FactoryDenom, pool types.LiquidityPool) error {
	unlockingIterator := k.LiquidityUnlockingsIterator(ctx)
	var deleteKeys []uint64

	for unlockingIterator.Valid() {
		keyValue := unlockingIterator.GetNextKeyValue()
		unlocking := keyValue.Value().Value()
		if unlocking.FactoryDenomHash != factoryDenom.FullName {
			continue
		}

		coins := sdk.NewCoins(
			sdk.NewCoin(factoryDenom.FullName, unlocking.FactoryDenomAmount),
			sdk.NewCoin(pool.KCoin, unlocking.KCoinAmount),
		)

		acc, _ := sdk.AccAddressFromBech32(unlocking.Address)
		if err := k.BankKeeper.SendCoinsFromModuleToAccount(ctx, types.PoolFactoryLiquidity, acc, coins); err != nil {
			return fmt.Errorf("could not send coins from module to account: %w", err)
		}

		deleteKeys = append(deleteKeys, keyValue.Key())
	}

	for _, deleteKey := range deleteKeys {
		k.liquidityUnlockings.Remove(ctx, deleteKey)
	}

	return nil
}

func (k Keeper) payoutLiquidityProviders(ctx context.Context, factoryDenom types.FactoryDenom, pool types.LiquidityPool) error {
	ratio, err := getPoolRatio(pool)
	if err != nil {
		return err
	}

	shareIterator := k.LiquidityShareIterator(ctx, factoryDenom.FullName)

	for shareIterator.Valid() {
		keyValue := shareIterator.GetNextKeyValue()

		amountFactory := pool.FactoryDenomAmount.ToLegacyDec().Mul(keyValue.Value().Value().Share)
		amountOtherDenom := amountFactory.Mul(ratio)

		coins := sdk.NewCoins(
			sdk.NewCoin(factoryDenom.FullName, amountFactory.TruncateInt()),
			sdk.NewCoin(pool.KCoin, amountOtherDenom.TruncateInt()),
		)

		acc, _ := sdk.AccAddressFromBech32(keyValue.Key())
		if err := k.BankKeeper.SendCoinsFromModuleToAccount(ctx, types.PoolFactoryLiquidity, acc, coins); err != nil {
			return fmt.Errorf("could not send coins from module to account: %w", err)
		}
	}

	return nil
}
