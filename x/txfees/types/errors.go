package types

// DONTCOVER

import (
	sdkerrors "cosmossdk.io/errors"
)

// x/txfees module sentinel errors
var (
	ErrInvalidSigner   = sdkerrors.Register(ModuleName, 1100, "expected gov account as only signer for proposal message")
	ErrTooManyFeeCoins = sdkerrors.Register(ModuleName, 1101, "too many fee coins. only accepts fees in one denom")
	ErrInvalidFeeToken = sdkerrors.Register(ModuleName, 1102, "invalid fee token")
)
