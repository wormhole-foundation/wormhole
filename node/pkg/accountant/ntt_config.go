package accountant

import (
	"fmt"

	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/wormhole-foundation/wormhole/sdk"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
)

type emitterConfigEntry struct {
	chainId vaa.ChainID
	addr    string
	logOnly bool
}

type emitterConfig []emitterConfigEntry

// nttGetEmitters returns the set of direct NTT and AR emitters based on the environment passed in.
func nttGetEmitters(env common.Environment) (validEmitters, validEmitters, error) {
	var directEmitterConfig emitterConfig
	arEmitterConfig := sdk.KnownAutomaticRelayerEmitters
	if env == common.MainNet {
		directEmitterConfig = emitterConfig{}
	} else if env == common.TestNet {
		directEmitterConfig = emitterConfig{}
		arEmitterConfig = sdk.KnownTestnetAutomaticRelayerEmitters
	} else {
		// Every other environment uses the devnet ones.
		directEmitterConfig = emitterConfig{
			{chainId: vaa.ChainIDEthereum, addr: "000000000000000000000000855FA758c77D68a04990E992aA4dcdeF899F654A"},
			{chainId: vaa.ChainIDEthereum, addr: "000000000000000000000000fA2435Eacf10Ca62ae6787ba2fB044f8733Ee843"},
			{chainId: vaa.ChainIDBSC, addr: "000000000000000000000000fA2435Eacf10Ca62ae6787ba2fB044f8733Ee843"},
			{chainId: vaa.ChainIDBSC, addr: "000000000000000000000000855FA758c77D68a04990E992aA4dcdeF899F654A"},
		}
		arEmitterConfig = sdk.KnownDevnetAutomaticRelayerEmitters
	}

	// Build the direct emitter map, setting the payload based on whether or not the config says it should be log only.
	directEmitters := make(validEmitters)
	for _, emitter := range directEmitterConfig {
		addr, err := vaa.StringToAddress(emitter.addr)
		if err != nil {
			return nil, nil, fmt.Errorf(`failed to parse direct emitter address "%s": %w`, emitter.addr, err)
		}
		ek := emitterKey{emitterChainId: emitter.chainId, emitterAddr: addr}
		if _, exists := directEmitters[ek]; exists {
			return nil, nil, fmt.Errorf(`duplicate direct emitter "%s:%s"`, emitter.chainId.String(), emitter.addr)
		}
		directEmitters[ek] = !emitter.logOnly
	}

	// Build the automatic relayer emitter map based on the standard config in the SDK.
	arEmitters := make(validEmitters)
	for _, emitter := range arEmitterConfig {
		addr, err := vaa.StringToAddress(emitter.Addr)
		if err != nil {
			return nil, nil, fmt.Errorf(`failed to parse AR emitter address "%s": %w`, emitter.Addr, err)
		}
		ek := emitterKey{emitterChainId: emitter.ChainId, emitterAddr: addr}
		if _, exists := directEmitters[ek]; exists {
			return nil, nil, fmt.Errorf(`duplicate AR emitter "%s:%s"`, emitter.ChainId.String(), emitter.Addr)
		}
		arEmitters[ek] = true
	}

	return directEmitters, arEmitters, nil
}
