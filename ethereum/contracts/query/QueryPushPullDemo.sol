// contracts/query/QueryPushPullDemo.sol
// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.0;

import "../libraries/external/BytesLib.sol";
import "../interfaces/IWormhole.sol";
import "./QueryResponse.sol";

error InvalidOwner();
// @dev for the onlyOwner modifier
error InvalidCaller();
error InvalidContractAddress();
error InvalidWormholeAddress();
error InvalidForeignChainID();
error InvalidDestinationChain();
error InvalidFinality();
error UnexpectedCallData();
error AlreadyReceived(bytes32 digest);

/// @dev QueryPushPullDemo is an example of using CCQ for messaging instead of the Core protocol's `publishMessage`
contract QueryPushPullDemo is QueryResponse {
    using BytesLib for bytes;

    struct Message {
        // unique identifier for this message type
        uint8 payloadID;
        // sequence (only used by pull)
        uint64 sequence;
        // destination chain id
        uint16 destinationChainID;
        // arbitrary message string
        string message;
    }

    event pullMessagePublished(uint8 payloadID, uint64 sequence, uint16 destinationChainID, string message);
    event pullMessageReceived(uint16 sourceChainID, uint8 payloadID, uint64 sequence, uint16 destinationChainID, string message);
    event pushMessageReceived(uint16 sourceChainID, uint8 payloadID, uint64 sequence, uint16 destinationChainID, string message);

    address private immutable owner;
    address private immutable wormhole;
    uint16 private immutable myChainID;
    uint64 public sequence;
    mapping(uint16 => bytes32) private chainRegistrations;
    mapping(bytes32 => bool) private coreReceived;
    mapping(bytes32 => bool) private ccqSent;
    mapping(bytes32 => bool) private ccqReceived;

    bytes4 private constant HasSentMessage = bytes4(keccak256("hasSentMessage(bytes32)"));

    constructor(address _owner, address _wormhole, uint16 _myChainID) {
        if (_owner == address(0)) {
            revert InvalidOwner();
        }
        owner = _owner;

        if (_wormhole == address(0)) {
            revert InvalidWormholeAddress();
        }
        wormhole = _wormhole;  
        myChainID = _myChainID;
    }

    // updateRegistration should be used to add the other chains and to set / update contract addresses.
    function updateRegistration(uint16 _chainID, bytes32 _contractAddress) public onlyOwner {
        chainRegistrations[_chainID] = _contractAddress;
    }

    /**
     * @notice Encodes the Message struct into bytes
     * @param parsedMessage Message struct with arbitrary message
     * @return encodedMessage Message encoded into bytes
     */
    function encodeMessage(Message memory parsedMessage) public pure returns (bytes memory encodedMessage) {
        // Convert message string to bytes so that we can use the .length attribute.
        // The length of the arbitrary messages needs to be encoded in the message
        // so that the corresponding decode function can decode the message properly.
        bytes memory encodedMessagePayload = abi.encodePacked(parsedMessage.message);

        // return the encoded message
        encodedMessage = abi.encodePacked(
            parsedMessage.payloadID,
            parsedMessage.sequence,
            parsedMessage.destinationChainID,
            uint16(encodedMessagePayload.length),
            encodedMessagePayload
        );
    }

    /**
     * @notice Decodes bytes into Message struct
     * @dev Verifies the payloadID and chainId
     * @param encodedMessage encoded arbitrary message
     * @return parsedMessage Message struct with arbitrary message
     */
    function decodeMessage(bytes memory encodedMessage) public pure returns (Message memory parsedMessage) {
        // starting index for byte parsing
        uint256 index = 0;

        // parse and verify the payloadID
        parsedMessage.payloadID = encodedMessage.toUint8(index);
        require(parsedMessage.payloadID == 1, "invalid payloadID");
        index += 1;

        // parse the sequence
        parsedMessage.sequence = encodedMessage.toUint64(index);
        index += 8;

        // parse the chainID
        parsedMessage.destinationChainID = encodedMessage.toUint16(index);
        index += 2;

        // parse the message string length
        uint256 messageLength = encodedMessage.toUint16(index);
        index += 2;

        // parse the message string
        bytes memory messageBytes = encodedMessage.slice(index, messageLength);
        parsedMessage.message = string(messageBytes);
        index += messageLength;

        // confirm that the message was the expected length
        require(index == encodedMessage.length, "invalid message length");
    }

    function sendPushMessage(uint16 _destinationChainID, string memory _message) public payable returns (uint64 _sequence) {
        // enforce a max size for the arbitrary message
        require(
            abi.encodePacked(_message).length < type(uint16).max,
            "message too large"
        );

        IWormhole _wormhole = IWormhole(wormhole);
        uint256 wormholeFee = _wormhole.messageFee();

        // Confirm that the caller has sent enough value to pay for the Wormhole message fee.
        require(msg.value == wormholeFee, "insufficient value");

        Message memory parsedMessage = Message({
            payloadID: uint8(1),
            sequence: 0,
            destinationChainID: _destinationChainID,
            message: _message
        });

        // encode the Message struct into bytes
        bytes memory encodedMessage = encodeMessage(parsedMessage);

        // Send the HelloWorld message by calling publishMessage on the
        // Wormhole core contract and paying the Wormhole protocol fee.
        _sequence = _wormhole.publishMessage{value: wormholeFee}(
            0, // nonce
            encodedMessage,
            201 // safe
        );
    }

    function sendPullMessage(uint16 _destinationChainID, string memory _message) public returns (uint64 _sequence) {
        // enforce a max size for the arbitrary message
        require(
            abi.encodePacked(_message).length < type(uint16).max,
            "message too large"
        );

        _sequence = ++sequence;

        Message memory parsedMessage = Message({
            payloadID: uint8(1),
            sequence: _sequence,
            destinationChainID: _destinationChainID,
            message: _message
        });

        // encode the Message struct into bytes
        bytes memory encodedMessage = encodeMessage(parsedMessage);

        // for consistency, match the inbound digest calculation
        bytes32 digest = keccak256(abi.encodePacked(myChainID, bytes32(uint256(uint160(address(this)))), keccak256(encodedMessage)));

        ccqSent[digest] = true;

        emit pullMessagePublished(parsedMessage.payloadID, parsedMessage.sequence, parsedMessage.destinationChainID, parsedMessage.message);
    }

    // hasSentMessage (call signature 8b9369e2) returns true if the given digest matches a message sent by this conract. It is meant to be used in a cross chain query.
    function hasSentMessage(bytes32 digest) public view returns (bool) {
        return ccqSent[digest];
    }

    function hasReceivedMessage(bytes32 digest) public view returns (bool) {
        return ccqReceived[digest];
    }

    function hasReceivedPushMessage(bytes32 digest) public view returns (bool) {
        return coreReceived[digest];
    }

    function receivePushMessage(bytes memory encodedMessage) public {
        // call the Wormhole core contract to parse and verify the encodedMessage
        (
            IWormhole.VM memory wormholeMessage,
            bool valid,
            string memory reason
        ) = IWormhole(wormhole).parseAndVerifyVM(encodedMessage);

        // confirm that the Wormhole core contract verified the message
        require(valid, reason);

        // verify that this message was emitted by a registered emitter
        bytes32 emitterAddress = chainRegistrations[wormholeMessage.emitterChainId];
        if (emitterAddress == bytes32(0)) {
            revert InvalidForeignChainID();
        }
        if (wormholeMessage.emitterAddress != emitterAddress) {
            revert InvalidContractAddress();
        }

        if (coreReceived[wormholeMessage.hash]) {
            revert AlreadyReceived(wormholeMessage.hash);
        }

        coreReceived[wormholeMessage.hash] = true;

        // decode the message payload into the Message struct
        Message memory parsedMessage = decodeMessage(
            wormholeMessage.payload
        );

        if (parsedMessage.destinationChainID != myChainID) {
            revert InvalidDestinationChain();
        }

        emit pushMessageReceived(wormholeMessage.emitterChainId, parsedMessage.payloadID, wormholeMessage.sequence, parsedMessage.destinationChainID, parsedMessage.message);
    }

    // @notice Takes the cross chain query response for any number of "safe" messages from registered contracts, stores the digest for replay protection, and "processes" (logs) the message.
    function receivePullMessages(bytes memory response, IWormhole.Signature[] memory signatures, bytes[] memory messages) public {
        ParsedQueryResponse memory r = parseAndVerifyQueryResponse(address(wormhole), response, signatures);
        uint numResponses = r.responses.length;
        uint messageIndex = 0;
        
        for (uint i=0; i < numResponses;) {
            uint16 requestChainID = r.responses[i].chainId;
            address foreignContract = _truncateAddress(chainRegistrations[requestChainID]);
            if (foreignContract == address(0)) {
                revert InvalidForeignChainID();
            }

            EthCallWithFinalityQueryResponse memory eqr = parseEthCallWithFinalityQueryResponse(r.responses[i]);

            if (eqr.requestFinality.length != 4 || keccak256(eqr.requestFinality) != keccak256("safe")) {
                revert InvalidFinality();
            }

            uint numCalls = eqr.result.length;
            for (uint resultIdx=0; resultIdx < numCalls;) {

                if (eqr.result[resultIdx].contractAddress != foreignContract) {
                    revert InvalidContractAddress();
                }

                // add the chain id and contract to form a unique digest
                bytes32 digest = keccak256(abi.encodePacked(requestChainID, chainRegistrations[requestChainID], keccak256(messages[messageIndex])));

                if (ccqReceived[digest]) {
                    // This could also just skip 
                    revert AlreadyReceived(digest);
                }

                // 36 bytes for abi.encodeWithSelector(HasSentMessage, digest)
                require(eqr.result[resultIdx].callData.length == 36, "invalid callData length");

                // this works the first time but not the second within this loop
                // if (!eqr.result[resultIdx].callData.equal(abi.encodeWithSelector(HasSentMessage, digest))) {
                //     revert UnexpectedCallData();
                // }

                // this works
                if (keccak256(eqr.result[resultIdx].callData) != keccak256(abi.encodeWithSelector(HasSentMessage, digest))) {
                    revert UnexpectedCallData();
                }

                require(eqr.result[resultIdx].result.length == 32, "result is not a bool");

                bool wasSent = abi.decode(eqr.result[resultIdx].result, (bool));
                require(wasSent, "result is not true");

                ccqReceived[digest] = true;

                Message memory parsedMessage = decodeMessage(
                    messages[messageIndex]
                );

                if (parsedMessage.destinationChainID != myChainID) {
                    revert InvalidDestinationChain();
                }

                emit pullMessageReceived(requestChainID, parsedMessage.payloadID, parsedMessage.sequence, parsedMessage.destinationChainID, parsedMessage.message);

                unchecked {
                    ++resultIdx;
                    ++messageIndex;
                }
            }

            unchecked {
                ++i;
            }
        }
    }

    /*
     * @dev Truncate a 32 byte array to a 20 byte address.
     *      Reverts if the array contains non-0 bytes in the first 12 bytes.
     *
     * @param bytes32 bytes The 32 byte array to be converted.
     */
    function _truncateAddress(bytes32 b) internal pure returns (address) {
        require(bytes12(b) == 0, "invalid EVM address");
        return address(uint160(uint256(b)));
    }

    modifier onlyOwner() {
        if (owner != msg.sender) {
            revert InvalidOwner();
        }
        _;
    }
}
