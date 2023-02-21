// contracts/Structs.sol
// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.0;

import "../interfaces/IWormhole.sol";

abstract contract CoreRelayerStructs {
    //This first group of structs are external facing API objects,
    //which should be considered untrusted and unmodifiable

    struct MultichainSend {
        address relayProviderAddress;
        Send[] requests;
    }

    // struct TargetDeliveryParameters {
    //     // encoded batchVM to be delivered on the target chain
    //     bytes encodedVM;
    //     // Index of the delivery VM in a batch
    //     uint8 deliveryIndex;
    //     uint8 multisendIndex;
    //     //uint32 targetCallGasOverride;
    // }

    struct TargetDeliveryParametersSingle {
        // encoded batchVM to be delivered on the target chain
        bytes[] encodedVMs;
        // Index of the delivery VM in a batch
        uint8 deliveryIndex;
        // Index of the target chain inside the delivery VM
        uint8 multisendIndex;
        // relayer refund address
        address payable relayerRefundAddress;
    }
    // Optional gasOverride which can be supplied by the relayer
    // uint32 targetCallGasOverride;

    struct TargetRedeliveryByTxHashParamsSingle {
        bytes redeliveryVM;
        bytes[] sourceEncodedVMs;
        address payable relayerRefundAddress;
    }

    struct Send {
        uint16 targetChain;
        bytes32 targetAddress;
        bytes32 refundAddress;
        uint256 maxTransactionFee;
        uint256 receiverValue;
        bytes relayParameters;
    }

    struct ResendByTx {
        uint16 sourceChain;
        bytes32 sourceTxHash;
        uint32 sourceNonce;
        uint16 targetChain;
        uint8 deliveryIndex;
        uint8 multisendIndex;
        uint256 newMaxTransactionFee;
        uint256 newReceiverValue;
        bytes newRelayParameters;
    }

    struct RelayParameters {
        uint8 version; //1
        bytes32 providerAddressOverride;
    }

    //Below this are internal structs

    //Wire Types
    struct DeliveryInstructionsContainer {
        uint8 payloadId; //1
        bool sufficientlyFunded;
        DeliveryInstruction[] instructions;
    }

    struct DeliveryInstruction {
        uint16 targetChain;
        bytes32 targetAddress;
        bytes32 refundAddress;
        uint256 maximumRefundTarget;
        uint256 receiverValueTarget;
        ExecutionParameters executionParameters; //Has the gas limit to execute with
    }

    struct ExecutionParameters {
        uint8 version;
        uint32 gasLimit;
        bytes32 providerDeliveryAddress;
    }

    struct RedeliveryByTxHashInstruction {
        uint8 payloadId; //2
        uint16 sourceChain;
        bytes32 sourceTxHash;
        uint32 sourceNonce;
        uint16 targetChain;
        uint8 deliveryIndex;
        uint8 multisendIndex;
        uint256 newMaximumRefundTarget;
        uint256 newReceiverValueTarget;
        ExecutionParameters executionParameters;
    }

    //End Wire Types

    //Internal usage structs

    struct AllowedEmitterSequence {
        // wormhole emitter address
        bytes32 emitterAddress;
        // wormhole message sequence
        uint64 sequence;
    }

    struct ForwardingRequest {
        bytes deliveryRequestsContainer;
        uint16 rolloverChain;
        uint32 nonce;
        address sender;
        uint256 msgValue;
        bool isValid;
    }
}
