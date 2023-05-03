// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.0;

import "@openzeppelin/contracts/proxy/ERC1967/ERC1967Upgrade.sol";

import "./CoreRelayerSendOverloads.sol";
import "./CoreRelayerDelivery.sol";

/*
 * Inheritance Graph:
 *
 *                    CoreRelayer 
 *                   /           \
 * CoreRelayerSendOverloads   CoreRelayerDelivery                       
 *                 |                |
 *          CoreRelayerSend         |
 *                  \              /
 *               CoreRelayerGovernance
 *                   /          \
 *    CoreRelayerSetters    CoreRelayerMessages
 *                  |            |
 *                  |       CoreRelayerGetters
 *                   \           /
 *                 CoreRelayerState
 *
 */

contract CoreRelayer is CoreRelayerSendOverloads, CoreRelayerDelivery {
    error ImplementationAlreadyInitialized();

    constructor(address _forwardWrapper) CoreRelayerGetters(_forwardWrapper) {}

    // this function needs to be exposed for an upgrade to pass
    function initialize() public virtual initializer {}

    modifier initializer() {
        address impl = ERC1967Upgrade._getImplementation();

        if (isInitialized(impl)) {
            revert ImplementationAlreadyInitialized();
        }

        setInitialized(impl);

        _;
    }
}
