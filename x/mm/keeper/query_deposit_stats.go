package keeper

import (
	"context"
	"fmt"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	denomtypes "github.com/kopi-money/kopi/x/denominations/types"
	"github.com/kopi-money/kopi/x/mm/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) GetDepositStats(ctx context.Context, _ *types.GetDepositStatsQuery) (*types.GetDepositStatsResponse, error) {
	referenceDenom, err := k.DexKeeper.GetHighestUSDReference(ctx)
	if err != nil {
		return nil, fmt.Errorf("could not get reference denom: %w", err)
	}

	var (
		acc   = k.AccountKeeper.GetModuleAccount(ctx, types.PoolVault)
		vault = k.BankKeeper.SpendableCoins(ctx, acc.GetAddress())

		totalAvailableUSD = math.LegacyZeroDec()
		totalBorrowedUSD  = math.LegacyZeroDec()
		totalRedeemingUSD = math.LegacyZeroDec()

		supplyUSD      math.LegacyDec
		availableUSD   math.LegacyDec
		borrowedUSD    math.LegacyDec
		borrowLimitUSD math.LegacyDec
		redeemingUSD   math.LegacyDec
		priceBaseUSD   math.LegacyDec
		priceCAssetUSD math.LegacyDec
	)

	var stats []*types.DepositDenomStats
	for _, cAsset := range k.DenomKeeper.GetCAssets(ctx) {
		supply := k.BankKeeper.GetSupply(ctx, cAsset.DexDenom).Amount
		supplyUSD, err = k.DexKeeper.GetValueIn(ctx, cAsset.DexDenom, referenceDenom, supply.ToLegacyDec())
		if err != nil {
			return nil, err
		}

		available := vault.AmountOf(cAsset.BaseDexDenom)
		availableUSD, err = k.DexKeeper.GetValueIn(ctx, cAsset.BaseDexDenom, referenceDenom, available.ToLegacyDec())
		if err != nil {
			return nil, err
		}

		borrowed := k.GetLoanSumWithDefault(ctx, cAsset.BaseDexDenom).LoanSum
		borrowedUSD, err = k.DexKeeper.GetValueIn(ctx, cAsset.BaseDexDenom, referenceDenom, borrowed)
		if err != nil {
			return nil, err
		}

		deposited := k.CalculateCAssetValue(ctx, cAsset)
		borrowLimit := deposited.Mul(cAsset.BorrowLimit)
		borrowLimitUSD, err = k.DexKeeper.GetValueIn(ctx, cAsset.BaseDexDenom, referenceDenom, borrowLimit)
		if err != nil {
			return nil, err
		}

		borrowLimitUsage := math.LegacyZeroDec()
		if borrowed.IsPositive() {
			borrowLimitUsage = deposited.Quo(borrowLimit) // C
		}

		redeeming := k.GetRedemptionSum(ctx, cAsset.BaseDexDenom)
		redeemingUSD, err = k.DexKeeper.GetValueIn(ctx, cAsset.BaseDexDenom, referenceDenom, redeeming.ToLegacyDec())
		if err != nil {
			return nil, err
		}

		totalAvailableUSD = totalAvailableUSD.Add(availableUSD)
		totalBorrowedUSD = totalBorrowedUSD.Add(borrowedUSD)
		totalRedeemingUSD = totalRedeemingUSD.Add(redeemingUSD)

		utilityRate := k.getUtilityRate(ctx, cAsset)
		interestRate := k.calculateInterestRate(ctx, utilityRate)

		priceBaseUSD, err = k.DexKeeper.CalculatePrice(ctx, cAsset.BaseDexDenom, referenceDenom)
		if err != nil {
			return nil, err
		}

		priceCAssetUSD, err = k.DexKeeper.CalculatePrice(ctx, cAsset.DexDenom, referenceDenom)
		if err != nil {
			return nil, err
		}

		depositStats := types.DepositDenomStats{}
		depositStats.CAssetDenom = cAsset.DexDenom
		depositStats.BaseDenom = cAsset.BaseDexDenom
		depositStats.SupplyCAsset = supply.String()
		depositStats.SupplyCAssetUsd = supplyUSD.String()
		depositStats.Available = available.String()
		depositStats.AvailableUsd = availableUSD.String()
		depositStats.Borrowed = borrowed.String()
		depositStats.BorrowedUsd = borrowedUSD.String()
		depositStats.BorrowLimit = borrowLimit.String()
		depositStats.BorrowLimitUsd = borrowLimitUSD.String()
		depositStats.BorrowLimitUsage = borrowLimitUsage.String()
		depositStats.UtilityRate = utilityRate.String()
		depositStats.InterestRate = interestRate.String()
		depositStats.PriceBaseUsd = priceBaseUSD.String()
		depositStats.PriceCAssetUsd = priceCAssetUSD.String()
		depositStats.Redeeming = redeeming.String()
		depositStats.RedeemingUsd = redeemingUSD.String()

		stats = append(stats, &depositStats)
	}

	return &types.GetDepositStatsResponse{
		Stats:             stats,
		TotalBorrowedUsd:  totalBorrowedUSD.String(),
		TotalAvailableUsd: totalAvailableUSD.String(),
		TotalDepositedUsd: totalAvailableUSD.Add(totalBorrowedUSD).String(),
		TotalRedeemingUsd: totalRedeemingUSD.String(),
	}, nil
}

func (k Keeper) GetDepositUserStats(goCtx context.Context, req *types.GetDepositUserStatsQuery) (*types.GetDepositUserStatsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	acc, err := sdk.AccAddressFromBech32(req.Address)
	if err != nil {
		return nil, types.ErrInvalidAddress
	}

	referenceDenom, err := k.DexKeeper.GetHighestUSDReference(ctx)
	if err != nil {
		return nil, fmt.Errorf("could not get reference denom: %w", err)
	}

	var (
		stats             = []*types.DepositUserStats{}
		coins             = k.BankKeeper.SpendableCoins(ctx, acc)
		totalDepositedUSD = math.LegacyZeroDec()
		totalRedeemingUSD = math.LegacyZeroDec()

		cAssetUSD    math.LegacyDec
		basePrice    math.LegacyDec
		cAssetPrice  math.LegacyDec
		redeemingUSD math.LegacyDec
	)

	for _, cAsset := range k.DenomKeeper.GetCAssets(ctx) {
		utilityRate := k.getUtilityRate(ctx, cAsset)
		interestRate := k.calculateInterestRate(ctx, utilityRate)
		cAssetSupply := k.getCAssetSupply(ctx, cAsset)
		cAssetValue := k.CalculateCAssetValue(ctx, cAsset)

		coin := coins.AmountOf(cAsset.DexDenom)
		redeeming, isRedeeming := k.redemptions.Get(ctx, cAsset.BaseDexDenom, req.Address)
		if !isRedeeming {
			redeeming.Amount = math.ZeroInt()
		}

		amountCAsset := math.LegacyNewDecFromInt(coin)
		amountBase := convertToBaseAmount(cAssetSupply.ToLegacyDec(), cAssetValue, amountCAsset)
		cAssetUSD, err = k.DexKeeper.GetValueIn(ctx, cAsset.BaseDexDenom, referenceDenom, amountBase)
		if err != nil {
			return nil, err
		}

		redeemingUSD, err = k.DexKeeper.GetValueIn(ctx, cAsset.DexDenom, referenceDenom, redeeming.Amount.ToLegacyDec())
		if err != nil {
			return nil, err
		}

		basePrice, err = k.DexKeeper.CalculatePrice(ctx, cAsset.BaseDexDenom, referenceDenom)
		if err != nil {
			return nil, err
		}

		cAssetPrice, err = k.DexKeeper.CalculatePrice(ctx, cAsset.DexDenom, referenceDenom)
		if err != nil {
			return nil, err
		}

		totalDepositedUSD = totalDepositedUSD.Add(cAssetUSD)
		totalRedeemingUSD = totalRedeemingUSD.Add(redeemingUSD)

		depositStats := types.DepositUserStats{}
		depositStats.CAssetDenom = cAsset.DexDenom
		depositStats.BaseDenom = cAsset.BaseDexDenom
		depositStats.CAssetSupply = cAssetSupply.String()
		depositStats.CAssetValue = cAssetValue.String()
		depositStats.BaseEquivalent = k.CalculateRedemptionAmount(ctx, cAsset, amountCAsset).String()
		depositStats.AmountCAsset = amountCAsset.String()
		depositStats.ValueCAssetUsd = cAssetUSD.String()
		depositStats.PriceBaseUsd = basePrice.String()
		depositStats.PriceCAssetUsd = cAssetPrice.String()
		depositStats.InterestRate = interestRate.String()
		depositStats.Redeeming = redeeming.Amount.String()
		depositStats.RedeemingUsd = redeemingUSD.String()
		depositStats.HasRedemptionRequest = isRedeeming

		stats = append(stats, &depositStats)
	}

	return &types.GetDepositUserStatsResponse{
		Stats:             stats,
		TotalDepositedUsd: totalDepositedUSD.String(),
		TotalRedeemingUsd: totalRedeemingUSD.String(),
	}, nil
}

func (k Keeper) GetDepositUserDenomStats(ctx context.Context, req *types.GetDepositUserDenomStatsQuery) (*types.DepositUserStats, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	acc, err := sdk.AccAddressFromBech32(req.Address)
	if err != nil {
		return nil, types.ErrInvalidAddress
	}

	coins := k.BankKeeper.SpendableCoins(ctx, acc)

	referenceDenom, err := k.DexKeeper.GetHighestUSDReference(ctx)
	if err != nil {
		return nil, fmt.Errorf("could not get reference denom: %w", err)
	}

	cAsset, err := k.DenomKeeper.GetCAssetByBaseName(ctx, req.Denom)
	if err != nil {
		return nil, err
	}

	utilityRate := k.getUtilityRate(ctx, cAsset)
	interestRate := k.calculateInterestRate(ctx, utilityRate)

	redeeming, found := k.redemptions.Get(ctx, cAsset.BaseDexDenom, req.Address)
	if !found {
		redeeming.Amount = math.ZeroInt()
	}

	amountCAsset := coins.AmountOf(cAsset.DexDenom)
	amountBase := k.ConvertToBaseAmount(ctx, cAsset, amountCAsset.ToLegacyDec())
	cAssetUSD, err := k.DexKeeper.GetValueIn(ctx, cAsset.BaseDexDenom, referenceDenom, amountBase)
	if err != nil {
		return nil, err
	}

	redeemingUSD, err := k.DexKeeper.GetValueIn(ctx, cAsset.DexDenom, referenceDenom, redeeming.Amount.ToLegacyDec())
	if err != nil {
		return nil, err
	}

	basePrice, err := k.DexKeeper.CalculatePrice(ctx, cAsset.BaseDexDenom, referenceDenom)
	if err != nil {
		return nil, err
	}

	cAssetPrice, err := k.DexKeeper.CalculatePrice(ctx, cAsset.DexDenom, referenceDenom)
	if err != nil {
		return nil, err
	}

	depositStats := types.DepositUserStats{}
	depositStats.CAssetDenom = cAsset.DexDenom
	depositStats.BaseDenom = cAsset.BaseDexDenom
	depositStats.BaseEquivalent = k.CalculateRedemptionAmount(ctx, cAsset, amountCAsset.ToLegacyDec()).String()
	depositStats.AmountCAsset = amountCAsset.String()
	depositStats.ValueCAssetUsd = cAssetUSD.String()
	depositStats.PriceBaseUsd = basePrice.String()
	depositStats.PriceCAssetUsd = cAssetPrice.String()
	depositStats.InterestRate = interestRate.String()
	depositStats.Redeeming = redeeming.Amount.String()
	depositStats.RedeemingUsd = redeemingUSD.String()

	return &depositStats, nil
}

func (k Keeper) getUtilityRate(ctx context.Context, cAsset *denomtypes.CAsset) math.LegacyDec {
	available := k.GetVaultAmount(ctx, cAsset)
	totalBorrowed := k.GetLoanSumWithDefault(ctx, cAsset.BaseDexDenom).LoanSum

	utilityRate := math.LegacyZeroDec()

	if available.ToLegacyDec().Add(totalBorrowed).IsPositive() {
		utilityRate = totalBorrowed.Quo(available.ToLegacyDec().Add(totalBorrowed)) // C
	}

	return utilityRate
}
