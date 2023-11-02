// contracts/Implementation.sol
// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.0;
pragma experimental ABIEncoderV2;

import "./Governance.sol";

import "@openzeppelin/contracts/proxy/ERC1967/ERC1967Upgrade.sol";

/**
@title Setup Contract
@dev This contract is used to setup a wormhole contract.
     It inherits from the standard ERC1967Upgrade, as well as the Setters contract.
*/
contract Setup is Setters, ERC1967Upgrade {
    /// @notice sets up a wormhole contract
    /// @param implementation the address of the wormhole contract
    /// @param initialGuardians the initial guardian set
    /// @param chainId the chain id
    /// @param governanceChainId the chain id of the governance
    /// @param governanceContract the contract of the governance
    /// @param evmChainId the EVM chain id
    function setup(
        address implementation,
        address[] memory initialGuardians,
        uint16 chainId,
        uint16 governanceChainId,
        bytes32 governanceContract,
        uint256 evmChainId
    ) public {
        require(initialGuardians.length > 0, "no guardians specified");

        Structs.GuardianSet memory initialGuardianSet = Structs.GuardianSet({
            keys : initialGuardians,
            expirationTime : 0
        });

        storeGuardianSet(initialGuardianSet, 0);
        // initial guardian set index is 0, which is the default value of the storage slot anyways

        setChainId(chainId);

        setGovernanceChainId(governanceChainId);
        setGovernanceContract(governanceContract);

        setEvmChainId(evmChainId);

        _upgradeTo(implementation);

        // See https://github.com/wormhole-foundation/wormhole/issues/1930 for
        // why we set this here
        setInitialized(implementation);
    }
}
