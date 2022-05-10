// contracts/Implementation.sol
// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.0;
pragma experimental ABIEncoderV2;

import "./Governance.sol";

import "@openzeppelin/contracts/proxy/ERC1967/ERC1967Upgrade.sol";

contract Setup is Setters, ERC1967Upgrade {
    function setup(
        address implementation,
        address[] memory initialGuardians,
        uint16 governanceChainId,
        bytes32 governanceContract
    ) public {
        require(initialGuardians.length > 0, "no guardians specified");

        setOwner(msg.sender);

        Structs.GuardianSet memory initialGuardianSet = Structs.GuardianSet({
            keys : initialGuardians,
            expirationTime : 0
        });

        storeGuardianSet(initialGuardianSet, 0);
        // initial guardian set index is 0, which is the default value of the storage slot anyways

        setGovernanceChainId(governanceChainId);
        setGovernanceContract(governanceContract);

        _upgradeTo(implementation);
    }
}
