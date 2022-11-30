// contracts/NFTBridgeSetup.sol
// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.0;
pragma experimental ABIEncoderV2;

import "./NFTBridgeGovernance.sol";

import "@openzeppelin/contracts/proxy/ERC1967/ERC1967Upgrade.sol";

contract NFTBridgeSetup is NFTBridgeSetters, ERC1967Upgrade {
    function setup(
        address implementation,
        uint16 chainId,
        address wormhole,
        uint16 governanceChainId,
        bytes32 governanceContract,
        address tokenImplementation,
        uint8 finality,
        uint256 evmChainId
    ) public {
        setChainId(chainId);

        setWormhole(wormhole);

        setGovernanceChainId(governanceChainId);
        setGovernanceContract(governanceContract);

        setTokenImplementation(tokenImplementation);

        setFinality(finality);

        setEvmChainId(evmChainId);

        _upgradeTo(implementation);

        // See https://github.com/wormhole-foundation/wormhole/issues/1930 for
        // why we set this here
        setInitialized(implementation);
    }
}
