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
		directEmitterConfig = emitterConfig{
			{chainId: vaa.ChainIDEthereum, addr: "000000000000000000000000000000000000000000000000656e64706f696e78"},
		}
		arEmitterConfig = emitterConfig{
			{chainId: vaa.ChainIDEthereum, addr: "000000000000000000000000cafd2f0a35a4459fa40c0517e17e6fa2939441ca"},
			{chainId: vaa.ChainIDBSC, addr: "000000000000000000000000cafd2f0a35a4459fa40c0517e17e6fa2939441ca"},
			{chainId: vaa.ChainIDPolygon, addr: "000000000000000000000000cafd2f0a35a4459fa40c0517e17e6fa2939441ca"},
			{chainId: vaa.ChainIDAvalanche, addr: "000000000000000000000000cafd2f0a35a4459fa40c0517e17e6fa2939441ca"},
			{chainId: vaa.ChainIDFantom, addr: "000000000000000000000000cafd2f0a35a4459fa40c0517e17e6fa2939441ca"},
			{chainId: vaa.ChainIDCelo, addr: "000000000000000000000000cafd2f0a35a4459fa40c0517e17e6fa2939441ca"},
			{chainId: vaa.ChainIDSui, addr: "00000000000000000000000057f4e0ba41a7045e29d435bc66cc4175f381eb700e6ec16d4fdfe92e5a4dff9f"},
			{chainId: vaa.ChainIDSolana, addr: "0000000000000000000000003vxKRPwUTiEkeUVyoZ9MXFe1V71sRLbLqu1gRYaWmehQ"},
			{chainId: vaa.ChainIDBase, addr: "000000000000000000000000aE8dc4a7438801Ec4edC0B035EcCCcF3807F4CC1"},
			{chainId: vaa.ChainIDMoonbeam, addr: "000000000000000000000000cafd2f0a35a4459fa40c0517e17e6fa2939441ca"},
		}
	} else if env == common.TestNet {
		directEmitterConfig = emitterConfig{
			{chainId: vaa.ChainIDEthereum, addr: "000000000000000000000000000000000000000000000000656e64706f696e76"},
		}
		arEmitterConfig = emitterConfig{
			{chainId: vaa.ChainIDEthereum, addr: "9563a59c15842a6f322b10f69d1dd88b41f2e97b"},
			{chainId: vaa.ChainIDBSC, addr: "9563a59c15842a6f322b10f69d1dd88b41f2e97b"},
			{chainId: vaa.ChainIDPolygon, addr: "9563a59c15842a6f322b10f69d1dd88b41f2e97b"},
			{chainId: vaa.ChainIDAvalanche, addr: "9563a59c15842a6f322b10f69d1dd88b41f2e97b"},
			{chainId: vaa.ChainIDFantom, addr: "9563a59c15842a6f322b10f69d1dd88b41f2e97b"},
			{chainId: vaa.ChainIDCelo, addr: "9563a59c15842a6f322b10f69d1dd88b41f2e97b"},
			{chainId: vaa.ChainIDSui, addr: "b30040e5120f8cb853b691cb6d45981ae884b1d68521a9dc7c3ae881c0031923"},
			{chainId: vaa.ChainIDBase, addr: "ae8dc4a7438801ec4edc0b035eccccf3807f4cc1"},
			{chainId: vaa.ChainIDMoonbeam, addr: "9563a59c15842a6f322b10f69d1dd88b41f2e97b"},
			{chainId: vaa.ChainIDSolana, addr: "3bPRWXqtSfUaCw3S4wdgvypQtsSzcmvDeaqSqPDkncrg"},
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
		emitters[emitterKey{emitterChainId: emitter.chainId, emitterAddr: addr}] = !emitter.logOnly
	}
	return emitters, nil
}
