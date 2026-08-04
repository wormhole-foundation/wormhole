// recovervaa is a one-off recovery tool. It reads a governance prototxt file
// (the same `InjectGovernanceVAARequest` format consumed by `guardiand admin
// governance-vaa-inject`), reconstructs the unsigned VAA body for every message,
// collects the guardian signatures for that VAA from Wormholescan's observations
// API, verifies the assembled VAA against the latest on-chain guardian set, and
// (when --broadcast is set) broadcasts each verified VAA over the gossip
// network as a `SignedVAAWithQuorum`.
//
// This is a hack tool and should not be imported by any production tooling.
//
// Example:
//
//	go run . \
//	  --prototxt ./vaas.prototxt \
//	  --ethRPC https://ethereum-rpc.publicnode.com \
//	  --broadcast
package main

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"sort"
	"time"

	"github.com/certusone/wormhole/node/pkg/adminrpc"
	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/certusone/wormhole/node/pkg/p2p"
	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
	nodev1 "github.com/certusone/wormhole/node/pkg/proto/node/v1"
	"github.com/certusone/wormhole/node/pkg/supervisor"
	"github.com/certusone/wormhole/node/pkg/watchers/evm"

	ethAbi "github.com/certusone/wormhole/node/pkg/watchers/evm/connectors/ethabi"
	ethBind "github.com/ethereum/go-ethereum/accounts/abi/bind"
	ethCommon "github.com/ethereum/go-ethereum/common"
	ethClient "github.com/ethereum/go-ethereum/ethclient"
	ethRpc "github.com/ethereum/go-ethereum/rpc"

	"github.com/wormhole-foundation/wormhole/sdk/vaa"
	"go.uber.org/zap"
	"google.golang.org/protobuf/encoding/prototext"
	"google.golang.org/protobuf/proto"
)

const (
	// defaultP2PPort is the default UDP port the broadcast node listens on.
	defaultP2PPort = 8997
	// defaultBroadcastWait is how long to stay connected after broadcasting, to allow propagation.
	defaultBroadcastWait = 15 * time.Second
	// rpcTimeout bounds Ethereum RPC and Wormholescan HTTP calls.
	rpcTimeout = 15 * time.Second
	// signatureLength is the length in bytes of a guardian ECDSA signature (R || S || V).
	signatureLength = 65
)

var (
	prototxtPath  = flag.String("prototxt", "", "Path to the governance prototxt file (InjectGovernanceVAARequest format)")
	ethRPC        = flag.String("ethRPC", "", "Ethereum RPC endpoint used to read the latest guardian set (defaults to the Ethereum public RPC from the evm chain config for the env)")
	coreContract  = flag.String("coreContract", "", "Core bridge contract address (defaults to the Ethereum core contract from the evm chain config for the env)")
	envStr        = flag.String("env", "mainnet", "Environment: mainnet or testnet")
	wormholescan  = flag.String("wormholescan", "https://api.wormholescan.io", "Wormholescan API base URL")
	nodeKeyPath   = flag.String("nodeKey", "/tmp/recovervaa_node.key", "Path to the libp2p node key (created if missing)")
	p2pPort       = flag.Uint("port", defaultP2PPort, "P2P UDP port to listen on")
	p2pNetworkID  = flag.String("network", "", "P2P network identifier (overrides the env default)")
	p2pBootstrap  = flag.String("bootstrap", "", "P2P bootstrap peers (overrides the env default)")
	broadcastWait = flag.Duration("broadcastWait", defaultBroadcastWait, "How long to stay connected after broadcasting, to allow propagation")
	doBroadcast   = flag.Bool("broadcast", false, "Broadcast the verified VAAs over gossip. If not set, the tool only reconstructs and verifies them.")
)

// wormholescanObservation models a single entry from the Wormholescan observations endpoint:
// GET /api/v1/observations/{chain}/{emitter}/{sequence}
type wormholescanObservation struct {
	// Hash is the base64-encoded VAA digest that was signed.
	Hash string `json:"hash"`
	// GuardianAddr is the hex (0x-prefixed) truncated eth address of the signing guardian.
	GuardianAddr string `json:"guardianAddr"`
	// Signature is the base64-encoded 65-byte ECDSA signature (R||S||V with V in {0,1}).
	Signature string `json:"signature"`
}

func main() {
	flag.Parse()

	if *prototxtPath == "" {
		log.Fatal("--prototxt is required")
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

	// Read the latest guardian set from the core contract.
	gs, err := fetchLatestGuardianSet(ctx, rpcURL, coreAddr)
	if err != nil {
		log.Fatalf("failed to fetch latest guardian set: %v", err)
	}
	logger.Info("fetched latest guardian set",
		zap.Uint32("index", gs.Index),
		zap.Int("numKeys", len(gs.Keys)),
		zap.Int("quorum", gs.Quorum()),
	)

	// Parse the governance prototxt file.
	b, err := os.ReadFile(*prototxtPath)
	if err != nil {
		log.Fatalf("failed to read prototxt file: %v", err)
	}
	var req nodev1.InjectGovernanceVAARequest
	if err := prototext.Unmarshal(b, &req); err != nil {
		log.Fatalf("failed to parse prototxt file: %v", err)
	}
	if len(req.Messages) == 0 {
		log.Fatal("no messages found in prototxt file")
	}
	timestamp := time.Unix(int64(req.Timestamp), 0)
	logger.Info("parsed prototxt",
		zap.Int("numMessages", len(req.Messages)),
		zap.Uint32("currentSetIndex", req.CurrentSetIndex),
		zap.Time("timestamp", timestamp),
	)

	httpClient := &http.Client{Timeout: rpcTimeout}

	// Reconstruct, sign, and verify each VAA.
	var verified []*vaa.VAA
	for i, message := range req.Messages {
		v, err := adminrpc.GovMsgToVaa(message, req.CurrentSetIndex, timestamp)
		if err != nil {
			logger.Error("failed to build VAA body from message; skipping",
				zap.Int("index", i), zap.Error(err))
			continue
		}

		// The guardian set index is not part of the digest, but the broadcast VAA
		// header must reference the set that actually produced the signatures.
		v.GuardianSetIndex = gs.Index

		digest := v.SigningDigest()
		logger.Info("reconstructed VAA body",
			zap.Int("index", i),
			zap.String("messageID", v.MessageID()),
			zap.String("digest", digest.Hex()),
		)

		sigs, err := fetchSignatures(ctx, httpClient, *wormholescan, v, gs, digest, logger)
		if err != nil {
			logger.Error("failed to fetch signatures; skipping",
				zap.String("messageID", v.MessageID()), zap.Error(err))
			continue
		}

		// VAA signatures must be sorted by guardian index.
		sort.Slice(sigs, func(a, b int) bool { return sigs[a].Index < sigs[b].Index })
		v.Signatures = sigs

		if err = v.Verify(gs.Keys); err != nil {
			logger.Error("assembled VAA failed verification against the latest guardian set; skipping",
				zap.String("messageID", v.MessageID()),
				zap.Int("numSignatures", len(sigs)),
				zap.Error(err),
			)
			continue
		}

		marshaled, err := v.Marshal()
		if err != nil {
			logger.Error("failed to marshal verified VAA; skipping",
				zap.String("messageID", v.MessageID()), zap.Error(err))
			continue
		}
		logger.Info("VAA verified",
			zap.String("messageID", v.MessageID()),
			zap.Int("numSignatures", len(sigs)),
			zap.String("vaa", hex.EncodeToString(marshaled)),
		)
		verified = append(verified, v)
	}

	logger.Info("reconstruction complete",
		zap.Int("verified", len(verified)),
		zap.Int("total", len(req.Messages)),
	)

	if len(verified) == 0 {
		log.Fatal("no VAAs could be verified; nothing to broadcast")
	}

	if !*doBroadcast {
		logger.Info("not broadcasting. Re-run with --broadcast to broadcast over gossip.")
		return
	}

	if err := broadcast(ctx, logger, env, gs, verified); err != nil {
		log.Fatalf("failed to broadcast VAAs: %v", err)
	}
}

// fetchLatestGuardianSet reads the current guardian set index and keys from the core contract.
func fetchLatestGuardianSet(ctx context.Context, rpcURL, coreAddr string) (*common.GuardianSet, error) {
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

	gsIndex, err := caller.GetCurrentGuardianSetIndex(&ethBind.CallOpts{Context: timeout})
	if err != nil {
		return nil, fmt.Errorf("failed to read current guardian set index: %w", err)
	}
	result, err := caller.GetGuardianSet(&ethBind.CallOpts{Context: timeout}, gsIndex)
	if err != nil {
		return nil, fmt.Errorf("failed to read guardian set %d: %w", gsIndex, err)
	}

	return common.NewGuardianSet(result.Keys, gsIndex), nil
}

// fetchSignatures pulls the guardian observations for the given VAA from Wormholescan and converts
// them into VAA signatures keyed by their index in the provided guardian set. Observations from
// guardians that are not in the current set, or whose signed digest does not match the reconstructed
// VAA body, are skipped.
func fetchSignatures(
	ctx context.Context,
	client *http.Client,
	baseURL string,
	v *vaa.VAA,
	gs *common.GuardianSet,
	digest ethCommon.Hash,
	logger *zap.Logger,
) ([]*vaa.Signature, error) {
	// Wormholescan expects the emitter address as a 64-character hex string (no 0x prefix).
	url := fmt.Sprintf("%s/api/v1/observations/%d/%s/%d",
		baseURL, v.EmitterChain, v.EmitterAddress, v.Sequence)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to build request: %w", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := common.SafeRead(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}

	var observations []wormholescanObservation
	if err := json.Unmarshal(body, &observations); err != nil {
		return nil, fmt.Errorf("failed to decode observations: %w", err)
	}
	if len(observations) == 0 {
		return nil, fmt.Errorf("no observations found")
	}

	expectedHash := base64.StdEncoding.EncodeToString(digest.Bytes())

	seen := make(map[uint8]bool)
	var sigs []*vaa.Signature
	for _, obs := range observations {
		// Ensure the signature was made over the body we reconstructed. If the hash differs,
		// the reconstructed body (most likely the timestamp) does not match what was signed.
		if obs.Hash != expectedHash {
			logger.Warn("observation hash does not match reconstructed digest; skipping observation",
				zap.String("messageID", v.MessageID()),
				zap.String("guardian", obs.GuardianAddr),
				zap.String("observationHash", obs.Hash),
				zap.String("expectedHash", expectedHash),
			)
			continue
		}

		addr := ethCommon.HexToAddress(obs.GuardianAddr)
		idx, found := gs.KeyIndex(addr)
		if !found {
			logger.Warn("observation guardian not in current set; skipping",
				zap.String("guardian", obs.GuardianAddr))
			continue
		}

		sigBytes, err := base64.StdEncoding.DecodeString(obs.Signature)
		if err != nil {
			logger.Warn("failed to decode signature; skipping observation",
				zap.String("guardian", obs.GuardianAddr), zap.Error(err))
			continue
		}
		if len(sigBytes) != signatureLength {
			logger.Warn("unexpected signature length; skipping observation",
				zap.String("guardian", obs.GuardianAddr), zap.Int("len", len(sigBytes)))
			continue
		}

		index := uint8(idx) // #nosec G115 -- guardian set is far smaller than 256
		if seen[index] {
			continue
		}
		seen[index] = true

		var sigData vaa.SignatureData
		copy(sigData[:], sigBytes)
		sigs = append(sigs, &vaa.Signature{Index: index, Signature: sigData})
	}

	if len(sigs) < gs.Quorum() {
		return nil, fmt.Errorf("insufficient signatures: got %d, need %d for quorum", len(sigs), gs.Quorum())
	}
	return sigs, nil
}

// broadcast starts a p2p node and publishes each verified VAA as a SignedVAAWithQuorum.
func broadcast(
	ctx context.Context,
	logger *zap.Logger,
	env common.Environment,
	gs *common.GuardianSet,
	vaas []*vaa.VAA,
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

	// Buffer the channel so all VAAs can be queued without blocking.
	gossipVaaSendC := make(chan []byte, len(vaas))
	for _, v := range vaas {
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
		gossipVaaSendC <- msg

		// Log the Wormholescan signed VAA link so it can be polled to confirm the broadcast was picked up.
		logger.Info("broadcasting VAA",
			zap.String("messageID", v.MessageID()),
			zap.String("wormholescan", fmt.Sprintf("%s/v1/signed_vaa/%d/%s/%d",
				*wormholescan, v.EmitterChain, v.EmitterAddress, v.Sequence)),
		)
	}

	rootCtx, rootCtxCancel := context.WithCancel(ctx)
	defer rootCtxCancel()

	logger.Info("starting p2p node to broadcast VAAs",
		zap.String("networkID", networkID),
		zap.Int("numVAAs", len(vaas)),
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
		// the configured duration to give the messages time to propagate, then shut down.
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
