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
		directEmitterConfig = emitterConfig{
			{chainId: vaa.ChainIDSolana, addr: "cf5f3614e2cd9b374558f35c7618b25f0d306d5e749b7d29cc030a1a15686238"},
			{chainId: vaa.ChainIDEthereum, addr: "000000000000000000000000Db55492d7190D1baE8ACbE03911C4E3E7426870c"},
			{chainId: vaa.ChainIDArbitrum, addr: "000000000000000000000000D1a8AB69e00266e8B791a15BC47514153A5045a6"},
			{chainId: vaa.ChainIDOptimism, addr: "0000000000000000000000009bD8b7b527CA4e6738cBDaBdF51C22466756073d"},
			{chainId: vaa.ChainIDBase, addr: "000000000000000000000000D1a8AB69e00266e8B791a15BC47514153A5045a6"},
		}
	} else if env == common.TestNet {
		directEmitterConfig = emitterConfig{
			{chainId: vaa.ChainIDSolana, addr: "7e6436b671cce379a1fa9833783e28b36d39a00e2cdc6bfeab5d2d836eb61c7f"},
			{chainId: vaa.ChainIDSepolia, addr: "0000000000000000000000001fdc902e30b188fd2ba976b421cb179943f57896"},
			{chainId: vaa.ChainIDArbitrumSepolia, addr: "0000000000000000000000000e24d17d7467467b39bf64a9dff88776bd6c74d7"},
			{chainId: vaa.ChainIDBaseSepolia, addr: "0000000000000000000000001e072169541f1171e427aa44b5fd8924bee71b0e"},
			{chainId: vaa.ChainIDOptimismSepolia, addr: "00000000000000000000000041265eb2863bf0238081f6aeefef73549c82c3dd"},
			// V1 Testnet Deployment Emitters
			{chainId: vaa.ChainIDSolana, addr: "cf5f3614e2cd9b374558f35c7618b25f0d306d5e749b7d29cc030a1a15686238"},
			{chainId: vaa.ChainIDSepolia, addr: "000000000000000000000000649fF7B32C2DE771043ea105c4aAb2D724497238"},
			{chainId: vaa.ChainIDArbitrumSepolia, addr: "000000000000000000000000fA42603152E4f133F5F3DA610CDa91dF5821d8bc"},
			{chainId: vaa.ChainIDBaseSepolia, addr: "000000000000000000000000149987472333cD48ac6D28293A338a1EEa6Be7EE"},
			{chainId: vaa.ChainIDOptimismSepolia, addr: "000000000000000000000000eCF0496DE01e9Aa4ADB50ae56dB550f52003bdB7"},
		}
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
