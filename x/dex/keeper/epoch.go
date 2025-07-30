package keeper

import (
	"context"
	"fmt"
	"strconv"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/kopi-money/constants"
	"github.com/kopi-money/kopi/x/dex/types"
)

const MinimumPayout = 1000

func (k Keeper) CheckEpoch(ctx context.Context) error {
	if k.shouldStartNewEpoch(ctx) {
		return k.RestartEpoch(ctx)
	}

	return nil
}

func (k Keeper) shouldStartNewEpoch(ctx context.Context) bool {
	previousStartTime, has := k.epochStartTime.Get(ctx)
	if !has {
		return true
	}

	runtime := sdk.UnwrapSDKContext(ctx).BlockTime().Sub(previousStartTime.CreatedAt).Seconds()
	return uint64(runtime) >= k.getEpochLength(ctx)
}

func (k Keeper) getEpochSecondsLeft(ctx context.Context) float64 {
	epochLength := float64(k.getEpochLength(ctx))
	previousStartTime, has := k.epochStartTime.Get(ctx)
	if !has {
		return epochLength
	}

	runtime := sdk.UnwrapSDKContext(ctx).BlockTime().Sub(previousStartTime.CreatedAt).Seconds()
	return epochLength - runtime
}

func (k Keeper) RestartEpoch(ctx context.Context) error {
	if err := k.DistributeCollectedFees(ctx); err != nil {
		return fmt.Errorf("distribute collected fees: %w", err)
	}

	if err := k.CreateEpochSnapshot(ctx); err != nil {
		return fmt.Errorf("create epoch snapshot: %w", err)
	}

	blockTime := sdk.UnwrapSDKContext(ctx).BlockTime()
	k.epochStartTime.Set(ctx, types.EpochStartTime{CreatedAt: blockTime})

	return nil
}

func (k Keeper) CreateEpochSnapshot(ctx context.Context) error {
	if err := k.DeleteOldSnapshot(ctx); err != nil {
		return fmt.Errorf("delete old snapshot: %w", err)
	}

	if err := k.CreateNewSnapshot(ctx); err != nil {
		return fmt.Errorf("create new snapshot: %w", err)
	}

	return nil
}

func (k Keeper) CreateNewSnapshot(ctx context.Context) error {
	totalValue, err := k.calcNetLiquidityValue(ctx)
	if err != nil {
		return fmt.Errorf("calc net liquidity value: %w", err)
	}

	if !totalValue.IsPositive() {
		return nil
	}

	addresses, err := k.liquidityPositions.OuterKeys(ctx)
	if err != nil {
		return fmt.Errorf("get outer keys: %w", err)
	}

	for _, address := range addresses {
		iterator := k.liquidityPositions.Iterator(ctx, nil, address)
		for iterator.Valid() {
			keyValue := iterator.GetNextKeyValue()

			positionValue, err := k.calculateLiquidityValueForPosition(ctx, keyValue.Key())
			if err != nil {
				return fmt.Errorf("calculate liquidity value: %w", err)
			}

			positionShare := positionValue.Quo(totalValue) // C

			k.epochShares.Set(ctx, address, keyValue.Key(), types.EpochShares{
				Shares: positionShare,
			})
		}
	}

	k.epochSharesSum.Set(ctx, types.EpochShares{Shares: totalValue})

	return nil
}

// TODO: replace with more efficient way to delete entries
func (k Keeper) DeleteOldSnapshot(ctx context.Context) error {
	addresses, err := k.epochShares.OuterKeys(ctx)
	if err != nil {
		return fmt.Errorf("get outer keys: %w", err)
	}

	for _, address := range addresses {
		iterator := k.epochShares.Iterator(ctx, nil, address)
		for iterator.Valid() {
			keyValue := iterator.GetNextKeyValue()
			k.epochShares.Remove(ctx, address, keyValue.Key())
		}
	}

	return nil
}

func (k Keeper) DistributeCollectedFees(ctx context.Context) error {
	addresses, err := k.epochShares.OuterKeys(ctx)
	if err != nil {
		return fmt.Errorf("get outer keys: %w", err)
	}

	if len(addresses) == 0 {
		return nil
	}

	epochShareSum, err := k.getEpochSharesSum(ctx)
	if err != nil {
		return fmt.Errorf("get epoch share sum: %w", err)
	}

	epochSharesUSD, err := k.DenomKeeper.GetValueInUSD(ctx, constants.BaseCurrency, epochShareSum)
	if err != nil {
		return fmt.Errorf("get epoch shares USD: %w", err)
	}

	accFees := k.AccountKeeper.GetModuleAccount(ctx, types.PoolFeeIncome)
	accLeftovers := k.AccountKeeper.GetModuleAccount(ctx, types.PoolFeeLeftovers)
	payoutFunds := k.BankKeeper.SpendableCoins(ctx, accFees.GetAddress())
	fundsLeftover := k.BankKeeper.SpendableCoins(ctx, accLeftovers.GetAddress())

	payoutUSD, err := k.calculatePaoyutUSD(ctx, payoutFunds)
	if err != nil {
		return fmt.Errorf("calc payout USD: %w", err)
	}

	if err = k.BankKeeper.SendCoinsFromModuleToModule(ctx, types.PoolFeeLeftovers, types.PoolFeeIncome, fundsLeftover); err != nil {
		return fmt.Errorf("get leftover funds: %w", err)
	}

	sendToDex := sdk.NewCoins()
	for _, address := range addresses {
		sendToDex, err = k.HandleEpochAddress(ctx, address, epochShareSum, payoutFunds, sendToDex)
		if err != nil {
			return fmt.Errorf("handle epoch address: %w", err)
		}
	}

	if !sendToDex.IsZero() {
		if err = k.BankKeeper.SendCoinsFromModuleToModule(ctx, types.PoolFeeIncome, types.PoolLiquidity, sendToDex); err != nil {
			return fmt.Errorf("return leftover funds: %w", err)
		}
	}

	leftOvers := k.BankKeeper.SpendableCoins(ctx, accFees.GetAddress())
	if err = k.BankKeeper.SendCoinsFromModuleToModule(ctx, types.PoolFeeIncome, types.PoolFeeLeftovers, leftOvers); err != nil {
		return fmt.Errorf("return leftover funds: %w", err)
	}

	sdk.UnwrapSDKContext(ctx).EventManager().EmitEvent(
		sdk.NewEvent("epoch_payout",
			sdk.Attribute{Key: "liquidity_usd", Value: epochSharesUSD.String()},
			sdk.Attribute{Key: "payout_usd", Value: payoutUSD.String()},
		),
	)

	return nil
}

func (k Keeper) calculatePaoyutUSD(ctx context.Context, payout sdk.Coins) (math.LegacyDec, error) {
	sumUSD := math.LegacyZeroDec()
	for _, amount := range payout {
		amountUSD, err := k.DenomKeeper.GetValueInUSD(ctx, amount.Denom, amount.Amount.ToLegacyDec())
		if err != nil {
			return math.LegacyZeroDec(), fmt.Errorf("get amount USD (%v): %w", amount.Denom, err)
		}

		sumUSD = sumUSD.Add(amountUSD)
	}

	return sumUSD, nil
}

func (k Keeper) HandleEpochAddress(ctx context.Context, address string, epochShareSum math.LegacyDec, payoutFunds, sendToDex sdk.Coins) (sdk.Coins, error) {
	iterator := k.epochShares.Iterator(ctx, nil, address)

	for iterator.Valid() {
		keyValue := iterator.GetNextKeyValue()
		depositEpoch := keyValue.Value().Value()

		var err error
		sendToDex, err = k.HandleEpochDeposit(ctx, address, keyValue.Key(), epochShareSum, *depositEpoch, payoutFunds, sendToDex)
		if err != nil {
			return sdk.Coins{}, fmt.Errorf("handle epoch deposit: %w", err)
		}
	}

	return sendToDex, nil
}

func (k Keeper) HandleEpochDeposit(ctx context.Context, address string, positionIndex uint64, epochShareSum math.LegacyDec, epochDeposit types.EpochShares, payoutFunds, sendToDex sdk.Coins) (sdk.Coins, error) {
	epochPayouts := k.collectFeesForDeposit(ctx, address, positionIndex, epochShareSum, epochDeposit.Shares, payoutFunds)
	hasAutoCompound, stillExists := k.hasAutoCompound(ctx, address, positionIndex)

	coins := sdk.NewCoins()
	if hasAutoCompound {
		sendToDex = k.handleEpochDepositAutoCompound(ctx, address, positionIndex, epochPayouts, sendToDex)
	} else {
		coins = coins.Add(k.handleEpochDepositPayout(epochPayouts, stillExists)...)
	}

	if len(coins) > 0 {
		acc, _ := sdk.AccAddressFromBech32(address)
		if err := k.BankKeeper.SendCoinsFromModuleToAccount(ctx, types.PoolFeeIncome, acc, coins); err != nil {
			return sdk.Coins{}, fmt.Errorf("distribute collected fees: %w", err)
		}
	}

	k.setEpochLeftovers(ctx, address, positionIndex, epochPayouts, stillExists)

	return sendToDex, nil
}

func (k Keeper) handleEpochDepositPayout(epochPayouts *types.EpochPayouts, stillExists bool) sdk.Coins {
	coinsToSend := sdk.NewCoins()
	for _, denom := range epochPayouts.Denoms() {
		amount := epochPayouts.GetTruncated(denom)
		if stillExists && amount.LT(math.LegacyNewDec(MinimumPayout)) {
			continue
		}

		epochPayouts.AddUsage(denom, amount)
		coinsToSend = coinsToSend.Add(sdk.NewCoin(denom, amount.TruncateInt()))
	}

	return coinsToSend
}

func (k Keeper) handleEpochDepositAutoCompound(ctx context.Context, address string, positionIndex uint64, epochPayouts *types.EpochPayouts, sendToDex sdk.Coins) sdk.Coins {
	for _, denom := range epochPayouts.Denoms() {
		amount := epochPayouts.GetTruncated(denom)
		if amount.LT(math.LegacyNewDec(MinimumPayout)) {
			continue
		}

		epochPayouts.AddUsage(denom, amount)
		k.addLiquidity(ctx, denom, address, amount.TruncateInt(), nil, positionIndex)
		sendToDex = sendToDex.Add(sdk.NewCoin(denom, amount.TruncateInt()))
	}

	return sendToDex
}

func (k Keeper) collectFeesForDeposit(ctx context.Context, address string, positionIndex uint64, epochShares, depositShares math.LegacyDec, payoutFunds sdk.Coins) *types.EpochPayouts {
	previousLeftOvers := k.GetEpochLeftovers(ctx, address, positionIndex)
	newFunds := types.EpochLeftovers{}
	share := depositShares.Quo(epochShares)

	for _, coin := range payoutFunds {
		payoutAmount := coin.Amount.ToLegacyDec().Mul(share)
		newFunds = newFunds.Add(coin.Denom, payoutAmount)
	}

	rewardsUSD := k.rewardsToUSD(ctx, newFunds)
	sdk.UnwrapSDKContext(ctx).EventManager().EmitEvent(
		sdk.NewEvent("rewards_earned",
			sdk.Attribute{Key: "position_index", Value: strconv.Itoa(int(positionIndex))},
			sdk.Attribute{Key: "usd", Value: rewardsUSD.String()},
		),
	)

	return &types.EpochPayouts{
		PreviousEpoch: previousLeftOvers,
		CurrentEpoch:  newFunds,
	}
}

func (k Keeper) GetEpochLeftovers(ctx context.Context, address string, positionIndex uint64) types.EpochLeftovers {
	epochLeftovers, has := k.epochLeftovers.Get(ctx, address, positionIndex)
	if !has {
		return types.EpochLeftovers{}
	}

	return epochLeftovers
}

func (k Keeper) setEpochLeftovers(ctx context.Context, address string, positionIndex uint64, epochPayout *types.EpochPayouts, stillExists bool) {
	leftovers := epochPayout.ToLeftovers()
	if stillExists && len(leftovers.Leftovers) > 0 {
		k.epochLeftovers.Set(ctx, address, positionIndex, leftovers)
	} else {
		k.epochLeftovers.Remove(ctx, address, positionIndex)
	}
}

func (k Keeper) GetEpochShareSum(ctx context.Context) math.LegacyDec {
	epochShareSum, has := k.epochSharesSum.Get(ctx)
	if !has {
		return math.LegacyZeroDec()
	}

	return epochShareSum.Shares
}

func (k Keeper) GetEpochSharesPerAddress(ctx context.Context, address string) (list []types.EpochShares) {
	iterator := k.epochShares.Iterator(ctx, nil, address)
	for iterator.Valid() {
		list = append(list, iterator.GetNext())
	}

	return
}

func (k Keeper) GetEpochSharesAddresses(ctx context.Context) ([]string, error) {
	return k.epochShares.OuterKeys(ctx)
}

func (k Keeper) GetEpochShares(ctx context.Context, address string, positionIndex uint64) (types.EpochShares, bool) {
	return k.epochShares.Get(ctx, address, positionIndex)
}

func (k Keeper) hasAutoCompound(ctx context.Context, address string, positionIndex uint64) (bool, bool) {
	liquidityShares, has := k.liquidityPositions.Get(ctx, address, positionIndex)
	if !has {
		return false, false
	}

	return liquidityShares.AutoCompound, true
}

func (k Keeper) getEpochSharesSum(ctx context.Context) (math.LegacyDec, error) {
	sum := math.LegacyZeroDec()

	addresses, err := k.epochShares.OuterKeys(ctx)
	if err != nil {
		return sum, fmt.Errorf("error getting outer keys: %w", err)
	}

	for _, address := range addresses {
		iterator := k.epochShares.Iterator(ctx, nil, address)
		for iterator.Valid() {
			shares := iterator.GetNext()
			sum = sum.Add(shares.Shares)
		}
	}

	return sum, nil
}

func (k Keeper) rewardsToUSD(ctx context.Context, newFunds types.EpochLeftovers) math.LegacyDec {
	sum := math.LegacyZeroDec()

	for _, funds := range newFunds.Leftovers {
		valueUSD, _ := k.DenomKeeper.GetValueInUSD(ctx, funds.Denom, funds.Amount)
		sum = sum.Add(valueUSD)
	}

	return sum
}

func (k Keeper) exportEpochSharesToGenesis(ctx context.Context) (list []types.GenesisEpochShares) {
	addresses, _ := k.epochShares.OuterKeys(ctx)
	for _, address := range addresses {
		iterator := k.epochShares.Iterator(ctx, nil, address)
		for iterator.Valid() {
			keyValue := iterator.GetNextKeyValue()
			list = append(list, types.GenesisEpochShares{
				Address:       address,
				PositionIndex: keyValue.Key(),
				Shares:        keyValue.Value().Value().Shares,
			})
		}
	}

	return
}

func (k Keeper) exportEpochLeftoversToGenesis(ctx context.Context) (list []types.GenesisEpochLeftover) {
	addresses, _ := k.epochLeftovers.OuterKeys(ctx)
	for _, address := range addresses {
		iterator := k.epochLeftovers.Iterator(ctx, nil, address)
		for iterator.Valid() {
			keyValue := iterator.GetNextKeyValue()

			list = append(list, types.GenesisEpochLeftover{
				Address:       address,
				PositionIndex: keyValue.Key(),
				Leftovers:     keyValue.Value().Value().Leftovers,
			})
		}
	}

	return
}
