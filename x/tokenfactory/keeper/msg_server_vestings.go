package keeper

import (
	"context"
	"fmt"

	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/kopi-money/kopi/x/tokenfactory/types"
)

func (k msgServer) CreateVestings(ctx context.Context, msg *types.MsgCreateVestings) (*types.Void, error) {
	factoryDenom, has := k.GetDenomByFullName(ctx, msg.FullFactoryDenomName)
	if !has {
		return nil, types.ErrDenomDoesNotExists
	}

	if factoryDenom.Admin != msg.Creator {
		return nil, types.ErrIncorrectAdmin
	}

	vestingAmount, ok := math.NewIntFromString(msg.FactoryDenomAmount)
	if !ok {
		return nil, types.ErrInvalidAmountFormat
	}

	if len(msg.Receivers) == 0 {
		return nil, fmt.Errorf("list of receivers is empty")
	}

	startTime := sdk.UnwrapSDKContext(ctx).BlockTime()

	for _, receiver := range msg.Receivers {
		if _, err := sdk.AccAddressFromBech32(receiver); err != nil {
			return nil, fmt.Errorf("invalid user address: %w", err)
		}

		if err := k.createVesting(ctx, factoryDenom.Admin, receiver, factoryDenom.FullName, vestingAmount, startTime, msg.VestedUntil, msg.NumUnlockSteps); err != nil {
			return nil, fmt.Errorf("create vesting: %w", err)
		}
	}

	return &types.Void{}, nil
}

func (k msgServer) CancelVestings(ctx context.Context, msg *types.MsgCancelVestings) (*types.Void, error) {
	factoryDenom, has := k.GetDenomByFullName(ctx, msg.FullFactoryDenomName)
	if !has {
		return nil, types.ErrDenomDoesNotExists
	}

	if factoryDenom.Admin != msg.Creator {
		return nil, types.ErrIncorrectAdmin
	}

	for _, index := range msg.Indexes {
		if err := k.cancelVesting(ctx, factoryDenom.Admin, factoryDenom.FullName, index); err != nil {
			return nil, err
		}
	}

	return &types.Void{}, nil
}
