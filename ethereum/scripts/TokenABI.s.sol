// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.0;

import "forge-std/Script.sol";
import "../contracts/Messages.sol";
import "../contracts/Structs.sol";
import "../contracts/bridge/Bridge.sol";
import "../contracts/bridge/BridgeStructs.sol";

contract BridgeTest is Bridge, Script {

    // forge script scripts/TokenABI.s.sol -s "token_constructor_args(bytes, address)" 0x01000000020d00b7ba3819d44da891c74c583e29eb2222dd37dabbe7929bdbf4f2186bbcc721085d85d9906bbd8ca5ae62cdf30c7555dc4c57fd15f84a0161c27e91846203439c0102736caa697f6c17c2e6b0526291b0e6b4dec760a8494df7f69c93be3df1956224637ef962be9a28ef2dbeebe6bdb30311d9f2394966a1bb170634bd69913abfb200038c598c6e7c288c5dbb0f0008c38168d3f00ac8da7b3ad5420f30c8808c94a8a972c090d25da27558f1b8f8d30f894850d3139f4df92c8e8736be7803d397f33e0006649e6aca07694046fd94b5851ff3711783d4f4c8e0319f9de9431232cb153bce2ff2ac0f7bfad6f3db461571cd6ecffc99d7740a7b653d2f6a25908d821d9ca70107b31051fda4062585f80b291978a480cae6c9191d37a67bc2e1e61db8e97907fa71b5064d2ada48b4cd2f8c4def7fd50484004d1ceb3438a8f67ea071a31a6a88000af4842bbcd0fad425bd3b82bc3b1acefd72555fd1fbb49b71700ec2b41ac6309f20222e24c557f4ad6af35d96f1d4c38fb25177e027a22d2d071956d5d45985ba000bc0ebb4202aae662de331bce75d5e49ea97ac9a74df65006250c96ca9d82a16be6e78c577004a6059169aa7640436e1e5deef5d80bfa52784cf82f67bb368e066010d14586fa1f6f37d2c4d0eae78c42ecc3c9fc6bf17b3a57406382165d615cfb4a1651b979419c42e40a3f62fbe05eb3ff4bafac0af30c15a060e39d935776e54cc000e98d02eb76745301cb5fb12e6b0c7e3e9be347460ed51be360828c46be3bc40ef622f9234fb443b431db9e98980a7165b36eda10bd37abf6998156ebbdf96c4b6010fda503c3deb9c937709ab5742c4a44ed29f04664585c4c73568cd3b4863e1e2326b9cab4d1b139d9698585bb8abcbdc4072b3f98fdfe1b50fa35656c1451f862400106644b7697f41052d4d7c1685d342df4828c7ba7231f86c04476805271c58b4e30614ee43988072decec39f0400a48583f7b6d0fb109516385f73a64ce2a2b16501111cc03c23da18a3ed794cb944aa6d131306c243d13f207796c9ea9430a6c7da063b0ffbc75416c924b588ecc24c3d1c6136ea8e181a4f3d8c1d3d1831c7d4ae7301120b48f7b0c43cb43b4541d179f4bdfe6b9c83289b5b7cd494f6ea33eec062b36408606f4ad406365539d6b3a6b59b2eeae70baf0266c341fb476c8092d64ebd620062cb923534d80000000e000000000000000000000000796dff6d74f3e27060b71255fe517bfb23c93eed00000000000000560102000000000000000000000000765de816845861e75a25fca122bb6898b8b1282a000e12635553440000000000000000000000000000000000000000000000000000000043656c6f20446f6c6c6172000000000000000000000000000000000000000000 $(worm contract mainnet ethereum TokenBridge)
    function token_constructor_args(bytes calldata encodedVM, address tokenBridge) public {
        Messages m = new Messages();
        Structs.VM memory vm = m.parseVM(encodedVM);
        BridgeStructs.AssetMeta memory meta = parseAssetMeta(vm.payload);
        token_constructor_args(bytes32ToString(meta.name), bytes32ToString(meta.symbol), meta.decimals, vm.sequence, tokenBridge, meta.tokenChain, meta.tokenAddress);
    }

    // forge script scripts/TokenABI.s.sol -s "token_constructor_args(string,string,uint8,uint64,address,uint16,bytes32)" "Wrapped Ether" "WETH" 18 69201 0x796Dff6D74F3E27060B71255Fe517BFb23C93eed 2 0x000000000000000000000000c02aaa39b223fe8d0a0e5c4f27ead9083c756cc2
    function token_constructor_args(string memory name, string memory symbol, uint8 decimals, uint64 sequence, address tokenBridge, uint16 tokenChain, bytes32 tokenAddress) view public {
        bytes memory initialisationArgs = abi.encodeWithSelector(
            TokenImplementation.initialize.selector,
            name,
            symbol,
            decimals,
            sequence,

            tokenBridge,

            tokenChain,
            tokenAddress
        );

        bytes memory constructorArgs = abi.encode(tokenBridge, initialisationArgs);

        console.logBytes(constructorArgs);
    }
}
