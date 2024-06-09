package mm

import (
	autocliv1 "cosmossdk.io/api/cosmos/autocli/v1"

	modulev1 "github.com/kopi-money/kopi/api/kopi/mm"
)

// AutoCLIOptions implements the autocli.HasAutoCLIConfig interface.
func (am AppModule) AutoCLIOptions() *autocliv1.ModuleOptions {
	return &autocliv1.ModuleOptions{
		Query: &autocliv1.ServiceCommandDescriptor{
			Service: modulev1.Query_ServiceDesc.ServiceName,
			RpcCommandOptions: []*autocliv1.RpcCommandOptions{
				{
					RpcMethod: "Params",
					Use:       "params",
					Short:     "Shows the parameters of the module",
				},
				// this line is used by ignite scaffolding # autocli/query
			},
		},
		Tx: &autocliv1.ServiceCommandDescriptor{
			Service:              modulev1.Msg_ServiceDesc.ServiceName,
			EnhanceCustomCommand: true, // only required if you want to use the custom command
			RpcCommandOptions: []*autocliv1.RpcCommandOptions{
				{
					RpcMethod: "AddCollateral",
					Use:       "add-collateral [denom] [amount]",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{
							ProtoField: "denom",
						},
						{
							ProtoField: "amount",
						},
					},
				},
				{
					RpcMethod: "RemoveCollateral",
					Use:       "remove-collateral [denom] [amount]",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{
							ProtoField: "denom",
						},
						{
							ProtoField: "amount",
						},
					},
				},
				{
					RpcMethod: "Borrow",
					Use:       "borrow [denom] [amount]",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{
							ProtoField: "denom",
						},
						{
							ProtoField: "amount",
						},
					},
				},
				{
					RpcMethod: "PartiallyRepayLoan",
					Use:       "partially-repay-loan [denom] [amount]",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{
							ProtoField: "denom",
						},
						{
							ProtoField: "amount",
						},
					},
				},
				{
					RpcMethod: "RepayLoan",
					Use:       "repay-loan [denom]",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{
							ProtoField: "denom",
						},
					},
				},
				{
					RpcMethod: "AddDeposit",
					Use:       "add-deposit [denom] [amount]",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{
							ProtoField: "denom",
						},
						{
							ProtoField: "amount",
						},
					},
				},
				{
					RpcMethod: "CreateRedemptionRequest",
					Use:       "create-redemption-request [denom] [c_asset_amount] [fee]",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{
							ProtoField: "denom",
						},
						{
							ProtoField: "c_asset_amount",
						},
						{
							ProtoField: "fee",
						},
					},
				},
				{
					RpcMethod: "UpdateRedemptionRequest",
					Use:       "update-redemption-request [denom] [c_asset_amount] [fee]",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{
							ProtoField: "denom",
						},
						{
							ProtoField: "c_asset_amount",
						},
						{
							ProtoField: "fee",
						},
					},
				},
				{
					RpcMethod: "CancelRedemptionRequest",
					Use:       "cancel-redemption-request [denom]",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{
							ProtoField: "denom",
						},
					},
				},
				// this line is used by ignite scaffolding # autocli/tx
			},
		},
	}
}
