// contracts/BridgeSetup.sol
// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.0;
pragma experimental ABIEncoderV2;

import "./BridgeGovernance.sol";

import "@openzeppelin/contracts/proxy/ERC1967/ERC1967Upgrade.sol";

contract BridgeSetup is BridgeSetters, ERC1967Upgrade {
    function setup(
        address implementation,
        uint16 chainId,
        address wormhole,
        uint16 governanceChainId,
        bytes32 governanceContract,
        address tokenImplementation,
        address WETH
    ) public {
        setChainId(chainId);

        setWormhole(wormhole);

        setGovernanceChainId(governanceChainId);
        setGovernanceContract(governanceContract);

        setTokenImplementation(tokenImplementation);

        setWETH(WETH);

        _upgradeTo(implementation);
    }
}
