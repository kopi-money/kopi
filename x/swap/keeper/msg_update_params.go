package keeper

import (
	"context"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"cosmossdk.io/math"

	errorsmod "cosmossdk.io/errors"
	"github.com/kopi-money/kopi/x/swap/types"
)

func (k msgServer) UpdateStakingShare(goCtx context.Context, req *types.MsgUpdateStakingShare) (*types.Void, error) {
	if k.GetAuthority() != req.Authority {
		return nil, errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), req.Authority)
	}

	ctx := startTX(sdk.UnwrapSDKContext(goCtx))
	defer k.CommitToCache(ctx)
	defer k.CommitToDB(ctx)

	stakingShare, err := math.LegacyNewDecFromStr(req.StakingShare)
	if err != nil {
		return nil, err
	}

	params := k.GetParams(ctx)
	params.StakingShare = stakingShare

	if err = params.Validate(); err != nil {
		return nil, err
	}

	if err = k.SetParams(ctx, params); err != nil {
		return nil, err
	}

	return &types.Void{}, nil
}
