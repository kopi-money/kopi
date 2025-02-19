package keeper

import (
	"context"
	"fmt"
	sdk "github.com/cosmos/cosmos-sdk/types"

	dexkeeper "github.com/kopi-money/kopi/x/dex/keeper"
	"github.com/kopi-money/kopi/x/reserve/types"
)

func (k msgServer) Burn(ctx context.Context, msg *types.MsgBurn) (*types.Void, error) {
	amount, err := dexkeeper.ParseAmount(msg.Amount)
	if err != nil {
		return nil, err
	}

	address, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return nil, err
	}

	coins := sdk.NewCoins(sdk.NewCoin(msg.Denom, amount))
	if err = k.BankKeeper.SendCoinsFromAccountToModule(ctx, address, types.Burner, coins); err != nil {
		return nil, fmt.Errorf("send from account to module address: %v", err)
	}

	if err = k.BankKeeper.BurnCoins(ctx, types.Burner, coins); err != nil {
		return nil, fmt.Errorf("burn coins: %v", err)
	}

	return &types.Void{}, nil
}

func (k Keeper) Burn(ctx context.Context) error {
	address := k.AccountKeeper.GetModuleAccount(ctx, types.Burner).GetAddress()
	spendableCoins := k.BankKeeper.SpendableCoins(ctx, address)

	if !spendableCoins.IsZero() {
		if err := k.BankKeeper.BurnCoins(ctx, types.Burner, spendableCoins); err != nil {
			return fmt.Errorf("burn coins: %v", err)
		}
	}

	return nil
}
