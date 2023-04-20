// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.0;

import "./IWormholeRelayerInternalStructs.sol";
import "./IWormholeReceiver.sol";
import "./IRelayProvider.sol";

interface IForwardWrapper {
    function executeInstruction(
        IWormholeRelayerInternalStructs.DeliveryInstruction memory instruction,
        IWormholeReceiver.DeliveryData memory deliveryData,
        bytes[] memory signedVaas
    ) external payable returns (bool callToTargetContractSucceeded, uint256 transactionFeeRefundAmount);

    function safeRelayProviderSupportsChain(IRelayProvider relayProvider, uint16 chainId)
        external
        view
        returns (bool isSupported);

    function parseUpgrade(bytes memory encodedUpgrade, bytes32 module)
        external
        pure
        returns (ContractUpgrade memory cu);

    function parseRegisterChain(bytes memory encodedRegistration, bytes32 module)
        external
        pure
        returns (RegisterChain memory registerChain);

    function parseUpdateDefaultProvider(bytes memory encodedDefaultProvider, bytes32 module)
        external
        pure
        returns (UpdateDefaultProvider memory defaultProvider);

    struct ContractUpgrade {
        bytes32 module;
        uint8 action;
        uint16 chain;
        address newContract;
    }

    struct RegisterChain {
        bytes32 module;
        uint8 action;
        uint16 chain; //TODO Why is this on this object?
        uint16 emitterChain;
        bytes32 emitterAddress;
    }

    //This could potentially be combined with ContractUpgrade
    struct UpdateDefaultProvider {
        bytes32 module;
        uint8 action;
        uint16 chain;
        address newProvider;
    }
}
