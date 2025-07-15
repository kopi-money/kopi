package types

import (
	"fmt"

	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	denomtypes "github.com/kopi-money/kopi/x/denominations/types"
)

var (
	_ sdk.Msg = &MsgPayFactoryReward{}
	_ sdk.Msg = &MsgPayKCoinReward{}
)

func (msg *MsgPayFactoryReward) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Creator); err != nil {
		return errorsmod.Wrap(err, "invalid creator address")
	}

	if err := ValidateDenomName(msg.FullFactoryDenomName); err != nil {
		return fmt.Errorf("full_factory_denom_name: %w", err)
	}

	if err := denomtypes.IsInt(msg.Amount, math.ZeroInt()); err != nil {
		return fmt.Errorf("factory_denom_amount: %w", err)
	}

	return nil
}

func (msg *MsgPayKCoinReward) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Creator); err != nil {
		return errorsmod.Wrap(err, "invalid creator address")
	}

	if err := ValidateDenomName(msg.FullFactoryDenomName); err != nil {
		return fmt.Errorf("full_factory_denom_name: %w", err)
	}

	if err := denomtypes.IsInt(msg.Amount, math.ZeroInt()); err != nil {
		return fmt.Errorf("factory_denom_amount: %w", err)
	}

	return nil
}
