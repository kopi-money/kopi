package arbitrage

import (
	"context"
	"encoding/json"
	"fmt"

	denomkeeper "github.com/kopi-money/kopi/x/denominations/keeper"
	dexkeeper "github.com/kopi-money/kopi/x/dex/keeper"
	mmkeeper "github.com/kopi-money/kopi/x/mm/keeper"

	"cosmossdk.io/core/appmodule"
	"cosmossdk.io/core/store"
	"cosmossdk.io/depinject"
	"cosmossdk.io/log"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/kopi-money/kopi/cache"

	// this line is used by starport scaffolding # 1

	modulev1 "github.com/kopi-money/kopi/api/kopi/strategies/module"
	"github.com/kopi-money/kopi/x/strategies/keeper"
	"github.com/kopi-money/kopi/x/strategies/types"
)

var (
	_ module.AppModuleBasic      = (*AppModule)(nil)
	_ module.AppModuleSimulation = (*AppModule)(nil)
	_ module.HasGenesis          = (*AppModule)(nil)
	_ module.HasInvariants       = (*AppModule)(nil)
	_ module.HasConsensusVersion = (*AppModule)(nil)

	_ appmodule.AppModule       = (*AppModule)(nil)
	_ appmodule.HasBeginBlocker = (*AppModule)(nil)
	_ appmodule.HasEndBlocker   = (*AppModule)(nil)

	_ types.DenomKeeper = (*denomkeeper.Keeper)(nil)
	_ types.DexKeeper   = (*dexkeeper.Keeper)(nil)
	_ types.MMKeeper    = (*mmkeeper.Keeper)(nil)
)

// ----------------------------------------------------------------------------
// AppModuleBasic
// ----------------------------------------------------------------------------

// AppModuleBasic implements the AppModuleBasic interface that defines the
// independent methods a Cosmos SDK module needs to implement.
type AppModuleBasic struct {
	cdc codec.BinaryCodec
}

func NewAppModuleBasic(cdc codec.BinaryCodec) AppModuleBasic {
	return AppModuleBasic{cdc: cdc}
}

// Name returns the name of the module as a string.
func (AppModuleBasic) Name() string {
	return types.ModuleName
}

// RegisterLegacyAminoCodec registers the amino codec for the module, which is used
// to marshal and unmarshal structs to/from []byte in order to persist them in the module's KVStore.
func (AppModuleBasic) RegisterLegacyAminoCodec(_ *codec.LegacyAmino) {}

// RegisterInterfaces registers a module's interface types and their concrete implementations as proto.Message.
func (a AppModuleBasic) RegisterInterfaces(reg cdctypes.InterfaceRegistry) {
	types.RegisterInterfaces(reg)
}

// DefaultGenesis returns a default GenesisState for the module, marshalled to json.RawMessage.
// The default GenesisState need to be defined by the module developer and is primarily used for testing.
func (AppModuleBasic) DefaultGenesis(cdc codec.JSONCodec) json.RawMessage {
	return cdc.MustMarshalJSON(types.DefaultGenesis())
}

// ValidateGenesis used to validate the GenesisState, given in its json.RawMessage form.
func (AppModuleBasic) ValidateGenesis(cdc codec.JSONCodec, _ client.TxEncodingConfig, bz json.RawMessage) error {
	var genState types.GenesisState
	if err := cdc.UnmarshalJSON(bz, &genState); err != nil {
		return fmt.Errorf("failed to unmarshal %s genesis state: %w", types.ModuleName, err)
	}
	return genState.Validate()
}

// RegisterGRPCGatewayRoutes registers the gRPC Gateway routes for the module.
func (AppModuleBasic) RegisterGRPCGatewayRoutes(clientCtx client.Context, mux *runtime.ServeMux) {
	if err := types.RegisterQueryHandlerClient(context.Background(), mux, types.NewQueryClient(clientCtx)); err != nil {
		panic(err)
	}
}

// ----------------------------------------------------------------------------
// AppModule
// ----------------------------------------------------------------------------

// AppModule implements the AppModule interface that defines the inter-dependent methods that modules need to implement
type AppModule struct {
	AppModuleBasic

	keeper             keeper.Keeper
	accountKeeper      types.AccountKeeper
	bankKeeper         types.BankKeeper
	blockspeedKeeper   types.BlockspeedKeeper
	distributionKeeper types.DistributionKeeper

	denomKeeper types.DenomKeeper
	dexKeeper   types.DexKeeper
	mmKeeper    types.MMKeeper
}

func NewAppModule(
	cdc codec.Codec,

	keeper keeper.Keeper,
	accountKeeper types.AccountKeeper,
	bankKeeper types.BankKeeper,
	distributionKeeper types.DistributionKeeper,

	blockspeedKeeper types.BlockspeedKeeper,
	denomKeeper types.DenomKeeper,
	dexKeeper types.DexKeeper,
	mmKeeper types.MMKeeper,
) AppModule {
	return AppModule{
		AppModuleBasic: NewAppModuleBasic(cdc),

		keeper:             keeper,
		accountKeeper:      accountKeeper,
		bankKeeper:         bankKeeper,
		distributionKeeper: distributionKeeper,

		blockspeedKeeper: blockspeedKeeper,
		denomKeeper:      denomKeeper,
		dexKeeper:        dexKeeper,
		mmKeeper:         mmKeeper,
	}
}

// RegisterServices registers a gRPC query service to respond to the module-specific gRPC queries
func (am AppModule) RegisterServices(cfg module.Configurator) {
	types.RegisterMsgServer(cfg.MsgServer(), keeper.NewMsgServerImpl(am.keeper))
	types.RegisterQueryServer(cfg.QueryServer(), am.keeper)
}

// RegisterInvariants registers the invariants of the module. If an invariant deviates from its predicted value, the InvariantRegistry triggers appropriate logic (most often the chain will be halted)
func (am AppModule) RegisterInvariants(_ sdk.InvariantRegistry) {}

// InitGenesis performs the module's genesis initialization. It returns no validator updates.
func (am AppModule) InitGenesis(ctx sdk.Context, cdc codec.JSONCodec, gs json.RawMessage) {
	var genState types.GenesisState
	// Initialize global index to index in genesis state
	cdc.MustUnmarshalJSON(gs, &genState)

	InitGenesis(ctx, am.keeper, genState)
}

// ExportGenesis returns the module's exported genesis state as raw JSON bytes.
func (am AppModule) ExportGenesis(ctx sdk.Context, cdc codec.JSONCodec) json.RawMessage {
	genState := ExportGenesis(ctx, am.keeper)
	return cdc.MustMarshalJSON(genState)
}

// ConsensusVersion is a sequence number for state-breaking change of the module.
// It should be incremented on each consensus-breaking change introduced by the module.
// To avoid wrong/empty versions, the initial version should be set to 1.
func (AppModule) ConsensusVersion() uint64 { return 1 }

// BeginBlock contains the logic that is automatically triggered at the beginning of each block.
// The begin block implementation is optional.
func (am AppModule) BeginBlock(ctx context.Context) error {
	if err := am.keeper.Initialize(ctx); err != nil {
		return fmt.Errorf("could not initialize dex module: %w", err)
	}

	return nil
}

// EndBlock contains the logic that is automatically triggered at the end of each block.
// The end block implementation is optional.
func (am AppModule) EndBlock(ctx context.Context) error {
	if err := cache.TransactWithNewMultiStore(ctx, func(innerCtx context.Context) error {
		if err := am.keeper.HandleArbitrageDenoms(innerCtx); err != nil {
			return fmt.Errorf("error doing arbitrage trades: %w", err)
		}

		return nil
	}); err != nil {
		//return fmt.Errorf("error handling arbitrage denoms: %w", err)
		am.keeper.Logger().Error(fmt.Sprintf("error handling arbitrage denoms: %v", err.Error()))
	}

	if err := am.keeper.HandleAutomations(ctx); err != nil {
		return err
	}

	return nil
}

// IsOnePerModuleType implements the depinject.OnePerModuleType interface.
func (am AppModule) IsOnePerModuleType() {}

// IsAppModule implements the appmodule.AppModule interface.
func (am AppModule) IsAppModule() {}

// ----------------------------------------------------------------------------
// App Wiring Setup
// ----------------------------------------------------------------------------

func init() {
	appmodule.Register(
		&modulev1.Module{},
		appmodule.Provide(ProvideModule),
	)
}

type ModuleInputs struct {
	depinject.In

	StoreService store.KVStoreService
	Cdc          codec.Codec
	Config       *modulev1.Module
	Logger       log.Logger

	AccountKeeper      types.AccountKeeper
	BankKeeper         types.BankKeeper
	DistributionKeeper types.DistributionKeeper
	StakingKeeper      types.StakingKeeper

	BlockspeedKeeper types.BlockspeedKeeper
	DenomKeeper      types.DenomKeeper
	DexKeeper        types.DexKeeper
	MMKeeper         types.MMKeeper
}

type ModuleOutputs struct {
	depinject.Out

	ArbitrageKeeper keeper.Keeper
	Module          appmodule.AppModule
}

func ProvideModule(in ModuleInputs) ModuleOutputs {
	// default to governance authority if not provided
	authority := authtypes.NewModuleAddress(govtypes.ModuleName)
	if in.Config.Authority != "" {
		authority = authtypes.NewModuleAddressOrBech32Address(in.Config.Authority)
	}
	k := keeper.NewKeeper(
		in.Cdc,
		in.StoreService,
		in.Logger,

		in.AccountKeeper,
		in.BankKeeper,
		in.DistributionKeeper,
		in.StakingKeeper,

		in.BlockspeedKeeper,
		in.DenomKeeper,
		in.DexKeeper,
		in.MMKeeper,
		authority.String(),
	)
	m := NewAppModule(
		in.Cdc,

		k,
		in.AccountKeeper,
		in.BankKeeper,
		in.DistributionKeeper,

		in.BlockspeedKeeper,
		in.DenomKeeper,
		in.DexKeeper,
		in.MMKeeper,
	)

	return ModuleOutputs{ArbitrageKeeper: k, Module: m}
}
