// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.0;

import "../../libraries/external/BytesLib.sol";

import "./CoreRelayerGetters.sol";
import "./CoreRelayerSetters.sol";
import "../../interfaces/relayer/IWormholeRelayerInternalStructs.sol";
import "../../interfaces/relayer/IForwardWrapper.sol";
import "./CoreRelayerMessages.sol";

import "@openzeppelin/contracts/proxy/ERC1967/ERC1967Upgrade.sol";

import "../../interfaces/IWormhole.sol";

abstract contract CoreRelayerGovernance is
    CoreRelayerGetters,
    CoreRelayerSetters,
    CoreRelayerMessages,
    ERC1967Upgrade
{
    using BytesLib for bytes;

    function submitContractUpgrade(bytes memory vaa) public {
        (bool success, bytes memory reason) =
            getWormholeRelayerCallerAddress().delegatecall(abi.encodeWithSignature("submitContractUpgrade(bytes)", vaa));
        require(success, string(reason));
    }

    function registerCoreRelayerContract(bytes memory vaa) public {
        (bool success, bytes memory reason) = getWormholeRelayerCallerAddress().delegatecall(
            abi.encodeWithSignature("registerCoreRelayerContract(bytes)", vaa)
        );
        require(success, string(reason));
    }

    function submitRecoverChainId(bytes memory vaa) public {
        (bool success, bytes memory reason) = getWormholeRelayerCallerAddress().delegatecall(
            abi.encodeWithSignature("submitRecoverChainId(bytes)", vaa)
        );
        require(success, string(reason));
    }


    function setDefaultRelayProvider(bytes memory vaa) public {
        (bool success, bytes memory reason) = getWormholeRelayerCallerAddress().delegatecall(
            abi.encodeWithSignature("setDefaultRelayProvider(bytes)", vaa)
        );
        require(success, string(reason));
    }
}
