package ictest

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strconv"
	"testing"

	"github.com/docker/docker/client"
	"github.com/strangelove-ventures/interchaintest/v4"
	"github.com/strangelove-ventures/interchaintest/v4/chain/cosmos"
	"github.com/strangelove-ventures/interchaintest/v4/chain/cosmos/wasm"
	"github.com/strangelove-ventures/interchaintest/v4/ibc"
	"github.com/strangelove-ventures/interchaintest/v4/relayer"
	"github.com/strangelove-ventures/interchaintest/v4/testreporter"
	"github.com/strangelove-ventures/interchaintest/v4/testutil"
	"go.uber.org/zap/zaptest"

	"github.com/stretchr/testify/require"

	"github.com/wormhole-foundation/wormchain/interchaintest/guardians"
	"github.com/wormhole-foundation/wormchain/interchaintest/helpers"
	"github.com/wormhole-foundation/wormchain/interchaintest/helpers/wormchain_ibc_receiver"
	"github.com/wormhole-foundation/wormchain/interchaintest/helpers/wormhole_ibc"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
)

const CUSTOM_IBC_VERSION string = "ibc-wormhole-v1"

func createChains(t *testing.T, wormchainVersion string, guardians guardians.ValSet) []ibc.Chain {
	numWormchainVals := len(guardians.Vals)
	wormchainConfig.Images[0].Version = wormchainVersion

	// Create chain factory with wormchain
	wormchainConfig.ModifyGenesis = ModifyGenesis(votingPeriod, maxDepositPeriod, guardians, len(guardians.Vals), false)

	cf := interchaintest.NewBuiltinChainFactory(zaptest.NewLogger(t), []*interchaintest.ChainSpec{
		{
			ChainName:     "wormchain",
			ChainConfig:   wormchainConfig,
			NumValidators: &numWormchainVals,
			NumFullNodes:  &numFullNodes,
		},
		{
			Name:    "osmosis",
			Version: "v15.1.2",
			ChainConfig: ibc.ChainConfig{
				ChainID:        "osmosis-1002", // hardcoded handling in osmosis binary for osmosis-1, so need to override to something different.
				GasPrices:      "1.0uosmo",
				EncodingConfig: wasm.WasmEncoding(),
			},
		},
	})

	// Get chains from the chain factory
	chains, err := cf.Chains(t.Name())
	require.NoError(t, err)

	return chains
}

func buildInterchain(t *testing.T, chains []ibc.Chain) (context.Context, ibc.Relayer, *testreporter.RelayerExecReporter, *client.Client) {
	// Create a new Interchain object which describes the chains, relayers, and IBC connections we want to use
	ic := interchaintest.NewInterchain()

	for _, chain := range chains {
		ic.AddChain(chain)
	}

	rep := testreporter.NewNopReporter()
	eRep := rep.RelayerExecReporter(t)

	wormOsmoPath := "wormosmo"
	ctx := context.Background()
	client, network := interchaintest.DockerSetup(t)
	r := interchaintest.NewBuiltinRelayerFactory(ibc.CosmosRly, zaptest.NewLogger(t),
		relayer.StartupFlags("-b", "100"),
		relayer.CustomDockerImage("ghcr.io/cosmos/relayer", "v2.5.2", "100:1000")).Build(
		t, client, network)
	ic.AddRelayer(r, "relayer")

	ic.AddLink(interchaintest.InterchainLink{
		Chain1:  chains[1], // Osmosis
		Chain2:  chains[0], // Wormchain
		Relayer: r,
		Path:    wormOsmoPath,
	})

	err := ic.Build(ctx, eRep, interchaintest.InterchainBuildOptions{
		TestName:          t.Name(),
		Client:            client,
		NetworkID:         network,
		SkipPathCreation:  false,
		BlockDatabaseFile: interchaintest.DefaultBlockDatabaseFilepath(),
	})
	require.NoError(t, err)

	t.Cleanup(func() {
		_ = ic.Close()
	})

	// Start the relayer
	err = r.StartRelayer(ctx, eRep, wormOsmoPath)
	require.NoError(t, err)

	//interchaintest.TempDir(sui)
	t.Cleanup(
		func() {
			err := r.StopRelayer(ctx, eRep)
			if err != nil {
				t.Logf("an error occured while stopping the relayer: %s", err)
			}
		},
	)

	return ctx, r, eRep, client
}

func TestIbcReceiverHappyPath(t *testing.T) {
	// Base setup
	numVals := 2
	guardians := guardians.CreateValSet(t, numVals)
	chains := createChains(t, "v2.24.2", *guardians)
	ctx, r, eRep, _ := buildInterchain(t, chains)

	// Chains
	wormchain := chains[0].(*cosmos.CosmosChain)
	osmosis := chains[1].(*cosmos.CosmosChain)

	// Instantiate the wormchain-ibc-receiver and wormhole-ibc contracts
	wormchainReceiverContractInfo, osmosisSenderContractInfo := instantiateWormholeIbcContracts(t, ctx, wormchain, osmosis, guardians)

	// Spin up a new channel for the contracts to communicate over (this new channel will need to be whitelisted on the wormhole-ibc contract)
	err := r.LinkPath(ctx, eRep, "wormosmo", ibc.CreateChannelOptions{
		SourcePortName: osmosisSenderContractInfo.ContractInfo.IbcPortID,
		DestPortName:   wormchainReceiverContractInfo.ContractInfo.IbcPortID,
		Order:          ibc.Unordered,
		Version:        CUSTOM_IBC_VERSION,
	}, ibc.CreateClientOptions{
		TrustingPeriod: "112h",
	})
	require.NoError(t, err)

	err = r.StopRelayer(ctx, eRep)
	require.NoError(t, err)
	err = r.StartRelayer(ctx, eRep, "wormosmo")
	require.NoError(t, err)

	// Get the new wormchain channel to receive messages from the osmosis contract
	wormholeChannelId := helpers.FindOpenChannelByVersion(t, ctx, eRep, r, wormchain, CUSTOM_IBC_VERSION).ChannelID

	// This is the channel we will send packets on from to wormhole from osmosis ibc contract
	osmosisChannelId := helpers.FindOpenChannelByVersion(t, ctx, eRep, r, osmosis, CUSTOM_IBC_VERSION).ChannelID

	// Add the new channel to the wormchain-ibc-receiver contract
	upgradeChainChannelVaa := wormchain_ibc_receiver.SubmitIbcReceiverUpdateChannelChainMsg(t,
		vaa.ChainID(OsmoChainID), wormholeChannelId,
		guardians)
	_, err = wormchain.ExecuteContract(ctx, "faucet", wormchainReceiverContractInfo.Address, upgradeChainChannelVaa)
	require.NoError(t, err)

	// Add the new channel to the osmosis wormhole-ibc contract
	upgradeChainChannelVaa = wormhole_ibc.SubmitWormholeIbcUpdateChannelChainMsg(t,
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

	postMessageTxHash, err := osmosis.ExecuteContract(ctx, "faucet", osmosisSenderContractInfo.Address,
		string(postMessageJson))
	require.NoError(t, err, "failed to execute wormhole-ibc post message")

	ibcTx, err := helpers.GetIBCTx(osmosis, postMessageTxHash)
	require.NoError(t, err, "failed to get ibc tx")

	// Poll for the receiver acknowledgement so that we can see if the packet was processed successfully
	osmosisAck, err := testutil.PollForAck(ctx, osmosis, ibcTx.Height, ibcTx.Height+10, ibcTx.Packet)
	require.NoError(t, err, "failed to poll for acknowledgement")

	var parsedAck wormchain_ibc_receiver.ReceiverAck
	err = json.Unmarshal(osmosisAck.Acknowledgement, &parsedAck)
	require.NoError(t, err, "failed to unmarshal acknowledgement")

	require.True(t, parsedAck.IsOk(), "receiver acknowledgement should be ok to signify that it was processed successfully")
}

func TestIbcReceiverWithoutReceiverWhitelist(t *testing.T) {
	// Base setup
	numVals := 2
	guardians := guardians.CreateValSet(t, numVals)
	chains := createChains(t, "v2.24.2", *guardians)
	ctx, r, eRep, _ := buildInterchain(t, chains)

	// Chains
	wormchain := chains[0].(*cosmos.CosmosChain)
	osmosis := chains[1].(*cosmos.CosmosChain)

	// Instantiate the wormchain-ibc-receiver and wormhole-ibc contracts
	wormchainReceiverContractInfo, osmosisSenderContractInfo := instantiateWormholeIbcContracts(t, ctx, wormchain, osmosis, guardians)

	// Spin up a new channel for the contracts to communicate over (this new channel will need to be whitelisted on the wormhole-ibc contract)
	err := r.LinkPath(ctx, eRep, "wormosmo", ibc.CreateChannelOptions{
		SourcePortName: osmosisSenderContractInfo.ContractInfo.IbcPortID,
		DestPortName:   wormchainReceiverContractInfo.ContractInfo.IbcPortID,
		Order:          ibc.Unordered,
		Version:        CUSTOM_IBC_VERSION,
	}, ibc.CreateClientOptions{
		TrustingPeriod: "112h",
	})
	require.NoError(t, err)

	err = r.StopRelayer(ctx, eRep)
	require.NoError(t, err)
	err = r.StartRelayer(ctx, eRep, "wormosmo")
	require.NoError(t, err)

	// This is the channel we will send packets on from Osmosis to wormhole from the osmosis ibc contract
	osmosisChannelId := helpers.FindOpenChannelByVersion(t, ctx, eRep, r, osmosis, CUSTOM_IBC_VERSION).ChannelID

	// SKIP UPGRADING THE WORMCHAIN IBC RECEIVER CONTRACT TO TEST THAT THE POST MESSAGE STILL COMPLETES

	// Add the new channel to the osmosis wormhole-ibc contract
	upgradeChainChannelVaa := wormhole_ibc.SubmitWormholeIbcUpdateChannelChainMsg(t,
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

	postMessageTxHash, err := osmosis.ExecuteContract(ctx, "faucet", osmosisSenderContractInfo.Address,
		string(postMessageJson))
	require.NoError(t, err)

	ibcTx, err := helpers.GetIBCTx(osmosis, postMessageTxHash)
	require.NoError(t, err)

	// Poll for the receiver acknowledgement so that we can see if the packet was processed successfully
	osmosisAck, err := testutil.PollForAck(ctx, osmosis, ibcTx.Height, ibcTx.Height+10, ibcTx.Packet)
	require.NoError(t, err)

	var parsedAck wormchain_ibc_receiver.ReceiverAck
	err = json.Unmarshal(osmosisAck.Acknowledgement, &parsedAck)
	require.NoError(t, err)

	require.True(t, parsedAck.IsOk(), "receiver acknowledgement should be ok to signify that it was processed successfully")
}

func TestIbcReceiverWormholeIbcState(t *testing.T) {
	// Base setup
	numVals := 2
	guardians := guardians.CreateValSet(t, numVals)
	chains := createChains(t, "v2.24.2", *guardians)
	ctx, r, eRep, _ := buildInterchain(t, chains)

	// Chains
	wormchain := chains[0].(*cosmos.CosmosChain)
	osmosis := chains[1].(*cosmos.CosmosChain)

	// Instantiate the wormchain-ibc-receiver and wormhole-ibc contracts
	wormchainReceiverContractInfo, osmosisSenderContractInfo := instantiateWormholeIbcContracts(t, ctx, wormchain, osmosis, guardians)

	// Spin up a new channel for the contracts to communicate over (this new channel will need to be whitelisted on the wormhole-ibc contract)
	err := r.LinkPath(ctx, eRep, "wormosmo", ibc.CreateChannelOptions{
		SourcePortName: osmosisSenderContractInfo.ContractInfo.IbcPortID,
		DestPortName:   wormchainReceiverContractInfo.ContractInfo.IbcPortID,
		Order:          ibc.Unordered,
		Version:        CUSTOM_IBC_VERSION,
	}, ibc.CreateClientOptions{
		TrustingPeriod: "112h",
	})
	require.NoError(t, err)

	err = r.StopRelayer(ctx, eRep)
	require.NoError(t, err)
	err = r.StartRelayer(ctx, eRep, "wormosmo")
	require.NoError(t, err)

	// Get the new wormchain channel to receive messages from the osmosis contract
	wormholeChannelId := helpers.FindOpenChannelByVersion(t, ctx, eRep, r, wormchain, CUSTOM_IBC_VERSION).ChannelID

	// This is the channel we will send packets on from to wormhole from osmosis ibc contract
	_ = helpers.FindOpenChannelByVersion(t, ctx, eRep, r, osmosis, CUSTOM_IBC_VERSION).ChannelID

	// Add the new channel to the wormchain-ibc-receiver contract
	upgradeChainChannelVaa := wormchain_ibc_receiver.SubmitIbcReceiverUpdateChannelChainMsg(t,
		vaa.ChainID(OsmoChainID), wormholeChannelId,
		guardians)
	_, err = wormchain.ExecuteContract(ctx, "faucet", wormchainReceiverContractInfo.Address, upgradeChainChannelVaa)
	require.NoError(t, err)

	// SKIPPING ADDING THE NEW CHANNEL TO THE WORMHOLE-IBC CONTRACT TO TEST THAT THE POST MESSAGE WILL NOT BE SENT

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

	_, err = osmosis.ExecuteContract(ctx, "faucet", osmosisSenderContractInfo.Address,
		string(postMessageJson))
	require.Error(t, err, "post message should fail since the wormhole-ibc contract does not have the new channel whitelisted")
}

func instantiateWormholeIbcContracts(t *testing.T, ctx context.Context,
	wormchain *cosmos.CosmosChain,
	remoteChain *cosmos.CosmosChain,
	guardians *guardians.ValSet) (helpers.ContractInfoResponse, helpers.ContractInfoResponse) {

	// Instantiate the Wormchain core contract
	coreInstantiateMsg := helpers.CoreContractInstantiateMsg(t, wormchainConfig, vaa.ChainIDWormchain, guardians)
	wormchainCoreContractInfo := helpers.StoreAndInstantiateWormholeContract(t, ctx, wormchain, "faucet", "./contracts/wormhole_core.wasm", "wormhole_core", coreInstantiateMsg, guardians)

	// Store wormhole-ibc-receiver contract on wormchain
	ibcReceiverContractCodeId := helpers.StoreContract(t, ctx, wormchain, "faucet", "./contracts/wormchain_ibc_receiver.wasm", guardians)
	ibcReceiverCodeId, err := strconv.ParseUint(ibcReceiverContractCodeId, 10, 32)
	require.NoError(t, err)

	// Migrate the core wormchain core contract to the ibc variant
	helpers.MigrateContract(t, ctx, wormchain, "faucet", wormchainCoreContractInfo.Address, fmt.Sprint(ibcReceiverCodeId), "{}", guardians)

	// Get the port id for the wormchain-ibc-receiver contract
	wormchainReceiverContractInfo := helpers.QueryContractInfo(t, wormchain, ctx, wormchainCoreContractInfo.Address)
	require.NotEmpty(t, wormchainReceiverContractInfo.ContractInfo.IbcPortID, "wormchain (wormchain-ibc-receiver) contract port id is nil")

	// Store and instantiate wormhole-ibc contract on osmosis
	senderInstantiateMsg := helpers.CoreContractInstantiateMsg(t, wormchainConfig, vaa.ChainIDWormchain, guardians)
	senderCodeId, err := remoteChain.StoreContract(ctx, "faucet", "./contracts/wormhole_ibc.wasm")
	require.NoError(t, err)
	senderContractAddr, err := remoteChain.InstantiateContract(ctx, "faucet", senderCodeId, senderInstantiateMsg, true)
	require.NoError(t, err)
	senderContractInfo := helpers.QueryContractInfo(t, remoteChain, ctx, senderContractAddr)
	require.NotEmpty(t, senderContractInfo.ContractInfo.IbcPortID, "sender (wormhole-ibc) contract port id is nil")

	return wormchainReceiverContractInfo, senderContractInfo
}
