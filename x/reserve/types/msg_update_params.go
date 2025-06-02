package types

import (
	"fmt"

	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	denomtypes "github.com/kopi-money/kopi/x/denominations/types"
)

var (
	_ sdk.Msg = &MsgUpdateKCoinBurnShare{}
)

func (msg *MsgUpdateKCoinBurnShare) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Authority); err != nil {
		return errorsmod.Wrap(err, "invalid authority address")
	}

	if err := denomtypes.IsDec(msg.KcoinBurnShare, math.LegacyZeroDec()); err != nil {
		return fmt.Errorf("kcoin_burn_share: %w", err)
	}

	return nil
}
