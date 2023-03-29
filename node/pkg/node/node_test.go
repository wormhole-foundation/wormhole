package node

import (
	"context"
	"crypto/ecdsa"
	"crypto/rand"
	"fmt"
	"testing"
	"time"

	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/certusone/wormhole/node/pkg/db"
	"github.com/certusone/wormhole/node/pkg/devnet"
	"github.com/certusone/wormhole/node/pkg/readiness"
	"github.com/certusone/wormhole/node/pkg/supervisor"
	"github.com/certusone/wormhole/node/pkg/watchers"
	"github.com/certusone/wormhole/node/pkg/watchers/mock"
	eth_crypto "github.com/ethereum/go-ethereum/crypto"
	libp2p_crypto "github.com/libp2p/go-libp2p/core/crypto"
	libp2p_peer "github.com/libp2p/go-libp2p/core/peer"
	"github.com/test-go/testify/assert"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
	"go.uber.org/zap"
)

type mockGuardianSet struct {
	p2pKeys []libp2p_crypto.PrivKey
}

func newMockGuardianSet(n int) *mockGuardianSet {
	p2pKeys := make([]libp2p_crypto.PrivKey, n)
	for i := 0; i < n; i++ {
		p2pKeys[i] = devnet.DeterministicP2PPrivKeyByIndex(int64(i))
	}

	return &mockGuardianSet{p2pKeys}
}

func (gs *mockGuardianSet) addMockGuardian(ctx context.Context, mockGuardianIndex uint) error {
	// Node's main lifecycle context.
	rootCtx, rootCtxCancel := context.WithCancel(ctx)
	defer rootCtxCancel()
	logger := supervisor.Logger(ctx)

	// setup db
	dataDir := fmt.Sprintf("/tmp/testguardian_%d", mockGuardianIndex)
	db := db.OpenDb(logger, &dataDir)

	// generate guardian key
	gk, err := ecdsa.GenerateKey(eth_crypto.S256(), rand.Reader)
	if err != nil {
		return err
	}

	// set environment
	env := common.GoTest

	// setup a mock watcher
	var watcherConfigs = []watchers.WatcherConfig{
		&mock.WatcherConfig{
			NetworkID: "eth",
			ChainID:   vaa.ChainIDEthereum,
		},
	}

	// configure p2p
	nodeName := fmt.Sprintf("g-%d", mockGuardianIndex)
	networkID := "/wormhole/localdev"
	zeroPeerId, err := libp2p_peer.IDFromPublicKey(gs.p2pKeys[0].GetPublic())
	if err != nil {
		return err
	}
	bootstrapPeers := fmt.Sprintf("/ip4/127.0.0.1/udp/11000/quic/p2p/%s", zeroPeerId.String())
	p2pPort := uint(11000 + mockGuardianIndex)

	guardianOptions := []GuardianOption{
		GuardianOptionWatchers(watcherConfigs),
		GuardianOptionAccountant("", "", false), // effectively disable accountant
		GuardianOptionGovernor(false),           // disable governor
		GuardianOptionP2P(gs.p2pKeys[mockGuardianIndex], networkID, bootstrapPeers, nodeName, false, p2pPort),
	}

	guardianNode := NewGuardianNode(
		rootCtx,
		rootCtxCancel,
		env,
		db,
		gk,
		nil,
	)

	if err := supervisor.Run(rootCtx, nodeName, guardianNode.Run(guardianOptions...)); err != nil {
		return err
	}

	return nil
}

func TestNodes(t *testing.T) {
	readiness.NoPanic = true // otherwise we'd panic when running multiple guardians

	rootCtx := context.Background()
	logger, err := zap.NewDevelopment()
	assert.NoError(t, err)

	supervisor.New(rootCtx, logger, func(ctx context.Context) error {
		gs := newMockGuardianSet(19)

		for i := 0; i < 19; i++ {
			err := gs.addMockGuardian(ctx, uint(i))
			assert.NoError(t, err)
		}

		timer := time.NewTimer(time.Second * 10)
		<-timer.C
		return nil
	})

	<-rootCtx.Done()
}
