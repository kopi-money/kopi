package keeper

import (
	"context"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"time"

	"cosmossdk.io/math"
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

	startTime := sdk.UnwrapSDKContext(ctx).BlockTime()
	vestedUntil := time.UnixMilli(msg.VestedUntil)

	for _, receiver := range msg.Receivers {
		if _, err := sdk.AccAddressFromBech32(receiver); err != nil {
			return nil, types.ErrInvalidAddress
		}

		if err := k.createVesting(ctx, factoryDenom.Admin, receiver, factoryDenom.FullName, vestingAmount, startTime, vestedUntil, msg.NumUnlockSteps); err != nil {
			return nil, err
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
