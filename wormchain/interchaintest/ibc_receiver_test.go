package ictest

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strconv"
	"testing"

	"github.com/strangelove-ventures/interchaintest/v4"
	"github.com/strangelove-ventures/interchaintest/v4/chain/cosmos"
	"github.com/strangelove-ventures/interchaintest/v4/ibc"
	"github.com/strangelove-ventures/interchaintest/v4/testutil"

	"github.com/stretchr/testify/require"

	"github.com/wormhole-foundation/wormchain/interchaintest/guardians"
	"github.com/wormhole-foundation/wormchain/interchaintest/helpers"
	"github.com/wormhole-foundation/wormchain/interchaintest/helpers/wormhole_ibc"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func TestWormchainIbc(t *testing.T) {
	// Base setup
	numVals := 2
	guardians := guardians.CreateValSet(t, numVals)
	chains := CreateChains(t, "v2.24.2", *guardians)
	ctx, r, eRep, _ := BuildInterchain(t, chains)

	// Chains
	wormchain := chains[0].(*cosmos.CosmosChain)
	gaia := chains[1].(*cosmos.CosmosChain)
	osmosis := chains[2].(*cosmos.CosmosChain)

	wormchainFaucetAddrBz, err := wormchain.GetAddress(ctx, "faucet")
	require.NoError(t, err)
	wormchainFaucetAddr := sdk.MustBech32ifyAddressBytes(wormchain.Config().Bech32Prefix, wormchainFaucetAddrBz)
	fmt.Println("Wormchain faucet addr: ", wormchainFaucetAddr)

	users := interchaintest.GetAndFundTestUsers(t, ctx, "default", int64(10_000_000_000), wormchain, gaia, osmosis, osmosis)
	_ = users[0] // Wormchain user
	// gaiaUser := users[1]
	osmoUser1 := users[2]
	osmoUser2 := users[3]
	// fmt.Println("Gaia user: ", gaiaUser)
	fmt.Println("Osmo user 1: ", osmoUser1)
	fmt.Println("Osmo user 2: ", osmoUser2)

	// Instantiate the Wormchain core contract
	coreInstantiateMsg := helpers.CoreContractInstantiateMsg(t, wormchainConfig, guardians)
	wormchainCoreContractInfo := helpers.StoreAndInstantiateWormholeContract(t, ctx, wormchain, "faucet", "./contracts/wormhole_core.wasm", "wormhole_core", coreInstantiateMsg, guardians)

	// Store wormhole-ibc-receiver contract on wormchain
	ibcReceiverContractCodeId := helpers.StoreContract(t, ctx, wormchain, "faucet", "./contracts/wormchain_ibc_receiver.wasm", guardians)
	ibcReceiverCodeId, err := strconv.ParseUint(ibcReceiverContractCodeId, 10, 32)
	require.NoError(t, err)

	// Migrate the core wormchain core contract to the ibc variant
	helpers.MigrateContract(t, ctx, wormchain, "faucet", wormchainCoreContractInfo.Address, fmt.Sprint(ibcReceiverCodeId), "{}", guardians)

	// Get the port id for the wormchain-ibc-receiver contract
	wormchainReceiverContractInfo := helpers.QueryContractInfo(t, wormchain, ctx, wormchainCoreContractInfo.Address)
	wormchainReceiverPortId := wormchainReceiverContractInfo.ContractInfo.IbcPortID
	require.NotEmpty(t, wormchainReceiverPortId, "wormchain (wormchain-ibc-receiver) contract port id is nil")
	fmt.Println("Wormchain receiver port id: ", wormchainReceiverPortId)

	// Store and instantiate wormhole-ibc contract on osmosis
	osmosisWormholeIbcInstantiateMsg := helpers.CoreContractInstantiateMsg(t, wormchainConfig, guardians)
	osmosisCodeId, err := osmosis.StoreContract(ctx, "faucet", "./contracts/wormhole_ibc.wasm")
	require.NoError(t, err)
	osmosisContractAddr, err := osmosis.InstantiateContract(ctx, "faucet", osmosisCodeId, osmosisWormholeIbcInstantiateMsg, true)
	require.NoError(t, err)
	osmosisSenderContractInfo := helpers.QueryContractInfo(t, osmosis, ctx, osmosisContractAddr)
	osmosisSenderPortId := osmosisSenderContractInfo.ContractInfo.IbcPortID
	require.NotEmpty(t, osmosisSenderPortId, "osmosis (wormhole-ibc) contract port id is nil")
	fmt.Println("Osmosis sender port id: ", osmosisSenderPortId)

	// Spin up a new channel for the contracts to communicate over (this new channel will need to be whitelisted on the receiver contract)
	err = r.GeneratePath(ctx, eRep, osmosis.Config().ChainID, wormchain.Config().ChainID,
		"osmoworm")
	require.NoError(t, err)

	err = r.LinkPath(ctx, eRep, "osmoworm", ibc.CreateChannelOptions{
		SourcePortName: osmosisSenderPortId,
		DestPortName:   wormchainReceiverPortId,
		Order:          ibc.Unordered,
		Version:        "ibc-wormhole-v1",
		// Override:       true,
	}, ibc.CreateClientOptions{
		TrustingPeriod: "112h",
	})
	require.NoError(t, err)

	testutil.WaitForBlocks(ctx, 2, wormchain, osmosis)

	// Get the new wormchain channel to receive messages from the osmosis contract
	wormholeChannelId := helpers.FindChannelByVersion(t, ctx, eRep, r, wormchain.Config().ChainID, "ibc-wormhole-v1").ChannelID

	// This is the channel we will send packets on from to wormhole from osmosis ibc contract
	osmosisChannelId := helpers.FindChannelByVersion(t, ctx, eRep, r, osmosis.Config().ChainID, "ibc-wormhole-v1").ChannelID

	// Add the new channel to the wormchain-ibc-receiver contract
	upgradeChainChannelVaa := helpers.SubmitIbcReceiverUpdateChannelChainMsg(t,
		vaa.ChainID(OsmoChainID), wormholeChannelId,
		guardians)
	_, err = wormchain.ExecuteContract(ctx, "faucet", wormchainReceiverContractInfo.Address, upgradeChainChannelVaa)
	require.NoError(t, err)

	// Add the new channel to the osmosis wormhole-ibc contract
	upgradeChainChannelVaa = helpers.SubmitWormholeIbcUpdateChannelChainMsg(t,
		vaa.ChainID(vaa.ChainIDWormchain), osmosisChannelId,
		guardians)
	_, err = osmosis.ExecuteContract(ctx, "faucet", osmosisSenderContractInfo.Address, upgradeChainChannelVaa)
	require.NoError(t, err)

	// Send a VAA from osmosis to wormhole
	postMessage := wormhole_ibc.ExecuteMsg{
		SubmitVAA: nil,
		PostMessage: &wormhole_ibc.ExecuteMsg_PostMessage{
			Message: wormhole_ibc.Binary(base64.StdEncoding.EncodeToString([]byte("080000000901007bfa71192f886ab6819fa4862e34b4d178962958d9b2e3d9437338c9e5fde1443b809d2886eaa69e0f0158ea517675d96243c9209c3fe1d94d5b19866654c6980000000b150000000500020001020304000000000000000000000000000000000000000000000000000000000000000000000a0261626364"))),
			Nonce:   0,
		},
		SubmitUpdateChannelChain: nil,
	}
	postMessageJson, err := json.Marshal(postMessage)
	require.NoError(t, err)
	t.Logf("Post message json: %s", postMessageJson)
	postMessageTxHash, err := osmosis.ExecuteContract(ctx, "faucet", osmosisSenderContractInfo.Address,
		string(postMessageJson))
	require.NoError(t, err)
	t.Logf("wormhole ibc post message submitted with hash: %s", postMessageTxHash)

	// tx, err := osmosis.GetTransaction(postMessageTxHash)
	// require.NoError(t, err)
	// t.Logf("tx: %v", tx)

}
