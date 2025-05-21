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
		&MsgDexAddDenom{},
		&MsgDexUpdateMinimumTradeLiquidity{},
		&MsgDexUpdateMinimumDexLiquidity{},
		&MsgDexUpdateMinimumOrderSize{},

		&MsgKCoinAddDenom{},
		&MsgKCoinAddReferences{},
		&MsgKCoinRemoveReferences{},
		&MsgKCoinUpdateSupplyLimit{},
		&MsgKCoinUpdateMintAmount{},
		&MsgKCoinUpdateBurnAmount{},

		&MsgCollateralAddDenom{},
		&MsgCollateralUpdateDepositLimit{},
		&MsgCollateralUpdateLTV{},

		&MsgCAssetAddDenom{},
		&MsgCAssetUpdateDexFeeShare{},
		&MsgCAssetUpdateBorrowLimit{},
		&MsgCAssetUpdateMinimumLoanSize{},

		&MsgAddArbitrageDenom{},
		&MsgArbitrageUpdateBuyThreshold{},
		&MsgArbitrageUpdateSellThreshold{},
		&MsgArbitrageUpdateBuyAmount{},
		&MsgArbitrageUpdateSellAmount{},
		&MsgArbitrageUpdateRedemptionFee{},
		&MsgArbitrageUpdateRedemptionFeeReserveShare{},
	)
	msgservice.RegisterMsgServiceDesc(registry, &_Msg_serviceDesc)
}
