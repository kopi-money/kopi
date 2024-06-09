package types

// DONTCOVER

import (
	sdkerrors "cosmossdk.io/errors"
)

// x/swap module sentinel errors
var (
	ErrInvalidSigner = sdkerrors.Register(ModuleName, 1100, "expected gov account as only signer for proposal message")
	ErrNoKCoin       = sdkerrors.Register(ModuleName, 1101, "given denom is no kCoin")
)
