package types

import (
	"fmt"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/msgservice"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	// this line is used by starport scaffolding # 1
)

func RegisterInterfaces(registry cdctypes.InterfaceRegistry) {
	// this line is used by starport scaffolding # 3

	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgAddDeposit{},
		&MsgCreateRedemptionRequest{},
		&MsgCancelRedemptionRequest{},
		&MsgUpdateRedemptionRequest{},
		&MsgAddCollateral{},
		&MsgAddCollateralForBeneficiary{},
		&MsgRemoveCollateral{},
		&MsgBorrow{},
		&MsgPartiallyRepayLoan{},
		&MsgRepayLoan{},
		&MsgUpdateCollateralDiscount{},
		&MsgUpdateInterestRateParameters{},
		&MsgUpdateRedemptionFees{},
		&MsgUpdateProtocolShare{},
	)
	msgservice.RegisterMsgServiceDesc(registry, &_Msg_serviceDesc)

	protoregistry.GlobalFiles.RangeFiles(func(fd protoreflect.FileDescriptor) bool {
		if fd.Path() == "kopi/mm/tx.proto" {
			fmt.Println("=== Methods in Msg service:")
			sd := fd.Services().ByName("Msg")
			for i := 0; i < sd.Methods().Len(); i++ {
				m := sd.Methods().Get(i)
				fmt.Println(" ->", m.Name())
			}
		}
		return true
	})
}
