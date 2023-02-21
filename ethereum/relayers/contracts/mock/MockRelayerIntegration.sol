// contracts/mock/MockBatchedVAASender.sol
// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.0;

import "../libraries/external/BytesLib.sol";
import "../interfaces/IWormhole.sol";
import "../interfaces/IWormholeRelayer.sol";
import "../interfaces/IWormholeReceiver.sol";

import "forge-std/console.sol";

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

    // mapping of other MockRelayerIntegration contracts
    mapping(uint16 => bytes32) registeredContracts;

    bytes[] messages;

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

    function sendMessage(bytes memory _message, uint16 targetChainId, address destination) public payable {
        sendMessageGeneral(_message, targetChainId, destination, destination, 0, 1);
    }

    function sendMessageWithRefundAddress(
        bytes memory _message,
        uint16 targetChainId,
        address destination,
        address refundAddress
    ) public payable {
        sendMessageGeneral(_message, targetChainId, destination, refundAddress, 0, 1);
    }

    function sendMessageWithForwardedResponse(
        bytes memory _message,
        uint16 targetChainId,
        address destination,
        address refundAddress
    ) public payable {
        uint16[] memory chains = new uint16[](1);
        chains[0] = wormhole.chainId();
        uint32[] memory gasLimits = new uint32[](1);
        gasLimits[0] = 1000000;
        bytes[] memory newMessages = new bytes[](2);
        newMessages[0] = bytes("received!");
        newMessages[1] = abi.encodePacked(uint8(0));
        FurtherInstructions memory instructions =
            FurtherInstructions({keepSending: true, newMessages: newMessages, chains: chains, gasLimits: gasLimits});
        wormhole.publishMessage{value: wormhole.messageFee()}(1, _message, 200);
        wormhole.publishMessage{value: wormhole.messageFee()}(1, encodeFurtherInstructions(instructions), 200);
        executeSend(targetChainId, destination, refundAddress, 0, 1);
    }

    function sendMessageGeneral(
        bytes memory fullMessage,
        uint16 targetChainId,
        address destination,
        address refundAddress,
        uint256 receiverValue,
        uint32 nonce
    ) public payable {
        wormhole.publishMessage{value: wormhole.messageFee()}(nonce, fullMessage, 200);
        wormhole.publishMessage{value: wormhole.messageFee()}(nonce, abi.encodePacked(uint8(0)), 200);
        executeSend(targetChainId, destination, refundAddress, receiverValue, nonce);
    }

    function sendMessagesWithFurtherInstructions(
        bytes[] memory messages,
        FurtherInstructions memory furtherInstructions,
        uint16[] memory chains,
        uint256[] memory computeBudgets
    ) public payable {
        for (uint16 i = 0; i < messages.length; i++) {
            wormhole.publishMessage{value: wormhole.messageFee()}(1, messages[i], 200);
        }
        wormhole.publishMessage{value: wormhole.messageFee()}(1, encodeFurtherInstructions(furtherInstructions), 200);
        IWormholeRelayer.Send[] memory requests = new IWormholeRelayer.Send[](chains.length);
        for (uint16 i = 0; i < chains.length; i++) {
            requests[i] = IWormholeRelayer.Send({
                targetChain: chains[i],
                targetAddress: registeredContracts[chains[i]],
                refundAddress: registeredContracts[chains[i]],
                maxTransactionFee: computeBudgets[i],
                receiverValue: 0,
                relayParameters: relayer.getDefaultRelayParams()
            });
        }
        IWormholeRelayer.MultichainSend memory container = IWormholeRelayer.MultichainSend({
            requests: requests,
            relayProviderAddress: relayer.getDefaultRelayProvider()
        });
        relayer.multichainSend{value: (msg.value - wormhole.messageFee() * (1 + messages.length))}(container, 1);
    }

    function executeSend(
        uint16 targetChainId,
        address destination,
        address refundAddress,
        uint256 receiverValue,
        uint32 nonce
    ) internal {
        IWormholeRelayer.Send memory request = IWormholeRelayer.Send({
            targetChain: targetChainId,
            targetAddress: relayer.toWormholeFormat(address(destination)),
            refundAddress: relayer.toWormholeFormat(address(refundAddress)), // This will be ignored on the target chain if the intent is to perform a forward
            maxTransactionFee: msg.value - 3 * wormhole.messageFee() - receiverValue,
            receiverValue: receiverValue,
            relayParameters: relayer.getDefaultRelayParams()
        });

        relayer.send{value: msg.value - 2 * wormhole.messageFee()}(request, nonce, relayer.getDefaultRelayProvider());
    }

    function receiveWormholeMessages(bytes[] memory wormholeObservations, bytes[] memory) public payable override {
        // loop through the array of wormhole observations from the batch and store each payload
        uint256 numObservations = wormholeObservations.length;
        messages = new bytes[](wormholeObservations.length - 2);
        for (uint256 i = 0; i < numObservations - 2; i++) {
            (IWormhole.VM memory parsed, bool valid, string memory reason) =
                wormhole.parseAndVerifyVM(wormholeObservations[i]);
            require(valid, reason);
            require(registeredContracts[parsed.emitterChainId] == parsed.emitterAddress);
            messages[i] = parsed.payload;
        }

        (IWormhole.VM memory parsed, bool valid, string memory reason) =
            wormhole.parseAndVerifyVM(wormholeObservations[wormholeObservations.length - 2]);
        FurtherInstructions memory instructions = decodeFurtherInstructions(parsed.payload);

        if (instructions.keepSending) {
            for (uint16 i = 0; i < instructions.newMessages.length; i++) {
                wormhole.publishMessage{value: wormhole.messageFee()}(parsed.nonce, instructions.newMessages[i], 200);
            }
            IWormholeRelayer.Send[] memory sendRequests = new IWormholeRelayer.Send[](instructions.chains.length);
            for (uint16 i = 0; i < instructions.chains.length; i++) {
                sendRequests[i] = IWormholeRelayer.Send({
                    targetChain: instructions.chains[i],
                    targetAddress: registeredContracts[instructions.chains[i]],
                    refundAddress: registeredContracts[instructions.chains[i]],
                    maxTransactionFee: relayer.quoteGas(
                        instructions.chains[i], instructions.gasLimits[i], relayer.getDefaultRelayProvider()
                        ),
                    receiverValue: 0,
                    relayParameters: relayer.getDefaultRelayParams()
                });
            }
            IWormholeRelayer.MultichainSend memory container = IWormholeRelayer.MultichainSend({
                requests: sendRequests,
                relayProviderAddress: relayer.getDefaultRelayProvider()
            });

            relayer.multichainForward(container, sendRequests[0].targetChain, parsed.nonce);
        }
    }

    function getPayload(bytes32 hash) public view returns (bytes memory) {
        return verifiedPayloads[hash];
    }

    function getMessage() public view returns (bytes memory) {
        if (messages.length == 0) {
            return new bytes(0);
        }
        return messages[0];
    }

    function getMessages() public view returns (bytes[] memory) {
        return messages;
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

    function registerEmitter(uint16 chainId, bytes32 emitterAddress) public {
        require(msg.sender == owner);
        registeredContracts[chainId] = emitterAddress;
    }

    function encodeFurtherInstructions(FurtherInstructions memory furtherInstructions)
        public
        view
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
        view
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
