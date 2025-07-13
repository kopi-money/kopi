package types

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"unicode"

	"github.com/kopi-money/kopi/constants"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

var (
	_ sdk.Msg = &MsgCreateDenom{}

	hashRegex = regexp.MustCompile(`^[A-F0-9]{64}$`)

	asciiLengthPattern = regexp.MustCompile(`^[[:ascii:]]{3,6}$`)
	splitPattern       = regexp.MustCompile(`^([a-zA-Z]{3,6})([0-9]{0,2})?$`)
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

	if err := IsValidSymbol(msg.Symbol); err != nil {
		return fmt.Errorf("invalid symbol: %w", err)
	}

	if err := IsValidDisplayName(msg.Name); err != nil {
		return fmt.Errorf("invalid name: %w", err)
	}

	if len(msg.Description) > constants.MaxDescriptionLength {
		return ErrDescriptionTooLong
	}

	if len(msg.Website) > constants.MaxWebsiteLength {
		return ErrWebsiteURLTooLong
	}

	// website is allowed to be empty, so only check the uri when it's not empty
	if msg.Website != "" {
		u, err := url.ParseRequestURI(msg.Website)
		if err != nil {
			return ErrWebsiteURLInvalid
		}

		if u.Scheme != "https" && u.Scheme != "http" {
			return ErrWebsiteURLInvalid
		}
	}

	return nil
}

func validateHash(hash string) bool {
	return hashRegex.Match([]byte(strings.ToUpper(hash)))
}

func IsValidSymbol(symbol string) error {
	if !asciiLengthPattern.MatchString(symbol) {
		return fmt.Errorf("must be 3-6 ASCII characters")
	}

	matches := splitPattern.FindStringSubmatch(symbol)
	if matches == nil {
		return fmt.Errorf("must be of 3-6 characters, optionally followed by a 0-2 digit suffix")
	}

	if len(matches[1])+len(matches[2]) > 6 {
		return fmt.Errorf("must be of 3-6 characters, optionally followed by a 0-2 digit suffix")
	}

	return nil
}

func IsValidDisplayName(displayName string) error {
	return IsValidName(displayName, 12)
}

func IsValidName(text string, maxLength int) error {
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
