package types

import (
	"fmt"
	"github.com/kopi-money/kopi/constants"
	"regexp"
	"strings"
	"unicode"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	denomtypes "github.com/kopi-money/kopi/x/denominations/types"
)

var (
	_ sdk.Msg = &MsgCreateDenom{}
	_ sdk.Msg = &MsgChangeAdmin{}
	_ sdk.Msg = &MsgUpdateDescription{}

	hashRegex = regexp.MustCompile(`^[A-F0-9]{64}$`)
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

	if err := isValidSymbol(msg.Symbol); err != nil {
		return fmt.Errorf("invalid symbol: %w", err)
	}

	if err := isValidDisplayName(msg.Name); err != nil {
		return fmt.Errorf("invalid name: %w", err)
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

func isValidSymbol(symbol string) error {
	return isValidName(symbol, 6)
}

func isValidDisplayName(displayName string) error {
	return isValidName(displayName, 12)
}

func isValidName(text string, maxLength int) error {
	if len(text) < 3 {
		return fmt.Errorf("must contain at least 3 characters")
	}

	if len(text) > maxLength {
		return fmt.Errorf("must not contain more than %v characters", maxLength)
	}

	for _, char := range text {
		if !unicode.IsLetter(char) {
			return fmt.Errorf("must only contain letters")
		}
	}

	return nil
}
