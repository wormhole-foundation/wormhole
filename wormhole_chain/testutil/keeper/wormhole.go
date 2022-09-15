package keeper

import (
	"io/ioutil"
	"testing"
	"time"

	"github.com/CosmWasm/wasmd/x/wasm"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/store"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/version"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	capabilitykeeper "github.com/cosmos/cosmos-sdk/x/capability/keeper"
	capabilitytypes "github.com/cosmos/cosmos-sdk/x/capability/types"
	"github.com/wormhole-foundation/wormhole-chain/app"
	"github.com/wormhole-foundation/wormhole-chain/x/wormhole/keeper"
	"github.com/wormhole-foundation/wormhole-chain/x/wormhole/types"

	paramskeeper "github.com/cosmos/cosmos-sdk/x/params/keeper"
	paramstypes "github.com/cosmos/cosmos-sdk/x/params/types"
	"github.com/stretchr/testify/require"
	"github.com/tendermint/spm/cosmoscmd"
	"github.com/tendermint/tendermint/libs/log"
	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"
	tmdb "github.com/tendermint/tm-db"
)

func WormholeKeeper(t testing.TB) (*keeper.Keeper, sdk.Context) {
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
	stateStore.MountStoreWithDB(keys[authtypes.StoreKey], sdk.StoreTypeIAVL, db)
	stateStore.MountStoreWithDB(keys[paramstypes.StoreKey], sdk.StoreTypeIAVL, db)
	stateStore.MountStoreWithDB(keys[capabilitytypes.StoreKey], sdk.StoreTypeIAVL, db)
	stateStore.MountStoreWithDB(keys[types.StoreKey], sdk.StoreTypeIAVL, db)
	stateStore.MountStoreWithDB(keys[wasmtypes.StoreKey], sdk.StoreTypeIAVL, db)
	stateStore.MountStoreWithDB(memKeys[types.MemStoreKey], sdk.StoreTypeMemory, nil)
	stateStore.MountStoreWithDB(tkeys[paramstypes.TStoreKey], sdk.StoreTypeTransient, nil)
	require.NoError(t, stateStore.LoadLatestVersion())

	encodingConfig := cosmoscmd.MakeEncodingConfig(app.ModuleBasics)
	appCodec := encodingConfig.Marshaler
	amino := encodingConfig.Amino
	// txDecoder := encodingConfig.TxConfig.TxDecoder()

	paramsKeeper := paramskeeper.NewKeeper(appCodec, amino, keys[paramstypes.StoreKey], tkeys[paramstypes.TStoreKey])
	paramsKeeper.Subspace(types.ModuleName)
	paramsKeeper.Subspace(wasm.ModuleName)

	paramsKeeper.Subspace(authtypes.ModuleName)
	subspace_auth, _ := paramsKeeper.GetSubspace(authtypes.ModuleName)
	accountKeeper := authkeeper.NewAccountKeeper(
		appCodec, keys[authtypes.StoreKey], subspace_auth, authtypes.ProtoBaseAccount, maccPerms,
	)
	// this line is used by starport scaffolding # stargate/app/paramSubspace

	subspaceWasmd, _ := paramsKeeper.GetSubspace(wasmtypes.ModuleName)

	bApp := baseapp.NewBaseApp("wormchain", log.NewNopLogger(), db, encodingConfig.TxConfig.TxDecoder())
	// bApp.SetCommitMultiStoreTracer(traceStore)
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
	)

	supportedFeatures := "iterator,staking,stargate"
	appapp.WormholeKeeper = *k

	appapp.CapabilityKeeper = capabilitykeeper.NewKeeper(appCodec, keys[capabilitytypes.StoreKey], memKeys[capabilitytypes.MemStoreKey])
	scopedWasmKeeper := appapp.CapabilityKeeper.ScopeToModule(wasm.ModuleName)

	wasmDir, err := ioutil.TempDir("", "")
	if err != nil {
		panic(err)
	}
	wasmKeeper := wasm.NewKeeper(
		appCodec,
		keys[wasmtypes.StoreKey],
		subspaceWasmd,
		&app.AccountKeeperHandler{AccountKeeper: accountKeeper},
		&app.BankKeeperHandler{},
		&app.StakingKeeperHandler{},
		&app.DistributionKeeperHandler{},
		&app.ChannelKeeperHandler{},
		&app.PortKeeperHandler{},
		scopedWasmKeeper,
		&app.ICS20TransferPortSourceHandler{},
		appapp.MsgServiceRouter(),
		appapp.GRPCQueryRouter(),
		wasmDir,
		wasm.DefaultWasmConfig(),
		// wasmConfig.ToWasmConfig(),
		supportedFeatures,
	)
	ctx := sdk.NewContext(stateStore, tmproto.Header{
		Time: time.Now(),
	}, false, log.NewNopLogger())
	appapp.MountKVStores(keys)
	appapp.MountTransientStores(tkeys)
	appapp.MountMemoryStores(memKeys)

	wasmGenState := wasmtypes.GenesisState{}
	wasmGenState.Params.CodeUploadAccess = wasmtypes.DefaultUploadAccess
	wasmGenState.Params.InstantiateDefaultPermission = wasmtypes.AccessTypeEverybody
	wasmKeeper.SetParams(ctx, wasmGenState.Params)
	appapp.WormholeKeeper.SetWasmdKeeper(wasmKeeper)
	k.SetWasmdKeeper(wasmKeeper)

	return k, ctx
}
