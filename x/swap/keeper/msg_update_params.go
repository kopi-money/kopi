package keeper

import (
	"context"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/kopi-money/kopi/cache"

	"cosmossdk.io/math"

	errorsmod "cosmossdk.io/errors"
	"github.com/kopi-money/kopi/x/swap/types"
)

func (k msgServer) UpdateStakingShare(goCtx context.Context, req *types.MsgUpdateStakingShare) (*types.Void, error) {
	err := cache.Transact(goCtx, func(ctx sdk.Context) error {
		if k.GetAuthority() != req.Authority {
			return errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), req.Authority)
		}

		stakingShare, err := math.LegacyNewDecFromStr(req.StakingShare)
		if err != nil {
			return err
		}

		params := k.GetParams(ctx)
		params.StakingShare = stakingShare

		if err = params.Validate(); err != nil {
			return err
		}

		if err = k.SetParams(ctx, params); err != nil {
			return err
		}
		return nil
	})

	return &types.Void{}, err
}
