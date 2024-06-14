package keeper

import (
	"context"
	"cosmossdk.io/collections"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	denomtypes "github.com/kopi-money/kopi/x/denominations/types"
	"github.com/kopi-money/kopi/x/mm/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) GetDepositStats(ctx context.Context, req *types.GetDepositStatsQuery) (*types.GetDepositStatsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	acc := k.AccountKeeper.GetModuleAccount(ctx, types.PoolVault)
	vault := k.BankKeeper.SpendableCoins(ctx, acc.GetAddress())

	totalAvailableUSD := math.LegacyZeroDec()
	totalBorrowedUSD := math.LegacyZeroDec()
	totalRedeemingUSD := math.LegacyZeroDec()

	var stats []*types.DepositDenomStats
	for _, cAsset := range k.DenomKeeper.GetCAssets(ctx) {
		supply := k.BankKeeper.GetSupply(ctx, cAsset.Name).Amount
		supplyUSD, err := k.DexKeeper.GetValueInUSD(ctx, cAsset.Name, supply.ToLegacyDec())
		if err != nil {
			return nil, err
		}

		available := vault.AmountOf(cAsset.BaseDenom)
		availableUSD, err := k.DexKeeper.GetValueInUSD(ctx, cAsset.BaseDenom, available.ToLegacyDec())
		if err != nil {
			return nil, err
		}

		borrowed := k.GetLoanSumWithDefault(ctx, cAsset.BaseDenom).LoanSum
		borrowedUSD, err := k.DexKeeper.GetValueInUSD(ctx, cAsset.BaseDenom, borrowed)
		if err != nil {
			return nil, err
		}

		deposited := k.calculateCAssetValue(ctx, cAsset)
		borrowLimit := deposited.Mul(cAsset.BorrowLimit)
		borrowLimitUSD, err := k.DexKeeper.GetValueInUSD(ctx, cAsset.BaseDenom, borrowLimit)
		if err != nil {
			return nil, err
		}

		borrowLimitUsage := math.LegacyZeroDec()
		if borrowed.GT(math.LegacyZeroDec()) {
			borrowLimitUsage = deposited.Quo(borrowLimit)
		}

		redeeming := k.GetRedemptionSum(ctx, cAsset.BaseDenom)
		redeemingUSD, err := k.DexKeeper.GetValueInUSD(ctx, cAsset.BaseDenom, redeeming.ToLegacyDec())
		if err != nil {
			return nil, err
		}

		totalAvailableUSD = totalAvailableUSD.Add(availableUSD)
		totalBorrowedUSD = totalBorrowedUSD.Add(borrowedUSD)
		totalRedeemingUSD = totalRedeemingUSD.Add(redeemingUSD)

		utilityRate := k.getUtilityRate(ctx, cAsset)
		interestRate := k.calculateInterestRate(ctx, utilityRate)

		priceBaseUSD, err := k.DexKeeper.GetPriceInUSD(ctx, cAsset.BaseDenom)
		if err != nil {
			return nil, err
		}

		priceCAssetUSD, err := k.DexKeeper.GetPriceInUSD(ctx, cAsset.Name)
		if err != nil {
			return nil, err
		}

		depositStats := types.DepositDenomStats{}
		depositStats.CAssetDenom = cAsset.Name
		depositStats.BaseDenom = cAsset.BaseDenom
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

	var coins sdk.Coins
	acc, err := sdk.AccAddressFromBech32(req.Address)
	if err != nil {
		return nil, types.ErrInvalidAddress
	}

	coins = k.BankKeeper.SpendableCoins(ctx, acc)

	totalDepositedUSD := math.LegacyZeroDec()
	totalRedeemingUSD := math.LegacyZeroDec()

	var stats []*types.DepositUserStats
	for _, cAsset := range k.DenomKeeper.GetCAssets(ctx) {
		utilityRate := k.getUtilityRate(ctx, cAsset)
		interestRate := k.calculateInterestRate(ctx, utilityRate)
		cAssetSupply := k.getCAssetSupply(ctx, cAsset)
		cAssetValue := k.calculateCAssetValue(ctx, cAsset)

		found, coin := coins.Find(cAsset.Name)
		if !found {
			coin = sdk.NewCoin(cAsset.Name, math.ZeroInt())
		}

		redeeming, found := k.redemptions.Get(ctx, collections.Join(cAsset.BaseDenom, req.Address))
		if !found {
			redeeming.Amount = math.ZeroInt()
		}

		amountCAsset := math.LegacyNewDecFromInt(coin.Amount)
		amountBase := convertToBaseAmount(cAssetSupply.ToLegacyDec(), cAssetValue, amountCAsset)
		cAssetUSD, err := k.DexKeeper.GetValueInUSD(ctx, cAsset.BaseDenom, amountBase)
		if err != nil {
			return nil, err
		}

		redeemingUSD, err := k.DexKeeper.GetValueInUSD(ctx, cAsset.Name, redeeming.Amount.ToLegacyDec())
		if err != nil {
			return nil, err
		}

		basePrice, err := k.DexKeeper.GetPriceInUSD(ctx, cAsset.BaseDenom)
		if err != nil {
			return nil, err
		}

		cAssetPrice, err := k.DexKeeper.GetPriceInUSD(ctx, cAsset.Name)
		if err != nil {
			return nil, err
		}

		totalDepositedUSD = totalDepositedUSD.Add(cAssetUSD)
		totalRedeemingUSD = totalRedeemingUSD.Add(redeemingUSD)

		depositStats := types.DepositUserStats{}
		depositStats.CAssetDenom = cAsset.Name
		depositStats.BaseDenom = cAsset.BaseDenom
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

	var coins sdk.Coins
	acc, err := sdk.AccAddressFromBech32(req.Address)
	if err != nil {
		return nil, types.ErrInvalidAddress
	}

	coins = k.BankKeeper.SpendableCoins(ctx, acc)

	cAsset, err := k.DenomKeeper.GetCAssetByBaseName(ctx, req.Denom)
	if err != nil {
		return nil, err
	}

	utilityRate := k.getUtilityRate(ctx, cAsset)
	interestRate := k.calculateInterestRate(ctx, utilityRate)

	redeeming, found := k.redemptions.Get(ctx, collections.Join(cAsset.BaseDenom, req.Address))
	if !found {
		redeeming.Amount = math.ZeroInt()
	}

	amountCAsset := coins.AmountOf(cAsset.Name)
	amountBase := k.ConvertToBaseAmount(ctx, cAsset, amountCAsset)
	cAssetUSD, err := k.DexKeeper.GetValueInUSD(ctx, cAsset.BaseDenom, amountBase)
	if err != nil {
		return nil, err
	}

	redeemingUSD, err := k.DexKeeper.GetValueInUSD(ctx, cAsset.Name, redeeming.Amount.ToLegacyDec())
	if err != nil {
		return nil, err
	}

	basePrice, err := k.DexKeeper.GetPriceInUSD(ctx, cAsset.BaseDenom)
	if err != nil {
		return nil, err
	}

	cAssetPrice, err := k.DexKeeper.GetPriceInUSD(ctx, cAsset.Name)
	if err != nil {
		return nil, err
	}

	depositStats := types.DepositUserStats{}
	depositStats.CAssetDenom = cAsset.Name
	depositStats.BaseDenom = cAsset.BaseDenom
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
	totalBorrowed := k.GetLoanSumWithDefault(ctx, cAsset.BaseDenom).LoanSum

	utilityRate := math.LegacyZeroDec()

	if available.ToLegacyDec().Add(totalBorrowed).GT(math.LegacyZeroDec()) {
		utilityRate = totalBorrowed.Quo(available.ToLegacyDec().Add(totalBorrowed))
	}

	return utilityRate
}
