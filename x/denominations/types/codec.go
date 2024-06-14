package types

import (
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/msgservice"
	// this line is used by starport scaffolding # 1
)

func RegisterInterfaces(registry cdctypes.InterfaceRegistry) {
	// this line is used by starport scaffolding # 3

	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgAddCAsset{},
		&MsgAddCollateralDenom{},
		&MsgAddDEXDenom{},
		&MsgAddKCoin{},
		&MsgAddKCoinReferences{},
		&MsgRemoveKCoinReferences{},
		&MsgUpdateCAssetDexFeeShare{},
		&MsgUpdateCollateralDenomMaxDeposit{},
		&MsgUpdateCollateralDenomLTV{},
		&MsgUpdateDEXDenomMinimumLiquidity{},
		&MsgUpdateDEXDenomMinimumOrderSize{},
		&MsgUpdateKCoinSupply{},
		&MsgUpdateKCoinMintAmount{},
		&MsgUpdateKCoinBurnAmount{},
	)
	msgservice.RegisterMsgServiceDesc(registry, &_Msg_serviceDesc)
}
