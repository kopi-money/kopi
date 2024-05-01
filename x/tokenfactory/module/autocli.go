package tokenfactory

import (
	autocliv1 "cosmossdk.io/api/cosmos/autocli/v1"

	modulev1 "github.com/kopi-money/kopi/api/kopi/tokenfactory"
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
					RpcMethod: "UpdateFeeAmount",
					Skip:      true, // skipped because authority gated
				},
				// this line is used by ignite scaffolding # autocli/tx

				{
					RpcMethod: "CreateDenom",
					Use:       "create-denom [denom]",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{
							ProtoField: "denom",
						},
					},
				},
				{
					RpcMethod: "ChangeAdmin",
					Use:       "create-denom [denom] [new_admin]",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{
							ProtoField: "denom",
						},
						{
							ProtoField: "new_admin",
						},
					},
				},
				{
					RpcMethod: "MintDenom",
					Use:       "mint-denom [denom] [amount] [target_address]",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{
							ProtoField: "denom",
						},
						{
							ProtoField: "amount",
						},
						{
							ProtoField: "target_address",
						},
					},
				},
				{
					RpcMethod: "BurnDenom",
					Use:       "burn-denom [denom] [amount] [target_address]",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{
							ProtoField: "denom",
						},
						{
							ProtoField: "amount",
						},
						{
							ProtoField: "target_address",
						},
					},
				},
				{
					RpcMethod: "ForceTransfer",
					Use:       "force-transfer [denom] [amount] [from_address] [target_address]",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{
							ProtoField: "denom",
						},
						{
							ProtoField: "amount",
						},
						{
							ProtoField: "from_address",
						},
						{
							ProtoField: "target_address",
						},
					},
				},
			},
		},
	}
}
