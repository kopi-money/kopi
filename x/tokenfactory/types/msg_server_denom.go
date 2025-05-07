package types

import (
	"fmt"
	"github.com/kopi-money/kopi/constants"
	"net/url"
	"regexp"
	"strings"
	"unicode"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

var (
	_ sdk.Msg = &MsgCreateDenom{}

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
		return ErrDescriptionTooLong
	}

	if len(msg.Website) > constants.MaxWebsiteLength {
		return ErrWebsiteURLTooLong
	}

	if _, err := url.Parse(msg.Website); err != nil {
		return ErrWebsiteURLInvalid
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
