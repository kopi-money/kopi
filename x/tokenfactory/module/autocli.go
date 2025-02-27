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
					RpcMethod: "CreateDenom",
					Use:       "create-denom [denom] [symbol] [icon_hash] [exponent]",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{
							ProtoField: "name",
						},
						{
							ProtoField: "symbol",
						},
						{
							ProtoField: "icon_hash",
						},
						{
							ProtoField: "exponent",
						},
					},
				},
				{
					RpcMethod: "MintDenom",
					Use:       "mint-denom [full_factory_denom_name] [amount] [target_address]",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{
							ProtoField: "full_factory_denom_name",
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
					RpcMethod: "CreatePool",
					Use:       "create-pool [full_factory_denom_name] [k_coin] [factory_denom_amount] [k_coin_amount] [pool_fee] [unlock_in_seconds]",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{
							ProtoField: "full_factory_denom_name",
						},
						{
							ProtoField: "k_coin",
						},
						{
							ProtoField: "factory_denom_amount",
						},
						{
							ProtoField: "k_coin_amount",
						},
						{
							ProtoField: "pool_fee",
						},
						{
							ProtoField: "unlock_in_seconds",
						},
					},
				},
			},
		},
	}
}
