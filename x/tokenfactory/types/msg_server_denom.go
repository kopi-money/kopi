package types

import (
	"fmt"
	"github.com/kopi-money/kopi/constants"
	"regexp"
	"strings"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	denomtypes "github.com/kopi-money/kopi/x/denominations/types"
)

var hashRegex = regexp.MustCompile(`^[A-F0-9]{64}$`)

var (
	_ sdk.Msg = &MsgCreateDenom{}
	_ sdk.Msg = &MsgChangeAdmin{}
	_ sdk.Msg = &MsgUpdateDescription{}
)

func (msg *MsgCreateDenom) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Creator); err != nil {
		return errorsmod.Wrap(err, "invalid creator address")
	}

	if len(msg.Name) > constants.MaxDenomNameLength {
		return fmt.Errorf("name must not have more than 32 characters")
	}

	if !validateHash(msg.IconHash) {
		return fmt.Errorf("invalid icon hash")
	}

	if len(msg.Symbol) < 3 {
		return fmt.Errorf("symbol must contain at least 3 characters")
	}

	if len(msg.Symbol) > 6 {
		return fmt.Errorf("symbol must not contain more than 6 characters")
	}

	if len(msg.Description) > constants.MaxDescriptionLength {
		return fmt.Errorf("description must not contain more than 256 characters")
	}

	return nil
}

func (msg *MsgChangeAdmin) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Creator); err != nil {
		return errorsmod.Wrap(err, "invalid creator address")
	}

	if _, err := sdk.AccAddressFromBech32(msg.NewAdmin); err != nil {
		return errorsmod.Wrap(err, "invalid new admin address")
	}

	if err := denomtypes.ValidateDenomName(msg.FullFactoryDenomName); err != nil {
		return err
	}

	return nil
}

func (msg *MsgUpdateIconHash) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Creator); err != nil {
		return errorsmod.Wrap(err, "invalid creator address")
	}

	if err := denomtypes.ValidateDenomName(msg.FullFactoryDenomName); err != nil {
		return err
	}

	if !validateHash(msg.IconHash) {
		return fmt.Errorf("invalid icon hash")
	}

	return nil
}

func (msg *MsgUpdateDescription) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Creator); err != nil {
		return errorsmod.Wrap(err, "invalid creator address")
	}

	if err := denomtypes.ValidateDenomName(msg.FullFactoryDenomName); err != nil {
		return err
	}

	if len(msg.Description) > constants.MaxDescriptionLength {
		return fmt.Errorf("description must not contain more than 256 characters")
	}

	return nil
}

func validateHash(hash string) bool {
	return hashRegex.Match([]byte(strings.ToUpper(hash)))
}
