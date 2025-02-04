package keeper

import (
	"context"

	errorsmod "cosmossdk.io/errors"
	"github.com/kopi-money/kopi/x/strategies/types"
)

func (k msgServer) UpdateAutomationsCosts(ctx context.Context, req *types.MsgUpdateAutomationsCosts) (*types.Void, error) {
	if k.GetAuthority() != req.Authority {
		return nil, errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), req.Authority)
	}

	params := k.GetParams(ctx)
	params.AutomationFeeCondition = req.ConditionFee
	params.AutomationFeeAction = req.ActionFee

	if err := k.SetParams(ctx, params); err != nil {
		return nil, err
	}

	return &types.Void{}, nil
}
