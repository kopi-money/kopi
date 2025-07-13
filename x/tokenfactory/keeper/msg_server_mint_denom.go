package keeper

import (
	"context"
	"fmt"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/kopi-money/kopi/x/tokenfactory/types"
)

func (k msgServer) MintDenom(ctx context.Context, msg *types.MsgMintDenom) (*types.Void, error) {
	factoryDenom, has := k.GetDenomByFullName(ctx, msg.FullFactoryDenomName)
	if !has {
		return nil, types.ErrDenomDoesNotExist
	}

	if factoryDenom.Admin != msg.Creator {
		return nil, types.ErrIncorrectAdmin
	}

	amount, ok := math.NewIntFromString(msg.Amount)
	if !ok {
		return nil, types.ErrInvalidAmountFormat
	}

	if err := k.mintDenom(ctx, factoryDenom, amount, msg.TargetAddress, false); err != nil {
		return nil, fmt.Errorf("failed to mint denom: %w", err)
	}

	return &types.Void{}, nil
}

func (k Keeper) mintDenom(ctx context.Context, factoryDenom types.FactoryDenom, amount math.Int, targetAddress string, onCreation bool) error {
	if !onCreation {
		if !factoryDenom.Mintable || factoryDenom.LocalName != "" {
			return types.ErrNotMintable
		}
	}

	if !amount.IsPositive() {
		return types.ErrNonPositiveAmount
	}

	coins := sdk.NewCoins(sdk.NewCoin(factoryDenom.FullName, amount))
	if err := k.BankKeeper.MintCoins(ctx, types.ModuleName, coins); err != nil {
		return err
	}

	targetAddr, err := sdk.AccAddressFromBech32(targetAddress)
	if err != nil {
		return fmt.Errorf("invalid user address: %w", err)
	}

	if err = k.BankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, targetAddr, coins); err != nil {
		return err
	}

	sdk.UnwrapSDKContext(ctx).EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			"factory_denom_coins_minted",
			sdk.NewAttribute("factory_denom_full_name", factoryDenom.FullName),
			sdk.NewAttribute("amount", amount.String()),
			sdk.NewAttribute("target_address", targetAddress),
		),
	})

	return nil
}
