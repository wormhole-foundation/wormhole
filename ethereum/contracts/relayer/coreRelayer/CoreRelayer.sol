// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.0;

import "@openzeppelin/contracts/proxy/ERC1967/ERC1967Upgrade.sol";

import "./CoreRelayerSendOverloads.sol";
import "./CoreRelayerDelivery.sol";

/*
 * Inheritance Graph:
 * Note: CoreRelayer prefix omitted
 *
 *          SendOverloads -> Send -v 
 *        /                        |--> Messages -> Getters -v
 * CoreRelayer                     |                         |-> State
 *        \                        |--> Setters -------------^
 *         Delivery ---------------^
 */

contract CoreRelayer is CoreRelayerSendOverloads, CoreRelayerDelivery, ERC1967Upgrade {
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

    function submitContractUpgrade(bytes memory vaa) public {
        (bool success, bytes memory reason) = getWormholeRelayerCallerAddress().delegatecall(
            abi.encodeWithSignature("submitContractUpgrade(bytes)", vaa)
        );
        require(success, string(reason));
    }

    function registerCoreRelayerContract(bytes memory vaa) public {
        (bool success, bytes memory reason) = getWormholeRelayerCallerAddress().delegatecall(
            abi.encodeWithSignature("registerCoreRelayerContract(bytes)", vaa)
        );
        require(success, string(reason));
    }

    function setDefaultRelayProvider(bytes memory vaa) public {
        (bool success, bytes memory reason) = getWormholeRelayerCallerAddress().delegatecall(
            abi.encodeWithSignature("setDefaultRelayProvider(bytes)", vaa)
        );
        require(success, string(reason));
    }

    function submitRecoverChainId(bytes memory vaa) public {
        (bool success, bytes memory reason) = getWormholeRelayerCallerAddress().delegatecall(
            abi.encodeWithSignature("submitRecoverChainId(bytes)", vaa)
        );
        require(success, string(reason));
    }
}
