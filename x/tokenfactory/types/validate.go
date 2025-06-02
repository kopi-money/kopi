package types

import (
	"fmt"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func ValidateDenomName(name string) error {
	parts := strings.Split(name, "/")
	if len(parts) != 3 {
		return fmt.Errorf("invalid number of denom parts: %d", len(parts))
	}

	if _, err := sdk.AccAddressFromBech32(parts[1]); err != nil {
		return fmt.Errorf("invalid address (%v): %w", parts[1], err)
	}

	if len(parts[2]) == 0 {
		return fmt.Errorf("symbol must not be empty")
	}

	return nil
}
