package keeper

import (
	"context"
	"fmt"
	"strconv"

	"github.com/cosmos/cosmos-sdk/types/query"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/kopi-money/kopi/constants"
	"github.com/kopi-money/kopi/x/strategies/types"
)

func (k Keeper) AutomationsAll(ctx context.Context, req *types.QueryAutomationsAllRequest) (*types.QueryAutomationsResponse, error) {
	automations, pageRes, err := query.CollectionPaginate(
		ctx,
		k.automations,
		req.Pagination,
		func(_ uint64, automation types.Automation) (*types.Automation, error) {
			return &automation, nil
		},
	)

	if err != nil {
		return nil, fmt.Errorf("could not get orders from pagination: %w", err)
	}

	return &types.QueryAutomationsResponse{
		Automations: automations,
		Pagination:  pageRes,
	}, nil
}

func (k Keeper) AutomationsAddress(ctx context.Context, req *types.QueryAutomationsAddressRequest) (*types.QueryAutomationsResponse, error) {
	if _, err := sdk.AccAddressFromBech32(req.Address); err != nil {
		return nil, types.ErrInvalidAddress
	}

	return &types.QueryAutomationsResponse{
		Automations: k.GetAutomationsByAddress(ctx, req.Address),
	}, nil
}

func (k Keeper) AutomationsIndex(ctx context.Context, req *types.QueryAutomationsByIndex) (*types.Automation, error) {
	index, err := strconv.Atoi(req.Index)
	if err != nil {
		return nil, fmt.Errorf("invalid index: %w", err)
	}

	automation, has := k.automations.Get(ctx, uint64(index))
	if !has {
		return nil, types.ErrAutomationNotFound
	}

	return &automation, nil
}

func (k Keeper) AutomationsAddressFunds(ctx context.Context, req *types.QueryAutomationsAddressFundsRequest) (*types.QueryAutomationsAddressFundsResponse, error) {
	acc, err := sdk.AccAddressFromBech32(req.Address)
	if err != nil {
		return nil, types.ErrInvalidAddress
	}

	funds := k.GetAutomationFunds(ctx, req.Address)
	coins := k.BankKeeper.SpendableCoin(ctx, acc, constants.KUSD).Amount

	return &types.QueryAutomationsAddressFundsResponse{
		Balance: coins.String(),
		Funds:   funds.String(),
	}, nil
}

func (k Keeper) AutomationsFunds(ctx context.Context, _ *types.QueryAutomationsFundsRequest) (*types.QueryAutomationsFundsResponse, error) {
	moduleAcc := k.AccountKeeper.GetModuleAccount(ctx, types.PoolAutomationFunds)
	poolAmount := k.BankKeeper.SpendableCoin(ctx, moduleAcc.GetAddress(), constants.KUSD).Amount

	sum := math.ZeroInt()
	iterator := k.AutomationFundsIterator(ctx)
	for iterator.Valid() {
		sum = sum.Add(iterator.GetNext().Funds)
	}

	return &types.QueryAutomationsFundsResponse{
		Pool: poolAmount.String(),
		Sum:  sum.String(),
	}, nil
}

func (k Keeper) AutomationsStats(ctx context.Context, _ *types.QueryAutomationsStatsRequest) (*types.QueryAutomationsStatsResponse, error) {
	var total int64 = 0
	var active int64 = 0

	iterator := k.AutomationIterator(ctx)
	for iterator.Valid() {
		automation := iterator.GetNext()

		total++
		if automation.Active {
			active++
		}
	}

	return &types.QueryAutomationsStatsResponse{
		Total:  total,
		Active: active,
	}, nil
}

func (k Keeper) AutomationInterval(ctx context.Context, req *types.QueryAutomationsByIndex) (*types.QueryAutomationIntervalResponse, error) {
	index, err := strconv.Atoi(req.Index)
	if err != nil {
		return nil, fmt.Errorf("invalid index: %w", err)
	}

	automation, has := k.automations.Get(ctx, uint64(index))
	if !has {
		return nil, types.ErrAutomationNotFound
	}

	secondsPerBlock := k.BlockspeedKeeper.GetSecondsPerBlock(ctx)
	blockHeight := sdk.UnwrapSDKContext(ctx).BlockHeight()

	intervalInSeconds, runtimeInSeconds, expectedChecks, err := k.getIntervalCheckData(ctx, secondsPerBlock, automation, blockHeight)
	if err != nil {
		return nil, fmt.Errorf("could not get interval check data: %w", err)
	}

	return &types.QueryAutomationIntervalResponse{
		RuntimeInBlocks:   strconv.Itoa(int(blockHeight - automation.PeriodStart)),
		RuntimeInSeconds:  runtimeInSeconds.String(),
		IntervalInSeconds: intervalInSeconds.String(),
		PeriodTimeChecks:  strconv.Itoa(int(automation.PeriodTimesChecked)),
		ExpectedChecks:    expectedChecks.String(),
	}, nil
}
