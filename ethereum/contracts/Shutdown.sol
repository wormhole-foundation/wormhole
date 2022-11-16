// contracts/Shutdown.sol
// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.0;
pragma experimental ABIEncoderV2;

import "./Governance.sol";

import "@openzeppelin/contracts/proxy/ERC1967/ERC1967Upgrade.sol";

/**
 * @title  Shutdown
 * @notice This contract implements a stripped-down version of the Wormhole core
 *         messaging protocol that is a drop-in replacement for Wormhole's
 *         implementation contract, effectively disabling all non-governance
 *         functionality.
 *         In particular, outgoing messages are disabled, but the contract
 *         remains upgradeable through governance.
 */
contract Shutdown is Governance  {

    function initialize() public {
        address implementation = ERC1967Upgrade._getImplementation();
        setInitialized(implementation);

        // this function needs to be exposed for an upgrade to pass
        // NOTE: leave this function empty! It specifically does not have an
        // 'initializer' modifier, to allow this contract to be upgraded to
        // multiple times.
    }
}
