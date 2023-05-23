// This tool can be used to send various queries to the p2p gossip network.
// It is meant for testing purposes only.

package main

import (
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/certusone/wormhole/node/pkg/p2p"
	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
	nodev1 "github.com/certusone/wormhole/node/pkg/proto/node/v1"
	ethCommon "github.com/ethereum/go-ethereum/common"
	ethCrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/libp2p/go-libp2p"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/libp2p/go-libp2p/core/routing"
	libp2ptls "github.com/libp2p/go-libp2p/p2p/security/tls"
	libp2pquic "github.com/libp2p/go-libp2p/p2p/transport/quic"
	"github.com/multiformats/go-multiaddr"
	"go.uber.org/zap"
	"golang.org/x/crypto/openpgp/armor" //nolint
	"google.golang.org/protobuf/proto"
)

var queryRequestPrefix = []byte("query_request_00000000000000000000|")

func queryRequestDigest(b []byte) ethCommon.Hash {
	return ethCrypto.Keccak256Hash(append(queryRequestPrefix, b...))
}

// this script has to be run inside kubernetes since it relies on UDP
// https://github.com/kubernetes/kubernetes/issues/47862
// kubectl --namespace=wormhole exec -it spy-0 -- sh -c "cd node/hack/query/ && go run send_req.go"
// one way to iterate inside the container
// kubectl --namespace=wormhole exec -it spy-0 -- bash
// apt update
// apt install nano
// cd node/hack/query
// echo "" > send_req.go
// nano send_req.go
// [paste, ^x, y, enter]
// go run send_req.go

func main() {

	//
	// BEGIN SETUP
	//

	p2pNetworkID := "/wormhole/dev"
	var p2pPort uint = 8998 // don't collide with spy so we can run from the same container in tilt
	p2pBootstrap := "/dns4/guardian-0.guardian/udp/8999/quic/p2p/12D3KooWL3XJ9EMCyZvmmGXL2LMiVBtrVa2BuESsJiXkSj7333Jw"
	nodeKeyPath := "/tmp/querier.key" // don't use node key so we get a new address

	ctx := context.Background()
	logger, _ := zap.NewDevelopment()

	signingKeyPath := string("./dev.guardian.key")

	logger.Info("Loading signing key", zap.String("signingKeyPath", signingKeyPath))
	sk, err := loadGuardianKey(signingKeyPath)
	if err != nil {
		logger.Fatal("failed to load guardian key", zap.Error(err))
	}
	logger.Info("Signing key loaded", zap.String("publicKey", ethCrypto.PubkeyToAddress(sk.PublicKey).Hex()))

	// Load p2p private key
	var priv crypto.PrivKey
	priv, err = common.GetOrCreateNodeKey(logger, nodeKeyPath)
	if err != nil {
		logger.Fatal("Failed to load node key", zap.Error(err))
	}

	// Manual p2p setup
	components := p2p.DefaultComponents()
	components.Port = p2pPort
	bootstrapPeers := p2pBootstrap
	networkID := p2pNetworkID
	h, err := libp2p.New(
		// Use the keypair we generated
		libp2p.Identity(priv),

		// Multiple listen addresses
		libp2p.ListenAddrStrings(
			components.ListeningAddresses()...,
		),

		// Enable TLS security as the only security protocol.
		libp2p.Security(libp2ptls.ID, libp2ptls.New),

		// Enable QUIC transport as the only transport.
		libp2p.Transport(libp2pquic.NewTransport),

		// Let's prevent our peer from having too many
		// connections by attaching a connection manager.
		libp2p.ConnectionManager(components.ConnMgr),

		// Let this host use the DHT to find other hosts
		libp2p.Routing(func(h host.Host) (routing.PeerRouting, error) {
			logger.Info("Connecting to bootstrap peers", zap.String("bootstrap_peers", bootstrapPeers))
			bootstrappers := make([]peer.AddrInfo, 0)
			for _, addr := range strings.Split(bootstrapPeers, ",") {
				if addr == "" {
					continue
				}
				ma, err := multiaddr.NewMultiaddr(addr)
				if err != nil {
					logger.Error("Invalid bootstrap address", zap.String("peer", addr), zap.Error(err))
					continue
				}
				pi, err := peer.AddrInfoFromP2pAddr(ma)
				if err != nil {
					logger.Error("Invalid bootstrap address", zap.String("peer", addr), zap.Error(err))
					continue
				}
				if pi.ID == h.ID() {
					logger.Info("We're a bootstrap node")
					continue
				}
				bootstrappers = append(bootstrappers, *pi)
			}
			// TODO(leo): Persistent data store (i.e. address book)
			idht, err := dht.New(ctx, h, dht.Mode(dht.ModeServer),
				// This intentionally makes us incompatible with the global IPFS DHT
				dht.ProtocolPrefix(protocol.ID("/"+networkID)),
				dht.BootstrapPeers(bootstrappers...),
			)
			return idht, err
		}),
	)

	if err != nil {
		panic(err)
	}

	topic := fmt.Sprintf("%s/%s", networkID, "broadcast")

	logger.Info("Subscribing pubsub topic", zap.String("topic", topic))
	ps, err := pubsub.NewGossipSub(ctx, h)
	if err != nil {
		panic(err)
	}

	th, err := ps.Join(topic)
	if err != nil {
		logger.Panic("failed to join topic", zap.Error(err))
	}

	sub, err := th.Subscribe()
	if err != nil {
		logger.Panic("failed to subscribe topic", zap.Error(err))
	}

	logger.Info("Node has been started", zap.String("peer_id", h.ID().String()),
		zap.String("addrs", fmt.Sprintf("%v", h.Addrs())))

	// Wait for peers
	for len(th.ListPeers()) < 1 {
		time.Sleep(time.Millisecond * 100)
	}

	//
	// END SETUP
	//

	to, _ := hex.DecodeString("0d500b1d8e8ef31e21c99d1db9a6444d3adf1270")
	data, _ := hex.DecodeString("18160ddd")
	// block := "0x28d9630"
	block := "latest"
	// block := "0x9999bac44d09a7f69ee7941819b0a19c59ccb1969640cc513be09ef95ed2d8e2"
	callRequest := &gossipv1.EthCallQueryRequest{
		To:    to,
		Data:  data,
		Block: block,
	}
	queryRequest := &gossipv1.QueryRequest{
		ChainId: 5,
		Nonce:   0,
		Message: &gossipv1.QueryRequest_EthCallQueryRequest{
			EthCallQueryRequest: callRequest}}

	queryRequestBytes, err := proto.Marshal(queryRequest)
	if err != nil {
		panic(err)
	}

	// Sign the query request using our private key.
	digest := queryRequestDigest(queryRequestBytes)
	sig, err := ethCrypto.Sign(digest.Bytes(), sk)
	if err != nil {
		panic(err)
	}

	signedQueryRequest := &gossipv1.SignedQueryRequest{
		QueryRequest: queryRequestBytes,
		Signature:    sig,
	}

	msg := gossipv1.GossipMessage{
		Message: &gossipv1.GossipMessage_SignedQueryRequest{
			SignedQueryRequest: signedQueryRequest,
		},
	}

	b, err := proto.Marshal(&msg)
	if err != nil {
		panic(err)
	}

	err = th.Publish(ctx, b)
	if err != nil {
		panic(err)
	}

	logger.Info("Waiting for message...")
	// TODO: max wait time
	for {
		envelope, err := sub.Next(ctx)
		if err != nil {
			logger.Panic("failed to receive pubsub message", zap.Error(err))
		}
		var msg gossipv1.GossipMessage
		err = proto.Unmarshal(envelope.Data, &msg)
		if err != nil {
			logger.Info("received invalid message",
				zap.Binary("data", envelope.Data),
				zap.String("from", envelope.GetFrom().String()))
			continue
		}
		var isMatchingResponse bool
		switch m := msg.Message.(type) {
		case *gossipv1.GossipMessage_SignedQueryResponse:
			// TODO: check if it's matching
			logger.Info("response received", zap.Any("response", m.SignedQueryResponse))
			isMatchingResponse = true
		default:
			continue
		}
		if isMatchingResponse {
			break
		}
	}

	//
	// BEGIN SHUTDOWN
	//

	// Cleanly shutdown
	// Without this the same host won't properly discover peers until some timeout
	sub.Cancel()
	if err := th.Close(); err != nil {
		logger.Fatal("Error closing the topic", zap.Error(err))
	}
	if err := h.Close(); err != nil {
		logger.Fatal("Error closing the host", zap.Error(err))
	}

	//
	// END SHUTDOWN
	//

	logger.Info("Success! All tests passed!")
}

const (
	GuardianKeyArmoredBlock = "WORMHOLE GUARDIAN PRIVATE KEY"
)

// loadGuardianKey loads a serialized guardian key from disk.
func loadGuardianKey(filename string) (*ecdsa.PrivateKey, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}

	p, err := armor.Decode(f)
	if err != nil {
		return nil, fmt.Errorf("failed to read armored file: %w", err)
	}

	if p.Type != GuardianKeyArmoredBlock {
		return nil, fmt.Errorf("invalid block type: %s", p.Type)
	}

	b, err := io.ReadAll(p.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var m nodev1.GuardianKey
	err = proto.Unmarshal(b, &m)
	if err != nil {
		return nil, fmt.Errorf("failed to deserialize protobuf: %w", err)
	}

	gk, err := ethCrypto.ToECDSA(m.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to deserialize raw key data: %w", err)
	}

	return gk, nil
}
