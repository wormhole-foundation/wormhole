package ethereum

import (
	"context"
	"fmt"
	"github.com/certusone/wormhole/bridge/pkg/common"
	"github.com/certusone/wormhole/bridge/pkg/ethereum/abi"
	"github.com/certusone/wormhole/bridge/pkg/supervisor"
	"github.com/certusone/wormhole/bridge/pkg/vaa"
	eth_common "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"sync"
)

type (
	EthBridgeWatcher struct {
		url    string
		bridge eth_common.Address

		evChan chan *common.ChainLock
	}
)

func NewEthBridgeWatcher(url string, bridge eth_common.Address, events chan *common.ChainLock) *EthBridgeWatcher {
	return &EthBridgeWatcher{url: url, bridge: bridge, evChan: events}
}

func (e *EthBridgeWatcher) Run(ctx context.Context) error {
	c, err := ethclient.Dial(e.url)
	if err != nil {
		return fmt.Errorf("dialing eth client failed: %w", err)
	}

	f, err := abi.NewWormholeBridgeFilterer(e.bridge, c)
	if err != nil {
		return fmt.Errorf("could not create wormhole bridge filter: %w", err)
	}

	sink := make(chan *abi.WormholeBridgeLogTokensLocked)
	subscription, err := f.WatchLogTokensLocked(nil, sink, nil, nil)
	if err != nil {
		return fmt.Errorf("failed to subscribe to eth events: %w", err)
	}
	defer subscription.Unsubscribe()

	wg := sync.WaitGroup{}
	wg.Add(1)

	go func() {
		defer wg.Done()

		for {
			select {
			case e := <-subscription.Err():
				err = e
				return
			case ev := <-sink:
				lock := &common.ChainLock{
					SourceAddress: ev.Sender,
					TargetAddress: ev.Recipient,
					SourceChain:   vaa.ChainIDEthereum,
					TargetChain:   vaa.ChainID(ev.TargetChain),
					TokenChain:    vaa.ChainID(ev.TokenChain),
					TokenAddress:  ev.Token,
					Amount:        ev.Amount,
				}
				e.evChan <- lock
			}
		}

	}()

	supervisor.Signal(ctx, supervisor.SignalHealthy)
	wg.Wait()

	return err
}
