package keeper

import (
	"os"
	"testing"
	"time"

	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/store"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/version"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	capabilitykeeper "github.com/cosmos/cosmos-sdk/x/capability/keeper"
	capabilitytypes "github.com/cosmos/cosmos-sdk/x/capability/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/wormhole-foundation/wormchain/app"
	"github.com/wormhole-foundation/wormchain/x/wormhole/keeper"
	"github.com/wormhole-foundation/wormchain/x/wormhole/types"

	tmdb "github.com/cometbft/cometbft-db"
	"github.com/cometbft/cometbft/libs/log"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	distrkeeper "github.com/cosmos/cosmos-sdk/x/distribution/keeper"
	paramskeeper "github.com/cosmos/cosmos-sdk/x/params/keeper"
	paramstypes "github.com/cosmos/cosmos-sdk/x/params/types"
	"github.com/stretchr/testify/require"
)

func WormholeKeeper(t testing.TB) (*keeper.Keeper, sdk.Context) {
	k, _, _, ctx := WormholeKeeperAndWasmd(t)
	return k, ctx
}

func WormholeKeeperAndWasmd(t testing.TB) (*keeper.Keeper, wasmkeeper.Keeper, *wasmkeeper.PermissionedKeeper, sdk.Context) {
	keys := sdk.NewKVStoreKeys(
		authtypes.StoreKey,
		paramstypes.StoreKey,
		capabilitytypes.StoreKey,
		types.StoreKey,
		wasmtypes.StoreKey,
	)
	tkeys := sdk.NewTransientStoreKeys(paramstypes.TStoreKey)
	memKeys := sdk.NewMemoryStoreKeys(capabilitytypes.MemStoreKey, types.MemStoreKey)
	maccPerms := map[string][]string{}

	db := tmdb.NewMemDB()
	stateStore := store.NewCommitMultiStore(db)
	stateStore.MountStoreWithDB(keys[authtypes.StoreKey], storetypes.StoreTypeIAVL, db)
	stateStore.MountStoreWithDB(keys[paramstypes.StoreKey], storetypes.StoreTypeIAVL, db)
	stateStore.MountStoreWithDB(keys[capabilitytypes.StoreKey], storetypes.StoreTypeIAVL, db)
	stateStore.MountStoreWithDB(keys[types.StoreKey], storetypes.StoreTypeIAVL, db)
	stateStore.MountStoreWithDB(keys[wasmtypes.StoreKey], storetypes.StoreTypeIAVL, db)
	stateStore.MountStoreWithDB(memKeys[types.MemStoreKey], storetypes.StoreTypeMemory, nil)
	stateStore.MountStoreWithDB(tkeys[paramstypes.TStoreKey], storetypes.StoreTypeTransient, nil)
	require.NoError(t, stateStore.LoadLatestVersion())

	encodingConfig := app.MakeEncodingConfig()
	appCodec := encodingConfig.Marshaler
	amino := encodingConfig.Amino

	paramsKeeper := paramskeeper.NewKeeper(appCodec, amino, keys[paramstypes.StoreKey], tkeys[paramstypes.TStoreKey])
	paramsKeeper.Subspace(types.ModuleName)
	paramsKeeper.Subspace(wasmtypes.ModuleName)
	paramsKeeper.Subspace(authtypes.ModuleName)

	govModAddress := authtypes.NewModuleAddress(govtypes.ModuleName).String()
	bech32Prefix := sdk.GetConfig().GetBech32AccountAddrPrefix()

	accountKeeper := authkeeper.NewAccountKeeper(
		appCodec,
		keys[authtypes.StoreKey],
		authtypes.ProtoBaseAccount,
		maccPerms,
		bech32Prefix,
		govModAddress,
	)

	bApp := baseapp.NewBaseApp("wormchain", log.NewNopLogger(), db, encodingConfig.TxConfig.TxDecoder())
	bApp.SetVersion(version.Version)
	bApp.SetInterfaceRegistry(encodingConfig.InterfaceRegistry)

	appapp := &app.App{
		BaseApp: bApp,
	}

	k := keeper.NewKeeper(
		appCodec,
		keys[types.StoreKey],
		memKeys[types.MemStoreKey],
		accountKeeper,
		nil,
		govModAddress,
	)

	supportedFeatures := "iterator,staking,stargate,wormhole"
	appapp.WormholeKeeper = *k

	appapp.CapabilityKeeper = capabilitykeeper.NewKeeper(appCodec, keys[capabilitytypes.StoreKey], memKeys[capabilitytypes.MemStoreKey])
	scopedWasmKeeper := appapp.CapabilityKeeper.ScopeToModule(wasmtypes.ModuleName)

	wasmDir, err := os.MkdirTemp("", "")
	if err != nil {
		panic(err)
	}

	wasmKeeper := wasmkeeper.NewKeeper(
		appCodec,
		keys[wasmtypes.StoreKey],
		accountKeeper,
		appapp.BankKeeper,
		appapp.StakingKeeper,
		distrkeeper.NewQuerier(appapp.DistrKeeper),
		appapp.HooksICS4Wrapper,
		appapp.IBCKeeper.ChannelKeeper,
		&appapp.IBCKeeper.PortKeeper,
		scopedWasmKeeper,
		appapp.TransferKeeper,
		appapp.MsgServiceRouter(),
		appapp.GRPCQueryRouter(),
		wasmDir,
		wasmtypes.DefaultWasmConfig(),
		supportedFeatures,
		govModAddress,
		wasmkeeper.WithQueryPlugins(keeper.NewCustomQueryHandler(appapp.WormholeKeeper)),
	)

	ctx := sdk.NewContext(stateStore, tmproto.Header{
		Time: time.Now(),
		// The height should be at least 1, because the allowlist antehandler
		// passes everything at height 0 for gen tx's.
		Height: 1,
	}, false, log.NewNopLogger())

	appapp.MountKVStores(keys)
	appapp.MountTransientStores(tkeys)
	appapp.MountMemoryStores(memKeys)

	wasmGenState := wasmtypes.GenesisState{}
	wasmGenState.Params.CodeUploadAccess = wasmtypes.DefaultUploadAccess
	wasmGenState.Params.InstantiateDefaultPermission = wasmtypes.AccessTypeEverybody
	wasmKeeper.SetParams(ctx, wasmGenState.Params)
	permissionedWasmKeeper := wasmkeeper.NewDefaultPermissionKeeper(wasmKeeper)
	appapp.WormholeKeeper.SetWasmdKeeper(permissionedWasmKeeper)
	k.SetWasmdKeeper(permissionedWasmKeeper)

	return k, wasmKeeper, permissionedWasmKeeper, ctx
}
