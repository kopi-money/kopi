package types

import (
	errorsmod "cosmossdk.io/errors"
	"fmt"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/kopi-money/kopi/constants"
	"net/url"
)

var (
	_ sdk.Msg = &MsgChangeAdmin{}
	_ sdk.Msg = &MsgUpdateDescription{}
	_ sdk.Msg = &MsgUpdateWebsite{}
	_ sdk.Msg = &MsgUpdateIconHash{}
)

func (msg *MsgUpdateDescription) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Creator); err != nil {
		return errorsmod.Wrap(err, "invalid creator address")
	}

	if err := ValidateDenomName(msg.FullFactoryDenomName); err != nil {
		return fmt.Errorf("full_factory_denom_name: %w", err)
	}

	if len(msg.Description) > constants.MaxDescriptionLength {
		return ErrDescriptionTooLong
	}

	return nil
}

func (msg *MsgUpdateWebsite) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Creator); err != nil {
		return errorsmod.Wrap(err, "invalid creator address")
	}

	if err := ValidateDenomName(msg.FullFactoryDenomName); err != nil {
		return err
	}

	if len(msg.Website) > constants.MaxWebsiteLength {
		return ErrWebsiteURLTooLong
	}

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

func (msg *MsgUpdateIconHash) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Creator); err != nil {
		return errorsmod.Wrap(err, "invalid creator address")
	}

	if err := ValidateDenomName(msg.FullFactoryDenomName); err != nil {
		return err
	}

	if !validateHash(msg.IconHash) {
		return fmt.Errorf("invalid icon hash")
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

	if msg.Creator == msg.NewAdmin {
		return fmt.Errorf("old and new address must be different")
	}

	if err := ValidateDenomName(msg.FullFactoryDenomName); err != nil {
		return err
	}

	return nil
}
