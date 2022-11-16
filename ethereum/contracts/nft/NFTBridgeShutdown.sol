// contracts/NFTBridgeShutdown.sol
// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.0;

import "./NFTBridgeGovernance.sol";

import "@openzeppelin/contracts/proxy/ERC1967/ERC1967Upgrade.sol";

/**
 * @title  BridgeShutdown
 * @notice This contract implements a stripped-down version of the NFT bridge
 *         asset transfer protocol that is a drop-in replacement for the
 *         NFTBridge implementation contract, effectively disabling all
 *         non-governance functionality.
 *         In particular, sending and receiving assets is disabled, but the
 *         contract remains upgradeable through governance.
 */
contract NFTBridgeShutdown is NFTBridgeGovernance {

    function initialize() public {
        address implementation = ERC1967Upgrade._getImplementation();
        setInitialized(implementation);

        // this function needs to be exposed for an upgrade to pass
        // NOTE: leave this function empty! It specifically does not have an
        // 'initializer' modifier, to allow this contract to be upgraded to
        // multiple times.
    }
}
