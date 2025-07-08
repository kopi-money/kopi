package keeper

import (
	"context"
	"net/url"
	"strings"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/kopi-money/kopi/constants"
	"github.com/kopi-money/kopi/x/tokenfactory/types"
)

func (k msgServer) UpdateDescription(ctx context.Context, msg *types.MsgUpdateDescription) (*types.Void, error) {
	factoryDenom, has := k.GetDenomByFullName(ctx, msg.FullFactoryDenomName)
	if !has {
		return nil, types.ErrPoolDoesNotExist
	}

	if factoryDenom.Admin != msg.Creator {
		return nil, types.ErrIncorrectAdmin
	}

	if len(msg.Description) > constants.MaxDescriptionLength {
		return nil, types.ErrDescriptionTooLong
	}

	lastChange := factoryDenom.LastDescriptionChange
	lastChange = lastChange.Add(time.Duration(k.GetParams(ctx).ChangeSecondsDescription))

	blockTime := sdk.UnwrapSDKContext(ctx).BlockTime()
	if lastChange.After(blockTime) {
		return nil, types.ErrChangeTooEarly
	}

	factoryDenom.Description = msg.Description
	factoryDenom.LastDescriptionChange = blockTime
	k.SetDenom(ctx, factoryDenom)

	return &types.Void{}, nil
}

func (k msgServer) UpdateWebsite(ctx context.Context, msg *types.MsgUpdateWebsite) (*types.Void, error) {
	factoryDenom, has := k.GetDenomByFullName(ctx, msg.FullFactoryDenomName)
	if !has {
		return nil, types.ErrPoolDoesNotExist
	}

	if factoryDenom.Admin != msg.Creator {
		return nil, types.ErrIncorrectAdmin
	}

	if len(msg.Website) > constants.MaxWebsiteLength {
		return nil, types.ErrWebsiteURLTooLong
	}

	if _, err := url.Parse(msg.Website); err != nil {
		return nil, types.ErrWebsiteURLInvalid
	}

	lastChange := factoryDenom.LastWebsiteChange
	lastChange = lastChange.Add(time.Duration(k.GetParams(ctx).ChangeSecondsWebsite))

	blockTime := sdk.UnwrapSDKContext(ctx).BlockTime()
	if lastChange.After(blockTime) {
		return nil, types.ErrChangeTooEarly
	}

	factoryDenom.Website = msg.Website
	factoryDenom.LastWebsiteChange = blockTime
	k.SetDenom(ctx, factoryDenom)

	return &types.Void{}, nil
}

func (k msgServer) UpdateIconHash(ctx context.Context, msg *types.MsgUpdateIconHash) (*types.Void, error) {
	factoryDenom, has := k.GetDenomByFullName(ctx, msg.FullFactoryDenomName)
	if !has {
		return nil, types.ErrPoolDoesNotExist
	}

	if factoryDenom.Admin != msg.Creator {
		return nil, types.ErrIncorrectAdmin
	}

	lastChange := factoryDenom.LastImageChange
	lastChange = lastChange.Add(time.Duration(k.GetParams(ctx).ChangeSecondsImage))

	blockTime := sdk.UnwrapSDKContext(ctx).BlockTime()
	if lastChange.After(blockTime) {
		return nil, types.ErrChangeTooEarly
	}

	factoryDenom.IconHash = strings.ToUpper(msg.IconHash)
	factoryDenom.LastImageChange = blockTime
	k.SetDenom(ctx, factoryDenom)

	return &types.Void{}, nil
}

func (k msgServer) ChangeAdmin(ctx context.Context, msg *types.MsgChangeAdmin) (*types.Void, error) {
	factoryDenom, has := k.GetDenomByFullName(ctx, msg.FullFactoryDenomName)
	if !has {
		return nil, types.ErrDenomDoesNotExist
	}

	if factoryDenom.Admin != msg.Creator {
		return nil, types.ErrIncorrectAdmin
	}

	if _, err := sdk.AccAddressFromBech32(msg.NewAdmin); err != nil {
		return nil, types.ErrInvalidAddress
	}

	oldAdmin := factoryDenom.Admin
	factoryDenom.Admin = msg.NewAdmin
	k.SetDenom(ctx, factoryDenom)

	sdk.UnwrapSDKContext(ctx).EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			"factory_denom_admin_change",
			sdk.NewAttribute("full_name", factoryDenom.FullName),
			sdk.NewAttribute("old_admin", oldAdmin),
			sdk.NewAttribute("new_admin", msg.NewAdmin),
		),
	})

	return &types.Void{}, nil
}
