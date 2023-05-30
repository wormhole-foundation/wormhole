// contracts/mock/relayer/MockRelayerIntegration.sol
// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.0;

import "../../libraries/external/BytesLib.sol";
import "../../interfaces/IWormhole.sol";
import "../../interfaces/relayer/IWormholeRelayerTyped.sol";
import "../../interfaces/relayer/IWormholeReceiver.sol";

import {toWormholeFormat} from "../../libraries/relayer/Utils.sol";

struct XAddress {
    uint16 chainId;
    bytes32 addr;
}

struct DeliveryData {
    bytes32 sourceAddress;
    uint16 sourceChain;
    bytes32 deliveryHash;
    bytes payload;
    bytes[] additionalVaas;
}

contract MockRelayerIntegration is IWormholeReceiver {
    using BytesLib for bytes;

    // wormhole instance on this chain
    IWormhole immutable wormhole;

    // trusted relayer contract on this chain
    IWormholeRelayer immutable relayer;

    // deployer of this contract
    address immutable owner;

    // latest delivery data
    DeliveryData latestDeliveryData;

    // mapping of other MockRelayerIntegration contracts
    mapping(uint16 => bytes32) registeredContracts;

    bytes[] messageHistory;

    enum Version {
        SEND,
        SEND_WITH_ADDITIONAL_VAA,
        FORWARD,
        MULTIFORWARD
    }

    struct Message {
        Version version;
        bytes message;
        bytes forwardMessage;
    }

    constructor(address _wormholeCore, address _coreRelayer) {
        wormhole = IWormhole(_wormholeCore);
        relayer = IWormholeRelayer(_coreRelayer);
        owner = msg.sender;
    }

    function sendMessage(
        bytes memory _message,
        uint16 targetChain,
        uint32 gasLimit,
        uint128 receiverValue
    ) public payable returns (uint64 sequence) {
        bytes memory fullMessage = encodeMessage(Message(Version.SEND, _message, bytes("")));
        return sendToEvm(
            targetChain,
            getRegisteredContractAddress(targetChain),
            gasLimit,
            targetChain,
            getRegisteredContractAddress(targetChain),
            receiverValue,
            0,
            fullMessage,
            new VaaKey[](0)
        );
    }

    function sendMessageWithAdditionalVaas(
        bytes memory _message,
        uint16 targetChain,
        uint32 gasLimit,
        uint128 receiverValue,
        VaaKey[] memory vaaKeys
    ) public payable returns (uint64 sequence) {
        bytes memory fullMessage = encodeMessage(Message(Version.SEND, _message, bytes("")));
        return sendToEvm(
            targetChain,
            getRegisteredContractAddress(targetChain),
            gasLimit,
            targetChain,
            getRegisteredContractAddress(targetChain),
            receiverValue,
            0,
            fullMessage,
            vaaKeys
        );
    }

    function sendMessageWithRefund(
        bytes memory _message,
        uint16 targetChain,
        uint32 gasLimit,
        uint128 receiverValue,
        uint16 refundChain,
        address refundAddress
    ) public payable returns (uint64 sequence) {
        bytes memory fullMessage = encodeMessage(Message(Version.SEND, _message, bytes("")));
        return sendToEvm(
            targetChain,
            getRegisteredContractAddress(targetChain),
            gasLimit,
            refundChain,
            refundAddress,
            receiverValue,
            0,
            fullMessage,
            new VaaKey[](0)
        );
    }

    function sendMessageWithForwardedResponse(
        bytes memory _message,
        bytes memory _forwardedMessage,
        uint16 targetChain,
        uint32 gasLimit,
        uint128 receiverValue
    ) public payable returns (uint64 sequence) {
        bytes memory fullMessage =
            encodeMessage(Message(Version.FORWARD, _message, _forwardedMessage));
        return sendToEvm(
            targetChain,
            getRegisteredContractAddress(targetChain),
            gasLimit,
            targetChain,
            getRegisteredContractAddress(targetChain),
            receiverValue,
            0,
            fullMessage,
            new VaaKey[](0)
        );
    }

    function sendMessageWithForwardedResponse(
        bytes memory _message,
        bytes memory _forwardedMessage,
        uint16 targetChain,
        uint32 gasLimit,
        uint128 receiverValue,
        uint16 refundChain,
        address refundAddress
    ) public payable returns (uint64 sequence) {
        bytes memory fullMessage =
            encodeMessage(Message(Version.FORWARD, _message, _forwardedMessage));
        return sendToEvm(
            targetChain,
            getRegisteredContractAddress(targetChain),
            gasLimit,
            refundChain,
            refundAddress,
            receiverValue,
            0,
            fullMessage,
            new VaaKey[](0)
        );
    }

    function sendMessageWithMultiForwardedResponse(
        bytes memory _message,
        bytes memory _forwardedMessage,
        uint16 targetChain,
        uint32 gasLimit,
        uint128 receiverValue
    ) public payable returns (uint64 sequence) {
        bytes memory fullMessage =
            encodeMessage(Message(Version.MULTIFORWARD, _message, _forwardedMessage));
        return sendToEvm(
            targetChain,
            getRegisteredContractAddress(targetChain),
            gasLimit,
            targetChain,
            getRegisteredContractAddress(targetChain),
            receiverValue,
            0,
            fullMessage,
            new VaaKey[](0)
        );
    }

    function sendToEvm(
        uint16 targetChain,
        address destination,
        uint32 gasLimit,
        uint16 refundChain,
        address refundAddress,
        uint128 receiverValue,
        uint256 paymentForExtraReceiverValue,
        bytes memory payload,
        VaaKey[] memory vaaKeys
    ) public payable returns (uint64 sequence) {
        sequence = relayer.sendToEvm{value: msg.value}(
            targetChain,
            destination,
            payload,
            TargetNative.wrap(receiverValue),
            LocalNative.wrap(paymentForExtraReceiverValue),
            Gas.wrap(gasLimit),
            refundChain,
            refundAddress,
            relayer.getDefaultDeliveryProvider(),
            vaaKeys,
            200
        );
    }

    function resend(
        uint16 chainId,
        uint64 sequence,
        uint16 targetChain,
        uint32 newGasLimit,
        uint128 newReceiverValue
    ) public payable returns (uint64 resendSequence) {
        VaaKey memory deliveryVaaKey = VaaKey(chainId, getRegisteredContract(chainId), sequence);
        resendSequence = relayer.resendToEvm{value: msg.value}(
            deliveryVaaKey,
            targetChain,
            TargetNative.wrap(newReceiverValue),
            Gas.wrap(newGasLimit),
            relayer.getDefaultDeliveryProvider()
        );
    }

    function receiveWormholeMessages(
        bytes memory payload,
        bytes[] memory additionalVaas,
        bytes32 sourceAddress,
        uint16 sourceChain,
        bytes32 deliveryHash
    ) public payable override {
        // loop through the array of wormhole observations from the batch and store each payload
        require(msg.sender == address(relayer), "Wrong msg.sender");

        latestDeliveryData =
            DeliveryData(sourceAddress, sourceChain, deliveryHash, payload, additionalVaas);

        Message memory message;
        if (payload.length > 0) {
            message = decodeMessage(payload);
        }

        messageHistory.push(message.message);

        if (message.version == Version.FORWARD || message.version == Version.MULTIFORWARD) {
            relayer.forwardToEvm{value: msg.value}(
                sourceChain,
                getRegisteredContractAddress(sourceChain),
                encodeMessage(Message(Version.SEND, message.forwardMessage, bytes(""))),
                TargetNative.wrap(0),
                LocalNative.wrap(0),
                Gas.wrap(500000),
                sourceChain,
                getRegisteredContractAddress(sourceChain),
                relayer.getDefaultDeliveryProvider(),
                new VaaKey[](0),
                15
            );
        }
        if (message.version == Version.MULTIFORWARD) {
            relayer.forwardToEvm{value: 0}(
                wormhole.chainId(),
                getRegisteredContractAddress(wormhole.chainId()),
                encodeMessage(Message(Version.SEND, message.forwardMessage, bytes(""))),
                TargetNative.wrap(0),
                LocalNative.wrap(0),
                Gas.wrap(500000),
                wormhole.chainId(),
                getRegisteredContractAddress(wormhole.chainId()),
                relayer.getDefaultDeliveryProvider(),
                new VaaKey[](0),
                15
            );
        }
    }

    function getMessage() public view returns (bytes memory) {
        if (messageHistory.length == 0) {
            return new bytes(0);
        }
        return messageHistory[messageHistory.length - 1];
    }

    function getDeliveryData() public view returns (DeliveryData memory deliveryData) {
        deliveryData = latestDeliveryData;
    }

    function getMessageHistory() public view returns (bytes[] memory) {
        return messageHistory;
    }

    function emitterAddress() public view returns (bytes32) {
        return bytes32(uint256(uint160(address(this))));
    }

    function registerEmitter(uint16 chainId, bytes32 emitterAddress_) public {
        require(msg.sender == owner);
        registeredContracts[chainId] = emitterAddress_;
    }

    function registerEmitters(XAddress[] calldata emitters) public {
        require(msg.sender == owner);
        for (uint256 i = 0; i < emitters.length; i++) {
            registeredContracts[emitters[i].chainId] = emitters[i].addr;
        }
    }

    function getRegisteredContract(uint16 chainId) public view returns (bytes32) {
        return registeredContracts[chainId];
    }

    function getRegisteredContractAddress(uint16 chainId) public view returns (address) {
        return address(uint160(uint256(registeredContracts[chainId])));
    }

    function encodeMessage(Message memory message) public pure returns (bytes memory encoded) {
        return abi.encodePacked(
            uint8(message.version),
            uint32(message.message.length),
            message.message,
            uint32(message.forwardMessage.length),
            message.forwardMessage
        );
    }

    function decodeMessage(bytes memory encoded) internal pure returns (Message memory message) {
        uint256 index = 0;
        message.version = Version(encoded.toUint8(index));
        index += 1;
        uint32 length = encoded.toUint32(index);
        index += 4;
        message.message = encoded.slice(index, length);
        index += length;
        length = encoded.toUint32(index);
        index += 4;
        message.forwardMessage = encoded.slice(index, length);
        index += length;
        require(index == encoded.length, "Decoded message incorrectly");
    }

    receive() external payable {}
}
