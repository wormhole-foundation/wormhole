// broadcastvaa is a one-off recovery tool. It takes a single, already-signed VAA
// as a hex string, verifies it against the on-chain guardian set that it
// references, and (when --broadcast is set) broadcasts it over the gossip
// network as a `SignedVAAWithQuorum`.
//
// This is a hack tool and should not be imported by any production tooling.
//
// Example:
//
//	go run . \
//	  --vaa 01000000... \
//	  --ethRPC https://ethereum-rpc.publicnode.com \
//	  --broadcast
package main

import (
	"context"
	"encoding/hex"
	"flag"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/certusone/wormhole/node/pkg/p2p"
	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
	"github.com/certusone/wormhole/node/pkg/supervisor"
	"github.com/certusone/wormhole/node/pkg/watchers/evm"

	ethAbi "github.com/certusone/wormhole/node/pkg/watchers/evm/connectors/ethabi"
	ethBind "github.com/ethereum/go-ethereum/accounts/abi/bind"
	ethCommon "github.com/ethereum/go-ethereum/common"
	ethClient "github.com/ethereum/go-ethereum/ethclient"
	ethRpc "github.com/ethereum/go-ethereum/rpc"

	"github.com/wormhole-foundation/wormhole/sdk/vaa"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

const (
	// defaultP2PPort is the default UDP port the broadcast node listens on.
	defaultP2PPort = 8997
	// defaultBroadcastWait is how long to stay connected after broadcasting, to allow propagation.
	defaultBroadcastWait = 15 * time.Second
	// rpcTimeout bounds Ethereum RPC calls.
	rpcTimeout = 15 * time.Second
)

var (
	vaaHex        = flag.String("vaa", "", "The signed VAA to broadcast, as a hex string (with or without a 0x prefix)")
	ethRPC        = flag.String("ethRPC", "", "Ethereum RPC endpoint used to read the guardian set (defaults to the Ethereum public RPC from the evm chain config for the env)")
	coreContract  = flag.String("coreContract", "", "Core bridge contract address (defaults to the Ethereum core contract from the evm chain config for the env)")
	envStr        = flag.String("env", "mainnet", "Environment: mainnet or testnet")
	wormholescan  = flag.String("wormholescan", "https://api.wormholescan.io", "Wormholescan API base URL (used only to log the signed VAA link)")
	nodeKeyPath   = flag.String("nodeKey", "/tmp/broadcastvaa_node.key", "Path to the libp2p node key (created if missing)")
	p2pPort       = flag.Uint("port", defaultP2PPort, "P2P UDP port to listen on")
	p2pNetworkID  = flag.String("network", "", "P2P network identifier (overrides the env default)")
	p2pBootstrap  = flag.String("bootstrap", "", "P2P bootstrap peers (overrides the env default)")
	broadcastWait = flag.Duration("broadcastWait", defaultBroadcastWait, "How long to stay connected after broadcasting, to allow propagation")
	doBroadcast   = flag.Bool("broadcast", false, "Broadcast the verified VAA over gossip. If not set, the tool only parses and verifies it.")
)

func main() {
	flag.Parse()

	if *vaaHex == "" {
		log.Fatal("--vaa is required")
	}

	env, err := common.ParseEnvironment(*envStr)
	if err != nil || (env != common.MainNet && env != common.TestNet) {
		log.Fatalf("--env must be mainnet or testnet, got %q", *envStr)
	}

	logger, err := zap.NewDevelopment()
	if err != nil {
		log.Fatalf("failed to create logger: %v", err)
	}

	ctx := context.Background()

	// Parse the VAA from hex.
	vaaBytes, err := hex.DecodeString(strings.TrimPrefix(strings.TrimSpace(*vaaHex), "0x"))
	if err != nil {
		log.Fatalf("failed to hex-decode --vaa: %v", err)
	}
	v, err := vaa.Unmarshal(vaaBytes)
	if err != nil {
		log.Fatalf("failed to parse VAA: %v", err)
	}
	logger.Info("parsed VAA",
		zap.String("messageID", v.MessageID()),
		zap.Uint32("guardianSetIndex", v.GuardianSetIndex),
		zap.Int("numSignatures", len(v.Signatures)),
		zap.String("digest", v.SigningDigest().Hex()),
	)

	// Pull the Ethereum core contract address and public RPC from the evm chain config,
	// using them as defaults unless overridden on the command line.
	chainCfg, err := evm.GetChainConfigMap(env)
	if err != nil {
		log.Fatalf("failed to load evm chain config for env %s: %v", env, err)
	}
	ethCfg, ok := chainCfg[vaa.ChainIDEthereum]
	if !ok {
		log.Fatalf("no Ethereum entry in evm chain config for env %s", env)
	}

	coreAddr := *coreContract
	if coreAddr == "" {
		coreAddr = ethCfg.ContractAddr
	}
	rpcURL := *ethRPC
	if rpcURL == "" {
		rpcURL = ethCfg.PublicRPC
	}
	logger.Info("using Ethereum core contract and RPC",
		zap.String("coreContract", coreAddr),
		zap.String("ethRPC", rpcURL),
	)

	// Read the guardian set that the VAA references from the core contract, and verify
	// the VAA against it.
	gs, err := fetchGuardianSet(ctx, rpcURL, coreAddr, v.GuardianSetIndex)
	if err != nil {
		log.Fatalf("failed to fetch guardian set %d: %v", v.GuardianSetIndex, err)
	}
	logger.Info("fetched guardian set",
		zap.Uint32("index", gs.Index),
		zap.Int("numKeys", len(gs.Keys)),
		zap.Int("quorum", gs.Quorum()),
	)

	if err := v.Verify(gs.Keys); err != nil {
		log.Fatalf("VAA failed verification against guardian set %d: %v", gs.Index, err)
	}
	logger.Info("VAA verified",
		zap.String("messageID", v.MessageID()),
		zap.Int("numSignatures", len(v.Signatures)),
	)

	if !*doBroadcast {
		logger.Info("not broadcasting. Re-run with --broadcast to broadcast over gossip.")
		return
	}

	if err := broadcast(ctx, logger, env, gs, v); err != nil {
		log.Fatalf("failed to broadcast VAA: %v", err)
	}
}

// fetchGuardianSet reads the keys for the given guardian set index from the core contract.
func fetchGuardianSet(ctx context.Context, rpcURL, coreAddr string, index uint32) (*common.GuardianSet, error) {
	timeout, cancel := context.WithTimeout(ctx, rpcTimeout)
	defer cancel()

	rawClient, err := ethRpc.DialContext(timeout, rpcURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to ethereum: %w", err)
	}
	client := ethClient.NewClient(rawClient)
	caller, err := ethAbi.NewAbiCaller(ethCommon.HexToAddress(coreAddr), client)
	if err != nil {
		return nil, fmt.Errorf("failed to create core contract caller: %w", err)
	}

	result, err := caller.GetGuardianSet(&ethBind.CallOpts{Context: timeout}, index)
	if err != nil {
		return nil, fmt.Errorf("failed to read guardian set %d: %w", index, err)
	}

	return common.NewGuardianSet(result.Keys, index), nil
}

// broadcast starts a p2p node and publishes the verified VAA as a SignedVAAWithQuorum.
func broadcast(
	ctx context.Context,
	logger *zap.Logger,
	env common.Environment,
	gs *common.GuardianSet,
	v *vaa.VAA,
) error {
	networkID := *p2pNetworkID
	if networkID == "" {
		networkID = p2p.GetNetworkId(env)
	}
	bootstrap := *p2pBootstrap
	if bootstrap == "" {
		var err error
		bootstrap, err = p2p.GetBootstrapPeers(env)
		if err != nil {
			return fmt.Errorf("failed to determine bootstrap peers: %w", err)
		}
	}

	priv, err := common.GetOrCreateNodeKey(logger, *nodeKeyPath)
	if err != nil {
		return fmt.Errorf("failed to load node key: %w", err)
	}

	// Make the guardian set available to the p2p layer.
	gst := common.NewGuardianSetState(nil)
	gst.Set(gs)

	marshaled, err := v.Marshal()
	if err != nil {
		return fmt.Errorf("failed to marshal VAA %s: %w", v.MessageID(), err)
	}
	w := gossipv1.GossipMessage{Message: &gossipv1.GossipMessage_SignedVaaWithQuorum{
		SignedVaaWithQuorum: &gossipv1.SignedVAAWithQuorum{Vaa: marshaled},
	}}
	msg, err := proto.Marshal(&w)
	if err != nil {
		return fmt.Errorf("failed to marshal gossip message for %s: %w", v.MessageID(), err)
	}

	// Buffer the channel so the VAA can be queued without blocking.
	gossipVaaSendC := make(chan []byte, 1)
	gossipVaaSendC <- msg

	// Log the Wormholescan signed VAA link so it can be polled to confirm the broadcast was picked up.
	logger.Info("broadcasting VAA",
		zap.String("messageID", v.MessageID()),
		zap.String("wormholescan", fmt.Sprintf("%s/v1/signed_vaa/%d/%s/%d",
			*wormholescan, v.EmitterChain, v.EmitterAddress, v.Sequence)),
	)

	rootCtx, rootCtxCancel := context.WithCancel(ctx)
	defer rootCtxCancel()

	logger.Info("starting p2p node to broadcast VAA",
		zap.String("networkID", networkID),
		zap.Duration("broadcastWait", *broadcastWait),
	)

	supervisor.New(rootCtx, logger, func(supCtx context.Context) error {
		components := p2p.DefaultComponents()
		components.Port = *p2pPort

		params, err := p2p.NewRunParams(
			bootstrap,
			networkID,
			priv,
			gst,
			rootCtxCancel,
			p2p.WithVAASender(gossipVaaSendC),
			p2p.WithComponents(components),
		)
		if err != nil {
			return err
		}

		if err := supervisor.Run(supCtx, "p2p", p2p.Run(params)); err != nil {
			return err
		}

		logger.Info("p2p started; broadcasting and waiting for propagation")

		// The send loop drains gossipVaaSendC as soon as the topic is joined. Wait for
		// the configured duration to give the message time to propagate, then shut down.
		select {
		case <-supCtx.Done():
		case <-time.After(*broadcastWait):
		}
		rootCtxCancel()
		return nil
	}, supervisor.WithPropagatePanic)

	<-rootCtx.Done()
	logger.Info("done broadcasting")
	return nil
}
