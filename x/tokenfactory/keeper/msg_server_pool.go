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

	if _, has = k.liquidityPools.Get(ctx, factoryDenom.FullName); has {
		return nil, types.ErrPoolAlreadyExists
	}

	if factoryDenom.Moved {
		return nil, types.ErrDenomAlreadyMoved
	}

	if err := k.Keeper.CreatePool(ctx, factoryDenom, msg.KCoin, msg.FactoryDenomAmount, msg.KCoinAmount, msg.PoolFee, msg.UnlockInSeconds); err != nil {
		return nil, fmt.Errorf("failed to create pool: %w", err)
	}

	return &types.Void{}, nil
}

func (k Keeper) CreatePool(ctx context.Context, factoryDenom types.FactoryDenom, kCoin, factoryDenomAmount, kCoinAmount, poolFeeStr string, unlockInSeconds uint64) error {
	if kCoin == "" {
		return types.ErrEmptyKCoin
	}

	if !k.DenomKeeper.IsFactoryPoolDenom(ctx, kCoin) {
		return types.ErrNoValidPoolDenom
	}

	amountFactory, ok := math.NewIntFromString(factoryDenomAmount)
	if !ok {
		return fmt.Errorf("invalid factory denom amount: %v", factoryDenomAmount)
	}

	amountKCoin, ok := math.NewIntFromString(kCoinAmount)
	if !ok {
		return fmt.Errorf("invalid other kcoin amount: %v", kCoinAmount)
	}

	if amountKCoin.LT(k.DenomKeeper.MinimumFactoryPoolSize(ctx, kCoin)) {
		return types.ErrAmountBelowMinimum
	}

	poolFee, err := math.LegacyNewDecFromStr(poolFeeStr)
	if err != nil {
		return types.ErrInvalidFeeFormat
	}

	if poolFee.GT(k.getMaximumPoolFee(ctx)) {
		return types.ErrPoolFeeTooLarge
	}

	if poolFee.LT(k.getMinimumPoolFee(ctx)) {
		return types.ErrPoolFeeTooSmall
	}

	if int64(unlockInSeconds) < k.GetParams(ctx).MinimumUnlockInSeconds {
		return types.ErrUnlockTooShort
	}

	pool := types.LiquidityPool{
		KCoin:              kCoin,
		KCoinAmount:        amountKCoin,
		FactoryDenomAmount: amountFactory,
		PoolFee:            poolFee,
		UnlockInSeconds:    unlockInSeconds,
		CreatedAt:          sdk.UnwrapSDKContext(ctx).BlockHeight(),
	}

	k.liquidityPools.Set(ctx, factoryDenom.FullName, pool)

	var denom string
	if factoryDenom.LocalName != "" {
		denom = factoryDenom.LocalName
	} else {
		denom = factoryDenom.FullName
	}

	coins := sdk.NewCoins(
		sdk.NewCoin(denom, amountFactory),
		sdk.NewCoin(kCoin, amountKCoin),
	)

	adminAcc, _ := sdk.AccAddressFromBech32(factoryDenom.Admin)
	if err = k.BankKeeper.SendCoinsFromAccountToModule(ctx, adminAcc, types.PoolFactoryLiquidity, coins); err != nil {
		return err
	}

	provider := types.ProviderShare{Share: math.LegacyOneDec()}
	k.liquidityProviderShares.Set(ctx, factoryDenom.FullName, factoryDenom.Admin, provider)

	return nil
}

func (k msgServer) AddLiquidity(ctx context.Context, msg *types.MsgAddLiquidity) (*types.Void, error) {
	acc, _ := sdk.AccAddressFromBech32(msg.Creator)

	amount, ok := math.NewIntFromString(msg.FactoryDenomAmount)
	if !ok {
		return nil, fmt.Errorf("invalid factory denom amount: %v", msg.FactoryDenomAmount)
	}

	if err := k.Keeper.AddLiquidity(ctx, acc, amount, msg.FullFactoryDenomName, msg.FullFactoryDenomName); err != nil {
		return nil, fmt.Errorf("adding liquidity: %w", err)
	}

	return &types.Void{}, nil
}

func (k Keeper) AddLiquidity(ctx context.Context, acc sdk.AccAddress, amount math.Int, fullName, givenDenom string) error {
	factoryDenom, has := k.GetDenomByFullName(ctx, fullName)
	if !has {
		return types.ErrDenomDoesNotExists
	}

	pool, has := k.liquidityPools.Get(ctx, factoryDenom.FullName)
	if !has {
		return types.ErrPoolDoesNotExist
	}

	var (
		amountFactory math.Int
		amountKCoin   math.Int
	)

	if pool.KCoin == givenDenom {
		amountKCoin = amount
		amountFactoryDec, _ := pool.ConvertToFactory(amount)
		amountFactory = amountFactoryDec.Ceil().TruncateInt()
	} else {
		amountFactory = amount
		amountKCoinDec, _ := pool.ConvertToKCoin(amount)
		amountKCoin = amountKCoinDec.Ceil().TruncateInt()
	}

	coins := sdk.NewCoins(
		sdk.NewCoin(factoryDenom.FactoryTradeDenom(), amountFactory),
		sdk.NewCoin(pool.KCoin, amountKCoin),
	)

	if err := k.BankKeeper.SendCoinsFromAccountToModule(ctx, acc, types.PoolFactoryLiquidity, coins); err != nil {
		return fmt.Errorf("send coins to Liquidity pool: %w", err)
	}

	if err := k.updateLiquidityShare(ctx, factoryDenom, pool.FactoryDenomAmount.ToLegacyDec(), amountFactory.ToLegacyDec(), acc.String()); err != nil {
		return fmt.Errorf("update liquidity share: %w", err)
	}

	pool.FactoryDenomAmount = pool.FactoryDenomAmount.Add(amountFactory)
	pool.KCoinAmount = pool.KCoinAmount.Add(amountKCoin)
	k.liquidityPools.Set(ctx, factoryDenom.FullName, pool)

	return nil
}

func (k msgServer) AddKCoinLiquidity(ctx context.Context, msg *types.MsgAddKCoinLiquidity) (*types.Void, error) {
	amount, ok := math.NewIntFromString(msg.Amount)
	if !ok {
		return nil, fmt.Errorf("invalid amount: %s", msg.Amount)
	}

	factoryDenom, has := k.GetDenomByFullName(ctx, msg.FullFactoryDenomName)
	if !has {
		return nil, types.ErrDenomDoesNotExists
	}

	pool, has := k.liquidityPools.Get(ctx, msg.FullFactoryDenomName)
	if !has {
		return nil, types.ErrPoolDoesNotExist
	}

	acc, _ := sdk.AccAddressFromBech32(msg.Creator)
	if k.BankKeeper.SpendableCoin(ctx, acc, pool.KCoin).Amount.LT(amount) {
		return nil, types.ErrInsufficientFunds
	}

	if err := k.Keeper.AddKCoinLiquidity(ctx, factoryDenom, pool, amount, msg.Creator); err != nil {
		return nil, fmt.Errorf("adding liquidity: %w", err)
	}

	return &types.Void{}, nil
}

// AddKCoinLiquidity adds one-sided liquidity to the pool. When updating the shares, only half of the added amount is
// considered because shares are calculated based on adding two-sided.
func (k Keeper) AddKCoinLiquidity(ctx context.Context, factoryDenom types.FactoryDenom, pool types.LiquidityPool, kCoinAmount math.Int, creator string) error {
	coins := sdk.NewCoins(
		sdk.NewCoin(pool.KCoin, kCoinAmount),
	)

	acc, _ := sdk.AccAddressFromBech32(creator)
	if err := k.BankKeeper.SendCoinsFromAccountToModule(ctx, acc, types.PoolFactoryLiquidity, coins); err != nil {
		return fmt.Errorf("send coins to Liquidity pool: %w", err)
	}

	addedAmount := kCoinAmount.ToLegacyDec().Quo(math.LegacyNewDec(2))
	if err := k.updateLiquidityShare(ctx, factoryDenom, pool.KCoinAmount.ToLegacyDec(), addedAmount, acc.String()); err != nil {
		return fmt.Errorf("update liquidity share: %w", err)
	}

	pool.KCoinAmount = pool.KCoinAmount.Add(kCoinAmount)
	k.liquidityPools.Set(ctx, factoryDenom.FullName, pool)

	return nil
}

func (k msgServer) AddFactoryLiquidity(ctx context.Context, msg *types.MsgAddFactoryLiquidity) (*types.Void, error) {
	amount, ok := math.NewIntFromString(msg.Amount)
	if !ok {
		return nil, fmt.Errorf("invalid amount: %s", msg.Amount)
	}

	factoryDenom, has := k.GetDenomByFullName(ctx, msg.FullFactoryDenomName)
	if !has {
		return nil, types.ErrDenomDoesNotExists
	}

	pool, has := k.liquidityPools.Get(ctx, msg.FullFactoryDenomName)
	if !has {
		return nil, types.ErrPoolDoesNotExist
	}

	acc, _ := sdk.AccAddressFromBech32(msg.Creator)
	if k.BankKeeper.SpendableCoin(ctx, acc, pool.KCoin).Amount.LT(amount) {
		return nil, types.ErrInsufficientFunds
	}

	if err := k.Keeper.AddFactoryLiquidity(ctx, factoryDenom, pool, amount, msg.Creator); err != nil {
		return nil, fmt.Errorf("adding liquidity: %w", err)
	}

	return &types.Void{}, nil
}

// AddFactoryLiquidity adds one-sided liquidity to the pool. When updating the shares, only half of the added amount is
// considered because shares are calculated based on adding two-sided.
func (k Keeper) AddFactoryLiquidity(ctx context.Context, factoryDenom types.FactoryDenom, pool types.LiquidityPool, factoryAmount math.Int, creator string) error {
	coins := sdk.NewCoins(
		sdk.NewCoin(factoryDenom.FactoryTradeDenom(), factoryAmount),
	)

	acc, _ := sdk.AccAddressFromBech32(creator)
	if err := k.BankKeeper.SendCoinsFromAccountToModule(ctx, acc, types.PoolFactoryLiquidity, coins); err != nil {
		return fmt.Errorf("send coins to Liquidity pool: %w", err)
	}

	addedAmount := factoryAmount.ToLegacyDec().Quo(math.LegacyNewDec(2))
	if err := k.updateLiquidityShare(ctx, factoryDenom, pool.FactoryDenomAmount.ToLegacyDec(), addedAmount, acc.String()); err != nil {
		return fmt.Errorf("update liquidity share: %w", err)
	}

	pool.FactoryDenomAmount = pool.FactoryDenomAmount.Add(factoryAmount)
	k.liquidityPools.Set(ctx, factoryDenom.FullName, pool)

	return nil
}

func (k msgServer) UnlockLiquidity(ctx context.Context, msg *types.MsgUnlockLiquidity) (*types.Void, error) {
	amountFactory, ok := math.NewIntFromString(msg.FactoryDenomAmount)
	if !ok {
		return nil, fmt.Errorf("invalid amount: %s", msg.FactoryDenomAmount)
	}

	factoryDenom, has := k.GetDenomByFullName(ctx, msg.FullFactoryDenomName)
	if !has {
		return nil, types.ErrDenomDoesNotExists
	}

	pool, has := k.liquidityPools.Get(ctx, msg.FullFactoryDenomName)
	if !has {
		return nil, types.ErrPoolDoesNotExist
	}

	acc, _ := sdk.AccAddressFromBech32(msg.Creator)
	if err := k.Keeper.UnlockLiquidity(ctx, factoryDenom, pool, acc, amountFactory, factoryDenom.FullName); err != nil {
		return nil, fmt.Errorf("unlock liquidity: %w", err)
	}

	return &types.Void{}, nil
}

func (k Keeper) UnlockLiquidity(ctx context.Context, factoryDenom types.FactoryDenom, pool types.LiquidityPool, acc sdk.AccAddress, amount math.Int, unlockDenom string) error {
	providerShare := k.getLiquidityShare(ctx, factoryDenom.FullName, acc.String())

	var (
		amountFactory math.Int
		amountKCoin   math.Int
	)

	if unlockDenom == factoryDenom.FullName {
		factoryAmountShare := pool.FactoryDenomAmount.ToLegacyDec().Mul(providerShare).TruncateInt()
		if factoryAmountShare.LT(amount) {
			return types.ErrAmountTooLarge
		}

		amouncKCoinDec, _ := pool.ConvertToKCoin(amount)
		amountKCoin = amouncKCoinDec.TruncateInt()
		amountFactory = amount
	} else {
		kCoinAmountShare := pool.KCoinAmount.ToLegacyDec().Mul(providerShare).TruncateInt()
		if kCoinAmountShare.LT(amount) {
			return types.ErrAmountTooLarge
		}

		amountFactoryDec, _ := pool.ConvertToFactory(amount)
		amountFactory = amountFactoryDec.TruncateInt()
		amountKCoin = amount
	}

	pool.FactoryDenomAmount = pool.FactoryDenomAmount.Sub(amountFactory)
	pool.KCoinAmount = pool.KCoinAmount.Sub(amountKCoin)
	k.liquidityPools.Set(ctx, factoryDenom.FullName, pool)

	if pool.FactoryDenomAmount.IsNegative() || pool.KCoinAmount.IsNegative() {
		return types.ErrNegativeLiquidity
	}

	coins := sdk.NewCoins(
		sdk.NewCoin(factoryDenom.FactoryTradeDenom(), amountFactory),
		sdk.NewCoin(pool.KCoin, amountKCoin),
	)

	if err := k.BankKeeper.SendCoinsFromModuleToModule(ctx, types.PoolFactoryLiquidity, types.PoolUnlocking, coins); err != nil {
		return fmt.Errorf("send coins from module to module: %w", err)
	}

	if err := k.updateLiquidityShare(ctx, factoryDenom, pool.FactoryDenomAmount.ToLegacyDec(), amountFactory.ToLegacyDec(), acc.String()); err != nil {
		return fmt.Errorf("update liquidity share: %w", err)
	}

	k.SetLiquidityUnlocking(ctx, types.LiquidityUnlocking{
		Index:              0,
		Address:            acc.String(),
		FactoryDenomHash:   factoryDenom.FullName,
		FactoryDenomAmount: amountFactory,
		KCoin:              pool.KCoin,
		KCoinAmount:        amountKCoin,
		CreatedAt:          sdk.UnwrapSDKContext(ctx).BlockTime(),
	})

	return nil
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

	if pool.PoolFee.LT(k.getMinimumPoolFee(ctx)) {
		return nil, types.ErrPoolFeeTooSmall
	}

	if pool.PoolFee.GT(k.getMaximumPoolFee(ctx)) {
		return nil, types.ErrPoolFeeTooLarge
	}

	if int64(msg.UnlockInSeconds) < k.GetParams(ctx).MinimumUnlockInSeconds {
		return nil, types.ErrUnlockTooShort
	}

	pool.UnlockInSeconds = msg.UnlockInSeconds

	k.SetLiquidityPool(ctx, factoryDenom.FullName, pool)
	return &types.Void{}, nil
}

func (k Keeper) payoutLiquidityUnlockings(ctx context.Context, factoryDenom types.FactoryDenom, pool types.LiquidityPool) error {
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
			return fmt.Errorf("send coins from module to account: %w", err)
		}

		deleteKeys = append(deleteKeys, keyValue.Key())
	}

	for _, deleteKey := range deleteKeys {
		k.liquidityUnlockings.Remove(ctx, deleteKey)
	}

	return nil
}

func (k Keeper) payoutLiquidityProviders(ctx context.Context, factoryDenom types.FactoryDenom, pool types.LiquidityPool) error {
	ratio, err := pool.GetPoolRatio()
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
		if err = k.BankKeeper.SendCoinsFromModuleToAccount(ctx, types.PoolFactoryLiquidity, acc, coins); err != nil {
			return fmt.Errorf("send coins from module to account: %w", err)
		}
	}

	return nil
}
