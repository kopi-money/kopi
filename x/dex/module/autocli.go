package dex

import (
	autocliv1 "cosmossdk.io/api/cosmos/autocli/v1"

	modulev1 "github.com/kopi-money/kopi/api/kopi/dex"
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
				{
					RpcMethod: "SimulateTrade",
					Use:       "simulate-trade [denom_from] [denom_to] [amount]",
					Short:     "Simulates a trade without executing it",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{
							ProtoField: "denom_from",
						},
						{
							ProtoField: "denom_to",
						},
						{
							ProtoField: "amount",
						},
					},
				},
				{
					RpcMethod: "Liquidity",
					Use:       "liquidity [denom]",
					Short:     "Return the DEX liquidity of the given denom",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{
							ProtoField: "denom",
						},
					},
				},
				{
					RpcMethod: "LiquidityPair",
					Use:       "liquidity-pair [denom]",
					Short:     "Return the DEX liquidity pair of the given denom",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{
							ProtoField: "denom",
						},
					},
				},
				{
					RpcMethod: "OrdersAddress",
					Use:       "orders-address [address]",
					Short:     "Returns all open orders for a given address",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{
							ProtoField: "address",
						},
					},
				},
				// this line is used by ignite scaffolding # autocli/query
			},
		},
		Tx: &autocliv1.ServiceCommandDescriptor{
			Service:              modulev1.Msg_ServiceDesc.ServiceName,
			EnhanceCustomCommand: true, // only required if you want to use the custom command
			RpcCommandOptions: []*autocliv1.RpcCommandOptions{
				{
					RpcMethod:      "AddLiquidity",
					Use:            "add-liquidity",
					Short:          "Send a AddLiquidity tx",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{},
				},
				{
					RpcMethod: "Trade",
					Use:       "trade [denom_from] [denom_to] [amount] [max_price] [allow_incomplete]",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{
							ProtoField: "denom_from",
						},
						{
							ProtoField: "denom_to",
						},
						{
							ProtoField: "amount",
						},
						{
							ProtoField: "allow_incomplete",
						},
						{
							ProtoField: "max_price",
							Optional:   true,
						},
					},
				},
				// this line is used by ignite scaffolding # autocli/tx
			},
		},
	}
}
