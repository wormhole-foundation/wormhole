package ictest

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"cosmossdk.io/math"
	"github.com/docker/docker/client"
	"github.com/strangelove-ventures/interchaintest/v7"
	"github.com/strangelove-ventures/interchaintest/v7/chain/cosmos"
	"github.com/strangelove-ventures/interchaintest/v7/chain/cosmos/wasm"
	"github.com/strangelove-ventures/interchaintest/v7/ibc"
	interchaintestrelayer "github.com/strangelove-ventures/interchaintest/v7/relayer"
	"github.com/strangelove-ventures/interchaintest/v7/testreporter"
	"github.com/strangelove-ventures/interchaintest/v7/testutil"
	"github.com/stretchr/testify/require"
	"github.com/wormhole-foundation/wormchain/interchaintest/guardians"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
	"go.uber.org/zap/zaptest"

	transfertypes "github.com/cosmos/ibc-go/v7/modules/apps/transfer/types"
)

var GenesisWalletAmount = math.NewInt(10_000_000_000)

// creeateIbcClientUpdateVaa creates a governance VAA to update expired
// ibc clients
func createIbcClientUpdateVaa(
	subjectClientId string,
	substituteClientId string,
) ([]byte, error) {

	subjectBz := [64]byte{}
	buf, err := vaa.LeftPadBytes(subjectClientId, 64)
	if err != nil {
		return nil, err
	}
	copy(subjectBz[:], buf.Bytes())

	substituteBz := [64]byte{}
	buf, err = vaa.LeftPadBytes(substituteClientId, 64)
	if err != nil {
		return nil, err
	}
	copy(substituteBz[:], buf.Bytes())

	bodyPayload := vaa.BodyGatewayIBCClientUpdate{
		SubjectClientId:    subjectBz,
		SubstituteClientId: substituteBz,
	}

	return bodyPayload.Serialize()
}

// buildIC creates a single node cluster of wormchain and osmo
func buildIC(t *testing.T, guardians guardians.ValSet) ([]ibc.Chain, *interchaintest.Interchain, context.Context, ibc.Relayer, *testreporter.RelayerExecReporter, *client.Client, string) {
	numVals := len(guardians.Vals)
	numFull := 1

	cfg := WormchainConfig
	cfg.ModifyGenesis = ModifyGenesis(VotingPeriod, MaxDepositPeriod, guardians, numVals, false)

	cf := interchaintest.NewBuiltinChainFactory(zaptest.NewLogger(t), []*interchaintest.ChainSpec{
		{
			ChainName:     cfg.Name,
			Version:       cfg.Images[0].Version,
			ChainConfig:   cfg,
			NumValidators: &numVals,
			NumFullNodes:  &numFull,
		},
		{
			Name:    "osmosis",
			Version: "v15.1.2",
			ChainConfig: ibc.ChainConfig{
				Bech32Prefix:   "osmo",
				ChainID:        "osmosis-1002", // hardcoded handling in osmosis binary for osmosis-1, so need to override to something different.
				GasPrices:      "1.0uosmo",
				EncodingConfig: wasm.WasmEncoding(),
			},
			NumValidators: &numVals,
			NumFullNodes:  &numFull,
		},
	})

	// Get chains from the chain factory
	chains, err := cf.Chains(t.Name())
	require.NoError(t, err)

	ic := interchaintest.NewInterchain()

	for _, chain := range chains {
		ic.AddChain(chain)
	}

	rep := testreporter.NewNopReporter()
	eRep := rep.RelayerExecReporter(t)

	wormOsmoPath := "wormosmo"
	ctx := context.Background()
	client, network := interchaintest.DockerSetup(t)

	r := interchaintest.NewBuiltinRelayerFactory(
		ibc.CosmosRly,
		zaptest.NewLogger(t),
		interchaintestrelayer.CustomDockerImage(IBCRelayerImage, IBCRelayerVersion, "100:1000"),
	).Build(t, client, network)

	ic.AddRelayer(r, "relayer")
	ic.AddLink(interchaintest.InterchainLink{
		Chain1:  chains[0], // Wormchain
		Chain2:  chains[1], // osmo
		Relayer: r,
		Path:    wormOsmoPath,
	})

	err = ic.Build(ctx, eRep, interchaintest.InterchainBuildOptions{
		TestName:         t.Name(),
		Client:           client,
		NetworkID:        network,
		SkipPathCreation: true,
	})
	require.NoError(t, err)

	return chains, ic, ctx, r, eRep, client, network
}

// createClient creates a new client on wormchain and osmo
func createClient(
	ctx context.Context,
	r ibc.Relayer,
	eRep *testreporter.RelayerExecReporter,
	wormchain *cosmos.CosmosChain,
	osmo *cosmos.CosmosChain,
	path string,
	trustingPeriod string,
) error {
	// Create path on relayer
	if err := r.GeneratePath(ctx, eRep, wormchain.Config().ChainID, osmo.Config().ChainID, path); err != nil {
		return err
	}

	// Create wormchain client which will use provided trusting period
	if err := r.CreateClient(ctx, eRep, wormchain.Config().ChainID, osmo.Config().ChainID, path, ibc.CreateClientOptions{
		TrustingPeriod: trustingPeriod,
	}); err != nil {
		return err
	}

	// Create osmo client only on first call, it will not expire
	if err := r.CreateClient(ctx, eRep, osmo.Config().ChainID, wormchain.Config().ChainID, path, ibc.CreateClientOptions{
		TrustingPeriod: "24h",
	}); err != nil {
		return err
	}

	if err := testutil.WaitForBlocks(ctx, 1, wormchain, osmo); err != nil {
		return err
	}

	// Create a new connection
	if err := r.CreateConnections(ctx, eRep, path); err != nil {
		return err
	}

	if err := testutil.WaitForBlocks(ctx, 1, wormchain, osmo); err != nil {
		return err
	}

	// Create a new channel & get channels from each chain
	if err := r.CreateChannel(ctx, eRep, path, ibc.DefaultChannelOpts()); err != nil {
		return err
	}

	if err := testutil.WaitForBlocks(ctx, 1, wormchain, osmo); err != nil {
		return err
	}

	return nil
}

// overrideClient creates a new client on wormchain, regardless if a client already exists
func overrideClient(
	ctx context.Context,
	r ibc.Relayer,
	eRep *testreporter.RelayerExecReporter,
	wormchain *cosmos.CosmosChain,
	osmo *cosmos.CosmosChain,
	path string,
	trustingPeriod string,
) error {
	cmd := []string{"rly", "tx", "client", wormchain.Config().ChainID, osmo.Config().ChainID, path,
		"--home", "/home/relayer", "--client-tp", trustingPeriod, "--override"}
	res := r.Exec(ctx, eRep, cmd, []string{})

	return res.Err
}

// isClientExpired checks if a client has expired
func isClientExpired(ctx context.Context, wormchain *cosmos.CosmosChain, clientId string) (bool, error) {
	res, _, err := wormchain.GetNode().ExecQuery(ctx, "ibc", "client", "status", clientId)
	if err != nil {
		return false, err
	}

	return strings.Contains(string(res), "Expired"), nil
}

// waitForClientExpiration waits for a client to expire
func waitForClientExpiration(ctx context.Context, wormchain *cosmos.CosmosChain, clientId string) error {
	maxAttempts := 150
	attempt := 0

	for {
		// If we've tried too many times, return an error
		if attempt >= maxAttempts {
			return fmt.Errorf("client did not expire after %d blocks", maxAttempts)
		}

		// Query wormchain for client status
		expired, err := isClientExpired(ctx, wormchain, clientId)

		// If there was an error, return it
		if err != nil {
			return err
		}

		// If the client has expired, break
		if expired {
			break
		}

		// Wait for a block
		testutil.WaitForBlocks(ctx, 1, wormchain)

		attempt += 1
	}

	return nil
}

// sendIBCTransfer sends an IBC transfer from wormchain to osmo
func sendIBCTransfer(
	t *testing.T,
	ctx context.Context,
	wormchain *cosmos.CosmosChain,
	wormchainUser ibc.Wallet,
	osmo *cosmos.CosmosChain,
	osmoUser ibc.Wallet,
	isFirstTransfer bool,
) {
	// Get user addrs
	wormchainUserAddr := wormchainUser.FormattedAddress()
	osmoUserAddr := osmoUser.FormattedAddress()

	// Define transfer amount
	var transferAmount = math.NewInt(1_000)

	// Get original account balances
	wormchainOrigBal, err := wormchain.GetBalance(ctx, wormchainUserAddr, wormchain.Config().Denom)
	require.NoError(t, err)

	if isFirstTransfer {
		require.Equal(t, GenesisWalletAmount, wormchainOrigBal)
	} else {
		require.Equal(t, GenesisWalletAmount.Sub(transferAmount), wormchainOrigBal)
	}

	osmoOrigBal, err := osmo.GetBalance(ctx, osmoUserAddr, osmo.Config().Denom)
	require.NoError(t, err)
	require.Equal(t, GenesisWalletAmount, osmoOrigBal)

	// Compose an IBC transfer and send from Wormchain -> Osmo
	transfer := ibc.WalletAmount{
		Address: osmoUserAddr,
		Denom:   wormchain.Config().Denom,
		Amount:  transferAmount,
	}

	channelID := "channel-0"
	portID := "transfer"

	wormchainHeight, err := wormchain.Height(ctx)
	require.NoError(t, err)

	transferTx, err := wormchain.SendIBCTransfer(ctx, channelID, wormchainUserAddr, transfer, ibc.TransferOptions{})
	require.NoError(t, err)

	// Poll for the ack to know the transfer was successful
	_, err = testutil.PollForAck(ctx, wormchain, wormchainHeight, wormchainHeight+50, transferTx.Packet)
	require.NoError(t, err)

	err = testutil.WaitForBlocks(ctx, 25, wormchain)
	require.NoError(t, err)

	// Get the IBC denom for uworm on osmo
	wormchainTokenDenom := transfertypes.GetPrefixedDenom(portID, channelID, wormchain.Config().Denom)
	wormchainIBCDenom := transfertypes.ParseDenomTrace(wormchainTokenDenom).IBCDenom()

	// Assert that the funds are no longer present in user acc on wormchain and are in the user acc on osmo
	wormchainUpdateBal, err := wormchain.GetBalance(ctx, wormchainUserAddr, wormchain.Config().Denom)
	require.NoError(t, err)
	require.Equal(t, wormchainOrigBal.Sub(transferAmount), wormchainUpdateBal)

	osmoUpdateBal, err := osmo.GetBalance(ctx, osmoUserAddr, wormchainIBCDenom)
	require.NoError(t, err)

	if isFirstTransfer {
		require.Equal(t, transferAmount, osmoUpdateBal)
	} else {
		require.Equal(t, transferAmount.Add(transferAmount), osmoUpdateBal)
	}
}

// TestIBCClientUpdateVAA tests the governance VAA can restore expired ibc clients
func TestIBCClientUpdateVaa(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	t.Parallel()

	// base setup
	guardians := guardians.CreateValSet(t, 1)
	chains, ic, ctx, r, eRep, _, _ := buildIC(t, *guardians)
	require.NotNil(t, ic)
	require.NotNil(t, ctx)

	err := r.StopRelayer(ctx, eRep)
	require.NoError(t, err)

	wormchain := chains[0].(*cosmos.CosmosChain)
	osmo := chains[1].(*cosmos.CosmosChain)

	users := interchaintest.GetAndFundTestUsers(t, ctx, t.Name(), GenesisWalletAmount, wormchain, osmo)
	wormchainUser := users[0]
	osmoUser := users[1]

	_, err = wormchain.GetNode().ExecTx(ctx, "validator", "wormhole", "create-allowed-address", wormchainUser.FormattedAddress(), "wormchain-user")
	require.NoError(t, err)

	// ----------------------------------------------
	// Create initial client & send IBC transfer
	// ----------------------------------------------

	// Create first client between wormchain and osmo
	err = createClient(ctx, r, eRep, wormchain, osmo, "wormosmo", "2m")
	require.NoError(t, err)

	// Start the relayer
	err = r.StartRelayer(ctx, eRep)
	require.NoError(t, err)

	t.Cleanup(func() {
		r.StopRelayer(ctx, eRep)
	})

	// Send an IBC transfer from wormchain to osmo
	sendIBCTransfer(t, ctx, wormchain, wormchainUser, osmo, osmoUser, true)

	// ----------------------------------------------
	// Wait for client to expire & create new client
	// ----------------------------------------------

	// Stop relayer so client expires
	err = r.StopRelayer(ctx, eRep)
	require.NoError(t, err)

	// Wait for first clients to expire
	err = waitForClientExpiration(ctx, wormchain, "07-tendermint-0")
	require.NoError(t, err)

	// Create 2nd client between wormchain and osmo
	err = overrideClient(ctx, r, eRep, wormchain, osmo, "wormosmo", "24h")
	require.NoError(t, err)

	// Tell relayer to re-fetch src client
	srcClient := "07-tendermint-0"
	err = r.UpdatePath(ctx, eRep, "wormosmo", ibc.PathUpdateOptions{
		SrcClientID: &srcClient,
	})
	require.NoError(t, err)

	// ----------------------------------------------
	// Update to new client with VAA Payload
	// ----------------------------------------------

	// create a governance VAA to update the expired client
	payloadBytes, err := createIbcClientUpdateVaa("07-tendermint-0", "07-tendermint-1")
	require.NoError(t, err)

	// create and send
	err = createAndExecuteVaa(ctx, guardians, wormchain, payloadBytes)
	require.NoError(t, err)

	// wait 1 block
	err = testutil.WaitForBlocks(ctx, 1, wormchain)
	require.NoError(t, err)

	// ensure old client is now active again
	expired, err := isClientExpired(ctx, wormchain, "07-tendermint-0")
	require.NoError(t, err)
	require.False(t, expired, "Client 07-tendermint-0 is still expired")

	// ----------------------------------------------
	// Send funds again on ORIGINAL channel - pass
	// ----------------------------------------------

	err = r.StartRelayer(ctx, eRep)
	require.NoError(t, err)

	// Send an IBC transfer from wormchain to osmo
	sendIBCTransfer(t, ctx, wormchain, wormchainUser, osmo, osmoUser, false)
}
