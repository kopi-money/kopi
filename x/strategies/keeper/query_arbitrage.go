package keeper

import (
	"context"
	"fmt"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	denomtypes "github.com/kopi-money/kopi/x/denominations/types"
	"github.com/kopi-money/kopi/x/strategies/types"
)

func (k Keeper) ArbitrageDenomBalance(ctx context.Context, req *types.QueryArbitrageDenomBalanceRequest) (*types.QueryArbitrageDenomBalanceResponse, error) {
	aAsset, err := k.DenomKeeper.GetArbitrageDenomByName(ctx, req.Denom)
	if err != nil {
		return nil, err
	}

	acc := k.AccountKeeper.GetModuleAccount(ctx, types.PoolArbitrage)
	coins := k.BankKeeper.SpendableCoins(ctx, acc.GetAddress())

	supply := k.BankKeeper.GetSupply(ctx, aAsset.DexDenom)

	return &types.QueryArbitrageDenomBalanceResponse{
		KCoin:  coins.AmountOf(aAsset.KCoin).String(),
		CAsset: coins.AmountOf(aAsset.CAsset).String(),
		Supply: supply.Amount.String(),
	}, nil
}

func (k Keeper) ArbitrageBalance(ctx context.Context, _ *types.QueryArbitrageBalancesRequest) (*types.QueryArbitrageBalancesResponse, error) {
	referenceDenom, err := k.DexKeeper.GetHighestUSDReference(ctx)
	if err != nil {
		return nil, fmt.Errorf("could not get reference denom: %w", err)
	}

	var (
		acc           = k.AccountKeeper.GetModuleAccount(ctx, types.PoolArbitrage)
		coins         = k.BankKeeper.SpendableCoins(ctx, acc.GetAddress())
		totalValueUSD = math.LegacyZeroDec()

		cAsset        denomtypes.CAsset
		tokenValue    math.LegacyDec
		tokenValueUSD math.LegacyDec
		parity        *math.LegacyDec
	)

	var balances []*types.ArbitrageBalance
	for _, arbitrageDenom := range k.DenomKeeper.GetArbitrageDenoms(ctx) {
		cAsset, err = k.DenomKeeper.GetCAsset(ctx, arbitrageDenom.CAsset)
		if err != nil {
			return nil, fmt.Errorf("could not get cAsset: %w", err)
		}

		tokenValue, err = k.calculateArbitrageTokenValue(ctx, arbitrageDenom).get()
		if err != nil {
			return nil, fmt.Errorf("could not calculate arbcoin value of %v: %w", arbitrageDenom.DexDenom, err)
		}

		redemptionValue := k.MMKeeper.CalculateCAssetRedemptionValue(ctx, cAsset)
		baseValue := tokenValue.Mul(redemptionValue)

		tokenValueUSD, err = k.DexKeeper.GetValueIn(ctx, cAsset.BaseDexDenom, referenceDenom, baseValue)
		if err != nil {
			return nil, fmt.Errorf("could not get usd value of %v: %w", arbitrageDenom.DexDenom, err)
		}

		totalValueUSD = totalValueUSD.Add(tokenValueUSD)

		parity, _, err = k.DexKeeper.CalculateParity(ctx, arbitrageDenom.KCoin)
		if err != nil {
			return nil, fmt.Errorf("could not calclate kcoin parity: %w", err)
		}

		if parity == nil {
			return nil, fmt.Errorf("parity for %v is nil", arbitrageDenom.KCoin)
		}

		balances = append(balances, &types.ArbitrageBalance{
			Name:          arbitrageDenom.DexDenom,
			CAsset:        arbitrageDenom.CAsset,
			KCoin:         arbitrageDenom.KCoin,
			Base:          cAsset.BaseDexDenom,
			Supply:        k.BankKeeper.GetSupply(ctx, arbitrageDenom.DexDenom).Amount.String(),
			TokenValue:    tokenValue.String(),
			TokenValueUsd: tokenValueUSD.String(),
			VaultCAsset:   coins.AmountOf(arbitrageDenom.CAsset).String(),
			VaultKCoin:    k.getArbitrageDenom(ctx, arbitrageDenom.DexDenom).KCoinAmount.String(),
			Parity:        parity.String(),
		})
	}

	return &types.QueryArbitrageBalancesResponse{
		Balances:      balances,
		TotalValueUsd: totalValueUSD.String(),
	}, nil
}

func (k Keeper) ArbitrageBalanceAddress(ctx context.Context, req *types.QueryArbitrageBalancesAddressRequest) (*types.QueryArbitrageBalancesAddressResponse, error) {
	userAcc, err := sdk.AccAddressFromBech32(req.Address)
	if err != nil {
		return nil, err
	}

	referenceDenom, err := k.DexKeeper.GetHighestUSDReference(ctx)
	if err != nil {
		return nil, fmt.Errorf("could not get reference denom: %w", err)
	}

	var (
		moduleAcc   = k.AccountKeeper.GetModuleAccount(ctx, types.PoolArbitrage)
		moduleCoins = k.BankKeeper.SpendableCoins(ctx, moduleAcc.GetAddress())
		userCoins   = k.BankKeeper.SpendableCoins(ctx, userAcc)

		cAsset           denomtypes.CAsset
		balances         []*types.ArbitrageBalanceAddress
		tokenValueCAsset math.LegacyDec
		parity           *math.LegacyDec

		supply                        math.Int
		userBalanceArbitrage          math.Int
		userBalanceCAsset             math.Int
		userBalanceBase               math.Int
		userBalanceBaseArbRedeemed    math.Int
		userBalanceBaseArbRedeemedUSD math.LegacyDec

		userBalanceCAssetUSD math.LegacyDec
		userBalanceBaseUSD   math.LegacyDec
	)

	for _, arbitrageDenom := range k.DenomKeeper.GetArbitrageDenoms(ctx) {
		supply = k.BankKeeper.GetSupply(ctx, arbitrageDenom.DexDenom).Amount

		cAsset, err = k.DenomKeeper.GetCAsset(ctx, arbitrageDenom.CAsset)
		if err != nil {
			return nil, fmt.Errorf("could not get cAsset: %w", err)
		}

		tokenValueCAsset, err = k.calculateArbitrageTokenValue(ctx, arbitrageDenom).get()
		if err != nil {
			return nil, fmt.Errorf("could not calculate arbcoin value of %v: %w", arbitrageDenom.DexDenom, err)
		}

		var userShare math.LegacyDec
		if supply.IsZero() {
			userShare = math.LegacyZeroDec()
		} else {
			userShare = userCoins.AmountOf(arbitrageDenom.DexDenom).ToLegacyDec().Quo(supply.ToLegacyDec()) // C
		}

		userShareCAssetValue := userShare.Mul(tokenValueCAsset)

		parity, _, err = k.DexKeeper.CalculateParity(ctx, arbitrageDenom.KCoin)
		if err != nil {
			return nil, fmt.Errorf("could not calclate kcoin parity: %w", err)
		}

		if parity == nil {
			return nil, fmt.Errorf("parity for %v is nil", arbitrageDenom.KCoin)
		}

		userBalanceCAsset = userCoins.AmountOf(arbitrageDenom.CAsset)
		userBalanceCAssetUSD, err = k.DexKeeper.GetValueIn(ctx, cAsset.DexDenom, referenceDenom, userBalanceCAsset.ToLegacyDec())
		if err != nil {
			return nil, fmt.Errorf("could not get usd value of c asset balance: %w", err)
		}

		userBalanceBase = userCoins.AmountOf(cAsset.BaseDexDenom)
		userBalanceBaseUSD, err = k.DexKeeper.GetValueIn(ctx, cAsset.DexDenom, referenceDenom, userBalanceBase.ToLegacyDec())
		if err != nil {
			return nil, fmt.Errorf("could not get usd value of base: %w", err)
		}

		userBalanceArbitrage = userCoins.AmountOf(arbitrageDenom.DexDenom)
		userBalanceBaseArbRedeemed = k.arbitrageUserBaseValue(ctx, cAsset, userShareCAssetValue)
		userBalanceBaseArbRedeemedUSD, err = k.DexKeeper.GetValueIn(ctx, cAsset.BaseDexDenom, referenceDenom, userBalanceBaseArbRedeemed.ToLegacyDec())
		if err != nil {
			return nil, fmt.Errorf("could not get usd value of base arb redeemed: %w", err)
		}

		kCoinAmount := k.getArbitrageDenom(ctx, arbitrageDenom.DexDenom).KCoinAmount

		balances = append(balances, &types.ArbitrageBalanceAddress{
			Name:                          arbitrageDenom.DexDenom,
			CAsset:                        arbitrageDenom.CAsset,
			KCoin:                         arbitrageDenom.KCoin,
			Base:                          cAsset.BaseDexDenom,
			Supply:                        k.BankKeeper.GetSupply(ctx, arbitrageDenom.DexDenom).Amount.String(),
			VaultCAsset:                   moduleCoins.AmountOf(arbitrageDenom.CAsset).String(),
			VaultKCoin:                    kCoinAmount.String(),
			Parity:                        parity.String(),
			UserBalanceArbitrage:          userBalanceArbitrage.String(),
			UserBalanceArbitrageUsd:       userBalanceBaseArbRedeemedUSD.String(),
			UserBalanceCAsset:             userBalanceCAsset.String(),
			UserBalanceCAssetUsd:          userBalanceCAssetUSD.String(),
			UserBalanceBase:               userBalanceBase.String(),
			UserBalanceBaseUsd:            userBalanceBaseUSD.String(),
			UserBalanceBaseArbRedeemed:    userBalanceBaseArbRedeemed.String(),
			UserBalanceBaseArbRedeemedUsd: userBalanceBaseArbRedeemedUSD.String(),
		})
	}

	return &types.QueryArbitrageBalancesAddressResponse{
		Balances: balances,
	}, nil
}

func (k Keeper) arbitrageUserBaseValue(ctx context.Context, cAsset denomtypes.CAsset, userShareCAssetValue math.LegacyDec) math.Int {
	baseValue := k.MMKeeper.CalculateCAssetRedemptionValue(ctx, cAsset).Mul(userShareCAssetValue)
	return baseValue.TruncateInt()
}
