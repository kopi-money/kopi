package keeper

import (
	"context"
	"fmt"

	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/kopi-money/kopi/x/tokenfactory/types"
)

func (k msgServer) CreateDenom(ctx context.Context, msg *types.MsgCreateDenom) (*types.Void, error) {
	factoryDenom, err := k.Keeper.CreateDenom(ctx, msg.Creator, msg.Name, msg.Symbol, msg.CreationFeeDenom, msg.Description, msg.Website, msg.IconHash, msg.LocalName, msg.Exponent, msg.CategoryIndex, msg.Mintable)
	if err != nil {
		return nil, fmt.Errorf("create denom: %v", err)
	}

	sdk.UnwrapSDKContext(ctx).EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			"factory_denom_created",
			sdk.NewAttribute("factory_denom_full_name", factoryDenom.FullName),
			sdk.NewAttribute("creator", msg.Creator),
		),
	})

	if msg.InitialSupply > 0 {
		amount := math.NewInt(msg.InitialSupply)
		if err = k.mintDenom(ctx, factoryDenom, amount, msg.Creator, true); err != nil {
			return nil, fmt.Errorf("failed to mint initial supply: %w", err)
		}
	}

	return &types.Void{}, nil
}

func (k msgServer) CreateDenomAndPool(ctx context.Context, msg *types.MsgCreateDenomAndPool) (*types.Void, error) {
	factoryDenom, err := k.Keeper.CreateDenom(ctx, msg.Creator, msg.Name, msg.Symbol, msg.CreationFeeDenom, msg.Description, msg.Website, msg.IconHash, msg.LocalName, msg.Exponent, msg.CategoryIndex, msg.Mintable)
	if err != nil {
		return nil, fmt.Errorf("create denom: %v", err)
	}

	if msg.InitialSupply > 0 {
		amount := math.NewInt(msg.InitialSupply)
		if err = k.mintDenom(ctx, factoryDenom, amount, msg.Creator, true); err != nil {
			return nil, fmt.Errorf("failed to mint initial supply: %w", err)
		}
	}

	if err = k.Keeper.CreatePool(ctx, factoryDenom, msg.KCoin, msg.FactoryDenomAmount, msg.KCoinAmount, msg.PoolFee, msg.UnlockInSeconds); err != nil {
		return nil, fmt.Errorf("failed to create pool: %w", err)
	}

	return &types.Void{}, nil
}
