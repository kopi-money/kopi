package keeper

import (
	"context"
	"fmt"

	"cosmossdk.io/math"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/kopi-money/kopi/x/denominations/types"
)

func (k msgServer) AddDEXDenom(goCtx context.Context, req *types.MsgAddDEXDenom) (*types.MsgUpdateParamsResponse, error) {
	if k.GetAuthority() != req.Authority {
		return nil, errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), req.Authority)
	}

	ctx := startTX(sdk.UnwrapSDKContext(goCtx))
	defer k.CommitToCache(ctx)
	defer k.CommitToDB(ctx)

	params := k.GetParams(ctx)

	factor, err := math.LegacyNewDecFromStr(req.Factor)
	if err != nil {
		return nil, err
	}

	minLiquidity, ok := math.NewIntFromString(req.MinLiquidity)
	if !ok {
		return nil, fmt.Errorf("invalid min liquidity value: %v", req.MinLiquidity)
	}

	dexDenom := &types.DexDenom{
		Name:         req.Name,
		Factor:       &factor,
		MinLiquidity: minLiquidity,
	}

	params.DexDenoms = append(params.DexDenoms, dexDenom)

	if err = k.SetParams(ctx, params); err != nil {
		return nil, err
	}

	return &types.MsgUpdateParamsResponse{}, nil
}

func (k msgServer) UpdateDEXDenom(goCtx context.Context, req *types.MsgUpdateDEXDenom) (*types.MsgUpdateParamsResponse, error) {
	if k.GetAuthority() != req.Authority {
		return nil, errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), req.Authority)
	}

	ctx := startTX(sdk.UnwrapSDKContext(goCtx))
	defer k.CommitToCache(ctx)
	defer k.CommitToDB(ctx)

	params := k.GetParams(ctx)

	minLiquidity, ok := math.NewIntFromString(req.MinLiquidity)
	if !ok {
		return nil, fmt.Errorf("invalid min liquidity value: %v", req.MinLiquidity)
	}

	dexDenoms := []*types.DexDenom{}
	found := false

	for _, dexDenom := range params.DexDenoms {
		if dexDenom.Name == req.Name {
			dexDenom.MinLiquidity = minLiquidity
			found = true
		}

		dexDenoms = append(dexDenoms, dexDenom)
	}

	if !found {
		return nil, types.ErrInvalidDexAsset
	}

	params.DexDenoms = dexDenoms

	if err := k.SetParams(ctx, params); err != nil {
		return nil, err
	}

	return &types.MsgUpdateParamsResponse{}, nil
}
