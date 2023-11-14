// contracts/mock/relayer/MockRelayerIntegration.sol
// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.0;

import "../../libraries/external/BytesLib.sol";
import "../../interfaces/IWormhole.sol";
import "../../interfaces/relayer/IWormholeRelayerTyped.sol";
import "../../interfaces/relayer/IWormholeReceiver.sol";
import "../../relayer/wormholeRelayer/WormholeRelayer.sol";

import {toWormholeFormat} from "../../relayer/libraries/Utils.sol";

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
    using LocalNativeLib for LocalNative;

    // wormhole instance on this chain
    IWormhole immutable wormhole;

    // trusted relayer contract on this chain
    WormholeRelayer immutable relayer;

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
        SEND_BACK,
        MULTI_SEND_BACK,
        REENTRANT
    }

    struct Message {
        Version version;
        bytes message;
        bytes forwardMessage;
    }

    constructor(address _wormholeCore, address _coreRelayer) {
        wormhole = IWormhole(_wormholeCore);
        relayer = WormholeRelayer(_coreRelayer);
        owner = msg.sender;
    }

    function sendMessage(
        bytes memory _message,
        uint16 targetChain,
        uint32 gasLimit,
        uint128 receiverValue
    ) public payable returns (uint64 sequence) {
        return _sendMessageWithVersion(
            Message(Version.SEND, _message, bytes("")),
            targetChain,
            gasLimit,
            receiverValue,
            targetChain,
            getRegisteredContractAddress(targetChain)
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
        return _sendMessageWithVersion(
            Message(Version.SEND, _message, bytes("")),
            targetChain,
            gasLimit,
            receiverValue,
            refundChain,
            refundAddress
        );
    }

    function _sendMessageWithVersion(
        Message memory message,
        uint16 targetChain,
        uint32 gasLimit,
        uint128 receiverValue,
        uint16 refundChain,
        address refundAddress
    ) internal returns (uint64 sequence) {
        bytes memory fullMessage = encodeMessage(message);
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
        return _sendMessageWithVersion(
            Message(Version.MULTI_SEND_BACK, _message, _forwardedMessage),
            targetChain,
            gasLimit,
            receiverValue,
            targetChain,
            getRegisteredContractAddress(targetChain)
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
        return _sendMessageWithVersion(
            Message(Version.SEND_BACK, _message, _forwardedMessage),
            targetChain,
            gasLimit,
            receiverValue,
            refundChain,
            refundAddress
        );
    }

    function sendMessageWithForwardedResponse(
        bytes memory _message,
        bytes memory _forwardedMessage,
        uint16 targetChain,
        uint32 gasLimit,
        uint128 receiverValue
    ) public payable returns (uint64 sequence) {
        return _sendMessageWithVersion(
            Message(Version.SEND_BACK, _message, _forwardedMessage),
            targetChain,
            gasLimit,
            receiverValue,
            targetChain,
            getRegisteredContractAddress(targetChain)
        );
    }

    function sendMessageWithReentrantDelivery(
        uint16 targetChain,
        uint32 gasLimit,
        uint128 receiverValue
    ) public payable returns (uint64 sequence) {
        return _sendMessageWithVersion(
            Message(Version.REENTRANT, bytes(""), bytes("")),
            targetChain,
            gasLimit,
            receiverValue,
            targetChain,
            getRegisteredContractAddress(targetChain)
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

    function sendToEvm(
        uint16 targetChain,
        address destination,
        uint32 gasLimit,
        uint16 refundChain,
        address refundAddress,
        uint128 receiverValue,
        uint256 paymentForExtraReceiverValue,
        bytes memory payload,
        MessageKey[] memory messageKeys
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
            messageKeys,
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

    bytes deliveryVaa;

    function deliverReentrant(bytes memory _deliveryVaa) public payable {
        deliveryVaa = _deliveryVaa;
        relayer.deliver{value: msg.value}(
            new bytes[](0), _deliveryVaa, payable(address(this)), bytes("")
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
        } else {
            return;
        }

        messageHistory.push(message.message);

        if (message.version == Version.REENTRANT) {
            relayer.deliver{value: address(this).balance}(
                new bytes[](0), deliveryVaa, payable(address(this)), bytes("")
            );
        }
        if (message.version == Version.SEND_BACK || message.version == Version.MULTI_SEND_BACK) {
            (LocalNative cost,) =
                relayer.quoteEVMDeliveryPrice(sourceChain, TargetNative.wrap(0), Gas.wrap(500_000));
            relayer.forwardToEvm{value: LocalNative.unwrap(cost)}(
                sourceChain,
                getRegisteredContractAddress(sourceChain),
                encodeMessage(Message(Version.SEND, message.forwardMessage, bytes(""))),
                TargetNative.wrap(0),
                LocalNative.wrap(0),
                Gas.wrap(500_000),
                sourceChain,
                getRegisteredContractAddress(sourceChain),
                relayer.getDefaultDeliveryProvider(),
                new VaaKey[](0),
                15
            );
            if (message.version == Version.MULTI_SEND_BACK) {
                (cost,) = relayer.quoteEVMDeliveryPrice(
                    wormhole.chainId(), TargetNative.wrap(0), Gas.wrap(500_000)
                );
                relayer.forwardToEvm{value: LocalNative.unwrap(cost)}(
                    wormhole.chainId(),
                    getRegisteredContractAddress(wormhole.chainId()),
                    encodeMessage(Message(Version.SEND, message.forwardMessage, bytes(""))),
                    TargetNative.wrap(0),
                    LocalNative.wrap(address(this).balance - LocalNative.unwrap(cost)),
                    Gas.wrap(500_000),
                    sourceChain,
                    getRegisteredContractAddress(sourceChain),
                    relayer.getDefaultDeliveryProvider(),
                    new VaaKey[](0),
                    15
                );
            }
            (bool success,) = address(0).call{value: address(this).balance}("");
            require(success, "Failed to send funds");
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
