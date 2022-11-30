// contracts/BridgeShutdown.sol
// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.0;

import "./BridgeGovernance.sol";

import "@openzeppelin/contracts/security/ReentrancyGuard.sol";
import "@openzeppelin/contracts/proxy/ERC1967/ERC1967Upgrade.sol";

/**
 * @title  BridgeShutdown
 * @notice This contract implements a stripped-down version of the token bridge
 *         asset transfer protocol that is a drop-in replacement for the Bridge
 *         implementation contract, effectively disabling all non-governance
 *         functionality.
 *         In particular, sending and receiving assets is disabled, but the
 *         contract remains upgradeable through governance.
 * @dev    Technically the ReentrancyGuard is not used in this contract,
 *         but it adds a storage variable, so as a matter of principle, we
 *         inherit that here too in order keep the storage layout identical to
 *         the actual implementation contract (which does use the reentrancy
 *         guard).
 */
contract BridgeShutdown is BridgeGovernance, ReentrancyGuard {

    function initialize() public {
        address implementation = ERC1967Upgrade._getImplementation();
        setInitialized(implementation);

        // this function needs to be exposed for an upgrade to pass
        // NOTE: leave this function empty! It specifically does not have an
        // 'initializer' modifier, to allow this contract to be upgraded to
        // multiple times.
    }
}
