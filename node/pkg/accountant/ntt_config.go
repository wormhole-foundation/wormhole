package accountant

import (
	"fmt"

	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
)

type emitterConfigEntry struct {
	chainId vaa.ChainID
	addr    string
}

type emitterConfig []emitterConfigEntry

// nttGetEmitters returns the set of direct NTT and AR emitters based on the environment passed in.
func nttGetEmitters(env common.Environment) (validEmitters, validEmitters, error) {
	var directEmitterConfig, arEmitterConfig emitterConfig
	if env == common.MainNet {
		directEmitterConfig = emitterConfig{
			{chainId: vaa.ChainIDEthereum, addr: "000000000000000000000000000000000000000000000000656e64706f696e78"},
		}
		arEmitterConfig = emitterConfig{
			{chainId: vaa.ChainIDEthereum, addr: "000000000000000000000000000000000000000000000000656e64706f696e77"},
		}
	} else if env == common.TestNet {
		directEmitterConfig = emitterConfig{
			{chainId: vaa.ChainIDEthereum, addr: "000000000000000000000000000000000000000000000000656e64706f696e76"},
		}
		arEmitterConfig = emitterConfig{
			{chainId: vaa.ChainIDEthereum, addr: "000000000000000000000000000000000000000000000000656e64706f696e75"},
		}
	} else {
		// Every other environment uses the devnet ones.
		directEmitterConfig = emitterConfig{
			{chainId: vaa.ChainIDEthereum, addr: "000000000000000000000000000000000000000000000000656e64706f696e74"},
		}
		arEmitterConfig = emitterConfig{
			{chainId: vaa.ChainIDEthereum, addr: "000000000000000000000000000000000000000000000000656e64706f696e73"},
		}
	}

	directEmitters, err := nttBuildEmitterMap(directEmitterConfig)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to build direct emitter map: %w", err)
	}

	arEmitters, err := nttBuildEmitterMap(arEmitterConfig)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to build AR emitter map: %w", err)
	}

	return directEmitters, arEmitters, nil
}

// nttBuildEmitterMap converts a vector of configured emitters to an emitters map.
func nttBuildEmitterMap(cfg emitterConfig) (validEmitters, error) {
	emitters := make(validEmitters)
	for _, emitter := range cfg {
		addr, err := vaa.StringToAddress(emitter.addr)
		if err != nil {
			return nil, fmt.Errorf(`failed to parse emitter address "%s": %w`, emitter.addr, err)
		}
		emitters[emitterKey{emitterChainId: emitter.chainId, emitterAddr: addr}] = struct{}{}
	}
	return emitters, nil
}
