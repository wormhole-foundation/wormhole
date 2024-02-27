package accountant

import (
	"fmt"

	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
)

type emitterConfigEntry struct {
	chainId vaa.ChainID
	addr    string
	logOnly bool
}

type emitterConfig []emitterConfigEntry

// nttGetEmitters returns the set of direct NTT and AR emitters based on the environment passed in.
// The automatic relayers for mainnet and testnet are defined here:
// https://github.com/wormhole-foundation/connect-sdk/blob/d15564b12213016a8ead4a3638593ab2eaf386ca/core/base/src/constants/contracts/tokenBridgeRelayer.ts#L6-L34
func nttGetEmitters(env common.Environment) (validEmitters, validEmitters, error) {
	var directEmitterConfig, arEmitterConfig emitterConfig
	if env == common.MainNet {
		directEmitterConfig = emitterConfig{}
		arEmitterConfig = emitterConfig{}
	} else if env == common.TestNet {
		directEmitterConfig = emitterConfig{}
		arEmitterConfig = emitterConfig{}
	} else {
		// Every other environment uses the devnet ones.
		directEmitterConfig = emitterConfig{
			{chainId: vaa.ChainIDEthereum, addr: "000000000000000000000000855FA758c77D68a04990E992aA4dcdeF899F654A"},
			{chainId: vaa.ChainIDEthereum, addr: "000000000000000000000000fA2435Eacf10Ca62ae6787ba2fB044f8733Ee843"},
			{chainId: vaa.ChainIDBSC, addr: "000000000000000000000000fA2435Eacf10Ca62ae6787ba2fB044f8733Ee843"},
			{chainId: vaa.ChainIDBSC, addr: "000000000000000000000000855FA758c77D68a04990E992aA4dcdeF899F654A"},
		}
		arEmitterConfig = emitterConfig{
			{chainId: vaa.ChainIDEthereum, addr: "00000000000000000000000053855d4b64e9a3cf59a84bc768ada716b5536bc5"},
			{chainId: vaa.ChainIDBSC, addr: "00000000000000000000000053855d4b64e9a3cf59a84bc768ada716b5536bc5"},
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
		emitters[emitterKey{emitterChainId: emitter.chainId, emitterAddr: addr}] = !emitter.logOnly
	}
	return emitters, nil
}
