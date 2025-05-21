package keeper

import (
	"context"
	"cosmossdk.io/math"
	"fmt"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/kopi-money/kopi/x/tokenfactory/types"
)

func (k Keeper) QueryFactoryTokenBalance(ctx context.Context, req *types.GetFactoryTokenBalanceRequest) (*types.GetFactoryTokenBalanceResponse, error) {
	addr, _ := sdk.AccAddressFromBech32(req.Address)
	spendableCoins := k.BankKeeper.SpendableCoins(ctx, addr)

	referenceDenom, err := k.DenomKeeper.GetHighestUSDReference(ctx)
	if err != nil {
		return nil, fmt.Errorf("get highest usd reference: %w", err)
	}

	factoryBalances, err := k.loadFactoryBalances(ctx, spendableCoins, req.Address, referenceDenom)
	if err != nil {
		return nil, fmt.Errorf("factory balances: %w", err)
	}

	dexTokenBalance, err := k.loadDexBalances(ctx, spendableCoins, referenceDenom)

	return &types.GetFactoryTokenBalanceResponse{
		Factory: factoryBalances,
		Dex:     dexTokenBalance,
	}, nil
}

func (k Keeper) loadDexBalances(ctx context.Context, spendableCoins sdk.Coins, referenceDenom string) ([]types.DEXTokenBalance, error) {
	dexDenoms := k.DenomKeeper.KCoins(ctx)
	dexDenoms = append(dexDenoms, "ukopi")

	var dexTokenBalances []types.DEXTokenBalance
	for _, denom := range dexDenoms {
		amount := spendableCoins.AmountOf(denom)

		amountUSD, err := k.DenomKeeper.GetValueIn(ctx, denom, referenceDenom, amount.ToLegacyDec())
		if err != nil {
			return nil, fmt.Errorf("usd amount %s: %w", denom, err)
		}

		dexTokenBalances = append(dexTokenBalances, types.DEXTokenBalance{
			Denom:     denom,
			Amount:    amount.String(),
			AmountUsd: amountUSD.String(),
		})
	}

	return dexTokenBalances, nil
}

func (k Keeper) loadFactoryBalances(ctx context.Context, spendableCoins sdk.Coins, address, referenceDenom string) ([]types.FactoryTokenBalance, error) {
	var (
		balances                     []types.FactoryTokenBalance
		poolLiquidityFactoryTokenUSD math.LegacyDec
		poolLiquidityKCoinUSD        math.LegacyDec
		amountWalletUSD              math.LegacyDec
		amountVestedUSD              math.LegacyDec
		poolAmountKCoin              math.Int
		poolAmountFactoryToken       math.Int
		amountVested                 math.Int
		err                          error
	)

	tokenIterator := k.factoryDenoms.Iterator(ctx, nil)
	for tokenIterator.Valid() {
		keyValue := tokenIterator.GetNextKeyValue()
		walletAmount := spendableCoins.AmountOf(keyValue.Key())

		balance := types.FactoryTokenBalance{
			FactoryDenomHash:             keyValue.Key(),
			AmountWallet:                 walletAmount.String(),
			AmountWalletUsd:              "0",
			PoolLiquidityKcoin:           "0",
			PoolLiquidityKcoinUsd:        "0",
			PoolLiquidityFactoryToken:    "0",
			PoolLiquidityFactoryTokenUsd: "0",
			AmountVested:                 "0",
			AmountVestedUsd:              "0",
		}

		amountVested = k.getVestedAmount(ctx, keyValue.Key(), address)

		pool, has := k.liquidityPools.Get(ctx, keyValue.Key())
		if has {
			amountWalletUSD, err = k.convertToUSD(ctx, pool, walletAmount, referenceDenom)
			if err != nil {
				return nil, err
			}

			poolAmountKCoin, poolAmountFactoryToken, err = k.getLiquidity(ctx, keyValue.Key(), address)
			if err != nil {
				return nil, fmt.Errorf("get liquidity for address: %w", err)
			}

			poolLiquidityFactoryTokenUSD, err = k.convertToUSD(ctx, pool, poolAmountFactoryToken, referenceDenom)
			if err != nil {
				return nil, err
			}

			poolLiquidityKCoinUSD, err = k.DenomKeeper.GetValueIn(ctx, pool.KCoin, referenceDenom, poolAmountKCoin.ToLegacyDec())
			if err != nil {
				return nil, err
			}

			amountVestedUSD, err = k.DenomKeeper.GetValueIn(ctx, pool.KCoin, referenceDenom, amountVested.ToLegacyDec())
			if err != nil {
				return nil, err
			}

			balance.AmountWalletUsd = amountWalletUSD.String()
			balance.PoolLiquidityFactoryToken = poolAmountFactoryToken.String()
			balance.PoolLiquidityFactoryTokenUsd = poolLiquidityFactoryTokenUSD.String()
			balance.PoolLiquidityKcoin = poolAmountKCoin.String()
			balance.PoolLiquidityKcoinUsd = poolLiquidityKCoinUSD.String()
			balance.AmountVestedUsd = amountVestedUSD.String()
			balance.Kcoin = pool.KCoin
		}

		balances = append(balances, balance)
	}

	return balances, nil
}

func (k Keeper) convertToUSD(ctx context.Context, pool types.LiquidityPool, factoryAmount math.Int, referenceDenom string) (math.LegacyDec, error) {
	valueInKCoin, err := pool.ConvertToKCoin(factoryAmount)
	if err != nil {
		return math.LegacyDec{}, err
	}

	valueInUSD, err := k.DenomKeeper.GetValueIn(ctx, pool.KCoin, referenceDenom, valueInKCoin)
	if err != nil {
		return math.LegacyDec{}, err
	}

	return valueInUSD, nil
}
