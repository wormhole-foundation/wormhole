// contracts/PythSetup.sol
// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.0;
pragma experimental ABIEncoderV2;

import "./PythSetters.sol";

import "@openzeppelin/contracts/proxy/ERC1967/ERC1967Upgrade.sol";

contract PythSetup is PythSetters, ERC1967Upgrade {
    function setup(
        address implementation,

        uint16 chainId,
        address wormhole,

        uint16 governanceChainId,
        bytes32 governanceContract,

        uint16 pyth2WormholeChainId,
        bytes32 pyth2WormholeEmitter
    ) public {
        setChainId(chainId);

        setWormhole(wormhole);

        setGovernanceChainId(governanceChainId);
        setGovernanceContract(governanceContract);

        setPyth2WormholeChainId(pyth2WormholeChainId);
        setPyth2WormholeEmitter(pyth2WormholeEmitter);

        _upgradeTo(implementation);
    }
}
