// contracts/mock/MockBatchedVAASender.sol
// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.0;

import "../libraries/external/BytesLib.sol";
import "../interfaces/IWormhole.sol";
import "../interfaces/relayer/IWormholeRelayer.sol";
import "../interfaces/relayer/IWormholeReceiver.sol";

interface Structs {
    struct XAddress {
        uint16 chainId;
        bytes32 addr;
    }
}

contract MockRelayerIntegration is IWormholeReceiver {
    using BytesLib for bytes;

    // wormhole instance on this chain
    IWormhole immutable wormhole;

    // trusted relayer contract on this chain
    IWormholeRelayer immutable relayer;

    // deployer of this contract
    address immutable owner;

    // map that stores payloads from received VAAs
    mapping(bytes32 => bytes) verifiedPayloads;

    // latest delivery data
    DeliveryData latestDeliveryData;

    // mapping of other MockRelayerIntegration contracts
    mapping(uint16 => bytes32) registeredContracts;

    // bytes[] messages;
    bytes[][] messageHistory;

    struct FurtherInstructions {
        bool keepSending;
        bytes[] newMessages;
        uint16[] chains;
        uint32[] gasLimits;
    }

    constructor(address _wormholeCore, address _coreRelayer) {
        wormhole = IWormhole(_wormholeCore);
        relayer = IWormholeRelayer(_coreRelayer);
        owner = msg.sender;
    }

    function sendMessage(bytes memory _message, uint16 targetChainId, address destination)
        public
        payable
        returns (uint64 sequence)
    {
        sequence = sendMessageGeneral(_message, targetChainId, destination, targetChainId, destination, 0, bytes(""));
    }

    function sendMessageWithPayload(bytes memory _message, uint16 targetChainId, address destination, bytes memory payload)
        public
        payable
        returns (uint64 sequence)
    {
        sequence = sendMessageGeneral(_message, targetChainId, destination, targetChainId, destination, 0, payload);
    }

    function sendOnlyPayload(bytes memory payload, uint16 targetChainId, address destination) public payable returns (uint64 sequence) {
        sequence = relayer.send{value: msg.value}(
            targetChainId, relayer.toWormholeFormat(destination), targetChainId, relayer.toWormholeFormat(destination), msg.value - wormhole.messageFee(), 0, payload);
    }


    function sendMessageWithRefundAddress(
        bytes memory _message,
        uint16 targetChainId,
        address destination,
        address refundAddress,
        bytes memory payload
    ) public payable returns (uint64 sequence) {
        sequence = sendMessageGeneral(_message, targetChainId, destination, targetChainId, refundAddress, 0, payload);
    }

    function vaaKeysCreator(uint64 sequence1, uint64 sequence2)
        internal view
        returns (IWormholeRelayer.VaaKey[] memory vaaKeys)
    {
        vaaKeys = new IWormholeRelayer.VaaKey[](2);
        vaaKeys[0] = IWormholeRelayer.VaaKey(
            IWormholeRelayer.VaaKeyType.EMITTER_SEQUENCE,
            wormhole.chainId(),
            relayer.toWormholeFormat(address(this)),
            sequence1,
            bytes32(0x0)
        );
        vaaKeys[1] = IWormholeRelayer.VaaKey(
            IWormholeRelayer.VaaKeyType.EMITTER_SEQUENCE,
            wormhole.chainId(),
            relayer.toWormholeFormat(address(this)),
            sequence2,
            bytes32(0x0)
        );
    }

    function sendMessageWithForwardedResponse(
        bytes memory _message,
        uint16 targetChainId,
        address destination,
        address refundAddress
    ) public payable returns (uint64 sequence) {
        uint16[] memory chains = new uint16[](1);
        chains[0] = wormhole.chainId();
        uint32[] memory gasLimits = new uint32[](1);
        gasLimits[0] = 1000000;
        bytes[] memory newMessages = new bytes[](2);
        newMessages[0] = bytes("received!");
        newMessages[1] = abi.encodePacked(uint8(0));
        FurtherInstructions memory instructions =
            FurtherInstructions({keepSending: true, newMessages: newMessages, chains: chains, gasLimits: gasLimits});
        uint64 sequence0 = wormhole.publishMessage{value: wormhole.messageFee()}(0, _message, 200);
        uint64 sequence1 =
            wormhole.publishMessage{value: wormhole.messageFee()}(0, encodeFurtherInstructions(instructions), 200);
        sequence = executeSend(
            targetChainId, destination, targetChainId, refundAddress, 0, bytes(""), vaaKeysCreator(sequence0, sequence1)
        );
    }

    function sendMessageGeneral(
        bytes memory fullMessage,
        uint16 targetChainId,
        address destination,
        uint16 refundChain,
        address refundAddress,
        uint256 receiverValue,
        bytes memory payload
    ) public payable returns (uint64 sequence) {
        uint64 sequence0 = wormhole.publishMessage{value: wormhole.messageFee()}(0, fullMessage, 200);
        uint64 sequence1 = wormhole.publishMessage{value: wormhole.messageFee()}(0, abi.encodePacked(uint8(0)), 200);
        sequence = executeSend(
            targetChainId,
            destination,
            refundChain,
            refundAddress,
            receiverValue,
            payload,
            vaaKeysCreator(sequence0, sequence1)
        );
    }

    function sendMessagesWithFurtherInstructions(
        bytes[] memory messages,
        FurtherInstructions memory furtherInstructions,
        uint16[] memory chains,
        uint256[] memory computeBudgets
    ) public payable returns (uint64 sequence) {
        IWormholeRelayer.VaaKey[] memory vaaKeys = new IWormholeRelayer.VaaKey[](messages.length + 1);
        for (uint16 i = 0; i < messages.length; i++) {
            sequence = wormhole.publishMessage{value: wormhole.messageFee()}(0, messages[i], 200);
            vaaKeys[i] = IWormholeRelayer.VaaKey(
                IWormholeRelayer.VaaKeyType.EMITTER_SEQUENCE,
                wormhole.chainId(),
                relayer.toWormholeFormat(address(this)),
                sequence,
                bytes32(0x0)
            );
        }
        uint64 lastSequence = wormhole.publishMessage{value: wormhole.messageFee()}(
            0, encodeFurtherInstructions(furtherInstructions), 200
        );
        vaaKeys[messages.length] = IWormholeRelayer.VaaKey(
            IWormholeRelayer.VaaKeyType.EMITTER_SEQUENCE,
            wormhole.chainId(),
            relayer.toWormholeFormat(address(this)),
            lastSequence,
            bytes32(0x0)
        );
        IWormholeRelayer.Send[] memory requests = new IWormholeRelayer.Send[](chains.length);
        for (uint16 i = 0; i < chains.length; i++) {
            requests[i] = IWormholeRelayer.Send({
                targetChain: chains[i],
                targetAddress: registeredContracts[chains[i]],
                refundChain: chains[i],
                refundAddress: registeredContracts[chains[i]],
                maxTransactionFee: computeBudgets[i],
                receiverValue: 0,
                payload: bytes(""),
                relayParameters: relayer.getDefaultRelayParams()
            });
        }
        IWormholeRelayer.MultichainSend memory container = IWormholeRelayer.MultichainSend({
            requests: requests,
            consistencyLevel: 200,
            relayProviderAddress: relayer.getDefaultRelayProvider(),
            vaaKeys: vaaKeys
        });
        sequence = relayer.multichainSend{value: (msg.value - wormhole.messageFee() * (1 + messages.length))}(container);
    }

    function executeSend(
        uint16 targetChainId,
        address destination,
        uint16 refundChainId,
        address refundAddress,
        uint256 receiverValue,
        bytes memory payload,
        IWormholeRelayer.VaaKey[] memory vaaKeys
    ) internal returns (uint64 sequence) {

        IWormholeRelayer.Send memory request = IWormholeRelayer.Send({
            targetChain: targetChainId,
            targetAddress: relayer.toWormholeFormat(address(destination)),
            refundChain: refundChainId,
            refundAddress: relayer.toWormholeFormat(address(refundAddress)), // This will be ignored on the target chain if the intent is to perform a forward
            maxTransactionFee: msg.value - 3 * wormhole.messageFee() - receiverValue,
            receiverValue: receiverValue,
            payload: payload,
            relayParameters: relayer.getDefaultRelayParams()
        });

        sequence = relayer.send{value: msg.value - 2 * wormhole.messageFee()}(
            request, vaaKeys, relayer.getDefaultRelayProvider(), 200
        );
    }

    function receiveWormholeMessages(
        IWormholeReceiver.DeliveryData memory deliveryData,
        bytes[] memory wormholeObservations
    ) public payable override {
        // loop through the array of wormhole observations from the batch and store each payload
        latestDeliveryData = deliveryData;
        uint256 numObservations = wormholeObservations.length;
        if(numObservations == 0) return;
        bytes[] memory messages = new bytes[](numObservations - 1);
        uint16 emitterChainId;
        for (uint256 i = 0; i < numObservations - 1; i++) {
            (IWormhole.VM memory parsed_, bool valid_, string memory reason_) =
                wormhole.parseAndVerifyVM(wormholeObservations[i]);
            require(valid_, reason_);
            //require(registeredContracts[parsed.emitterChainId] == parsed.emitterAddress, "Emitter address not valid");
            emitterChainId = parsed_.emitterChainId;
            messages[i] = parsed_.payload;
        }
        messageHistory.push(messages);

        (IWormhole.VM memory parsed, bool valid, string memory reason) =
            wormhole.parseAndVerifyVM(wormholeObservations[numObservations - 1]);
        require(valid, reason);
        FurtherInstructions memory instructions = decodeFurtherInstructions(parsed.payload);
        if (instructions.keepSending) {
            IWormholeRelayer.VaaKey[] memory vaaKeys =
                new IWormholeRelayer.VaaKey[](instructions.newMessages.length);
            for (uint16 i = 0; i < instructions.newMessages.length; i++) {
                uint64 sequence = wormhole.publishMessage{value: wormhole.messageFee()}(
                    parsed.nonce, instructions.newMessages[i], 200
                );
                vaaKeys[i] = IWormholeRelayer.VaaKey(
                    IWormholeRelayer.VaaKeyType.EMITTER_SEQUENCE,
                    wormhole.chainId(),
                    relayer.toWormholeFormat(address(this)),
                    sequence,
                    bytes32(0x0)
                );
            }
            IWormholeRelayer.Send[] memory sendRequests = new IWormholeRelayer.Send[](instructions.chains.length);
            for (uint16 i = 0; i < instructions.chains.length; i++) {
                bytes memory emptyArray;
                sendRequests[i] = IWormholeRelayer.Send({
                    targetChain: instructions.chains[i],
                    targetAddress: registeredContracts[instructions.chains[i]],
                    refundChain: instructions.chains[i],
                    refundAddress: registeredContracts[instructions.chains[i]],
                    maxTransactionFee: relayer.quoteGas(
                        instructions.chains[i], instructions.gasLimits[i], relayer.getDefaultRelayProvider()
                        ),
                    receiverValue: 0,
                    payload: emptyArray,
                    relayParameters: relayer.getDefaultRelayParams()
                });
            }
            IWormholeRelayer.MultichainSend memory container = IWormholeRelayer.MultichainSend({
                requests: sendRequests,
                consistencyLevel: 200,
                relayProviderAddress: relayer.getDefaultRelayProvider(),
                vaaKeys: vaaKeys
            });

            relayer.multichainForward(container);
        }
    }

    function getPayload(bytes32 hash) public view returns (bytes memory) {
        return verifiedPayloads[hash];
    }

    function getMessage() public view returns (bytes memory) {
        if (messageHistory.length == 0 || messageHistory[messageHistory.length - 1].length == 0) {
            return new bytes(0);
        }
        return messageHistory[messageHistory.length - 1][0];
    }

    function getMessages() public view returns (bytes[] memory) {
        if (messageHistory.length == 0 || messageHistory[messageHistory.length - 1].length == 0) {
            return new bytes[](0);
        }
        return messageHistory[messageHistory.length - 1];
    }

    function getDeliveryData() public view returns (DeliveryData memory deliveryData){
        deliveryData = latestDeliveryData;
    }

    function getMessageHistory() public view returns (bytes[][] memory) {
        return messageHistory;
    }

    function clearPayload(bytes32 hash) public {
        delete verifiedPayloads[hash];
    }

    function parseWormholeObservation(bytes memory encoded) public view returns (IWormhole.VM memory) {
        return wormhole.parseVM(encoded);
    }

    function emitterAddress() public view returns (bytes32) {
        return bytes32(uint256(uint160(address(this))));
    }

    function registerEmitter(uint16 chainId, bytes32 emitterAddress_) public {
        require(msg.sender == owner);
        registeredContracts[chainId] = emitterAddress_;
    }

    function registerEmitters(Structs.XAddress[] calldata emitters) public {
        require(msg.sender == owner);
        for (uint256 i = 0; i < emitters.length; i++) {
            registeredContracts[emitters[i].chainId] = emitters[i].addr;
        }
    }

    function getRegisteredContract(uint16 chainId) public view returns (bytes32) {
        return registeredContracts[chainId];
    }

    function encodeFurtherInstructions(FurtherInstructions memory furtherInstructions)
        public
        pure
        returns (bytes memory encodedFurtherInstructions)
    {
        encodedFurtherInstructions = abi.encodePacked(
            furtherInstructions.keepSending ? uint8(1) : uint8(0), uint16(furtherInstructions.newMessages.length)
        );
        for (uint16 i = 0; i < furtherInstructions.newMessages.length; i++) {
            encodedFurtherInstructions = abi.encodePacked(
                encodedFurtherInstructions,
                uint16(furtherInstructions.newMessages[i].length),
                furtherInstructions.newMessages[i]
            );
        }
        encodedFurtherInstructions =
            abi.encodePacked(encodedFurtherInstructions, uint16(furtherInstructions.chains.length));
        for (uint16 i = 0; i < furtherInstructions.chains.length; i++) {
            encodedFurtherInstructions = abi.encodePacked(
                encodedFurtherInstructions, furtherInstructions.chains[i], furtherInstructions.gasLimits[i]
            );
        }
    }

    function decodeFurtherInstructions(bytes memory encodedFurtherInstructions)
        public
        pure
        returns (FurtherInstructions memory furtherInstructions)
    {
        uint256 index = 0;

        furtherInstructions.keepSending = encodedFurtherInstructions.toUint8(index) == 1;
        index += 1;

        if (!furtherInstructions.keepSending) {
            return furtherInstructions;
        }

        uint16 length = encodedFurtherInstructions.toUint16(index);
        index += 2;
        furtherInstructions.newMessages = new bytes[](length);
        for (uint16 i = 0; i < length; i++) {
            uint16 msgLength = encodedFurtherInstructions.toUint16(index);
            index += 2;
            furtherInstructions.newMessages[i] = encodedFurtherInstructions.slice(index, msgLength);
            index += msgLength;
        }

        length = encodedFurtherInstructions.toUint16(index);
        index += 2;
        uint16[] memory chains = new uint16[](length);
        uint32[] memory gasLimits = new uint32[](length);
        for (uint16 i = 0; i < length; i++) {
            chains[i] = encodedFurtherInstructions.toUint16(index);
            index += 2;
            gasLimits[i] = encodedFurtherInstructions.toUint32(index);
            index += 4;
        }
        furtherInstructions.chains = chains;
        furtherInstructions.gasLimits = gasLimits;
    }

    receive() external payable {}
}
