package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/kopi-money/kopi/x/tokenfactory/types"
)

func (k msgServer) DisableMinting(ctx context.Context, msg *types.MsgDisableMinting) (*types.Void, error) {
	factoryDenom, has := k.GetDenomByFullName(ctx, msg.FullFactoryDenomName)
	if !has {
		return nil, types.ErrDenomDoesNotExist
	}

	if factoryDenom.Admin != msg.Creator {
		return nil, types.ErrIncorrectAdmin
	}

	if !factoryDenom.Mintable {
		return nil, types.ErrMintingAlreadyDisabled
	}

	factoryDenom.Mintable = false
	k.SetDenom(ctx, factoryDenom)

	sdk.UnwrapSDKContext(ctx).EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			"factory_denom_disabled_minting",
			sdk.NewAttribute("factory_denom_full_name", factoryDenom.FullName),
		),
	})

	return &types.Void{}, nil
}
