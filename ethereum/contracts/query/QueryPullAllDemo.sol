// contracts/query/QueryPullAllDemo.sol
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
error InvalidResultHash(); //0x2687977f
error UnexpectedResultsLen();
error UnexpectedCallData(); //0x647a798b
error AlreadyRegistered();

/// @dev QueryPullAllDemo is an example of using CCQ for messaging instead of the Core protocol's `publishMessage`
/// This contract differs from QueryPushPullDemo in that it requires each message to be redeemed in order based on finalized CCQ results
/// Instead of storing the digest of every message sent, only the hash(previousHash, digest) is stored per destination chain
/// Then, redemption hashes are similarly tracked, one per chain (expecting emitting addresses never to change)
/// Primarily, this is an experiment in maximally reducing gas costs for frequently messaging applications with a periodic, dedicated relay service
/// where every message is expected to be redeemed in a timely manner
/// TODO: should this be extended to include chain and receiving/sending contract in the mappings
contract QueryPullAllDemo is QueryResponse {
    using BytesLib for bytes;

    struct Message {
        // unique identifier for this message type
        uint8 payloadID;
        // destination chain id
        uint16 destinationChainID;
        // arbitrary message string
        string message;
    }

    event pullMessagePublished(bytes32 previousHash, bytes32 latestHash, uint16 sourceChainID, uint8 payloadID, uint16 destinationChainID, string message);
    event pullMessageReceived(bytes32 previousHash, bytes32 latestHash, uint16 sourceChainID, uint8 payloadID, uint16 destinationChainID, string message);

    address private immutable owner;
    address private immutable wormhole;
    uint16 private immutable myChainID;
    mapping(uint16 => bytes32) public chainRegistrations;
    mapping(uint16 => bytes32) private latestSentTo;
    mapping(uint16 => bytes32) private latestReceivedFrom;

    bytes4 private constant LatestSentMessage = bytes4(keccak256("latestSentMessage(uint16)"));

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

    // updateRegistration should be used to add the other chains and to set contract addresses.
    function updateRegistration(uint16 _chainID, bytes32 _contractAddress) public onlyOwner {
        if (chainRegistrations[_chainID] != bytes32(0)) {
            revert AlreadyRegistered();
        }
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

    function sendPullMessage(uint16 _destinationChainID, string memory _message) public returns (bytes32) {
        // enforce a max size for the arbitrary message
        require(
            abi.encodePacked(_message).length < type(uint16).max,
            "message too large"
        );

        Message memory parsedMessage = Message({
            payloadID: uint8(1),
            destinationChainID: _destinationChainID,
            message: _message
        });

        // encode the Message struct into bytes
        bytes memory encodedMessage = encodeMessage(parsedMessage);

        bytes32 _previousHash = latestSentTo[_destinationChainID];
        latestSentTo[_destinationChainID] = keccak256(abi.encodePacked(_previousHash, keccak256(encodedMessage)));

        emit pullMessagePublished(_previousHash, latestSentTo[_destinationChainID], myChainID, parsedMessage.payloadID, parsedMessage.destinationChainID, parsedMessage.message);

        return latestSentTo[_destinationChainID];
    }

    // latestSentMessage (call signature 0xce68d748) returns the last digest sent by this conract to the given chain. It is meant to be used in a cross chain query.
    function latestSentMessage(uint16 _destinationChainID) public view returns (bytes32) {
        return latestSentTo[_destinationChainID];
    }

    function lastReceivedMessage(uint16 _sourceChainId) public view returns (bytes32) {
        return latestReceivedFrom[_sourceChainId];
    }

    // @notice Takes the cross chain query response for any number of "finalized" messages from registered contracts, 
    // "processes" (logs) the message, 
    // and ensures that the resulting hash(previousHash, digest) matches the query
    // This expects one response per chain, and the messages to be grouped in order of chain
    // Hashing the messages for a chain from lastReceivedMessage[_sourceChainId] should result in <sourceChain's> latestSentTo[_destinationChainID]
    function receivePullMessages(bytes memory response, IWormhole.Signature[] memory signatures, bytes[] memory messages) public {
        ParsedQueryResponse memory r = parseAndVerifyQueryResponse(address(wormhole), response, signatures);
        uint numResponses = r.responses.length;
        uint numMessages = messages.length;
        uint messageIndex = 0;
        
        for (uint i=0; i < numResponses;) {
            uint16 requestChainID = r.responses[i].chainId;
            address foreignContract = _truncateAddress(chainRegistrations[requestChainID]);
            if (foreignContract == address(0)) {
                revert InvalidForeignChainID();
            }

            EthCallWithFinalityQueryResponse memory eqr = parseEthCallWithFinalityQueryResponse(r.responses[i]);

            if (eqr.requestFinality.length != 9 || keccak256(eqr.requestFinality) != keccak256("finalized")) {
                revert InvalidFinality();
            }

            if (eqr.result.length != 1) {
                revert UnexpectedResultsLen();
            }

            if (eqr.result[0].contractAddress != foreignContract) {
                revert InvalidContractAddress();
            }

            // 36 bytes for abi.encodeWithSelector(LatestSentMessage, chainId)
            require(eqr.result[0].callData.length == 36, "invalid callData length");

            if (keccak256(eqr.result[0].callData) != keccak256(abi.encodeWithSelector(LatestSentMessage, myChainID))) {
                revert UnexpectedCallData();
            }

            require(eqr.result[0].result.length == 32, "result is not a bytes32");

            bytes32 _targetHash = abi.decode(eqr.result[0].result, (bytes32));
            bytes32 _previousHash = latestReceivedFrom[requestChainID];

            while (messageIndex < numMessages && _previousHash != _targetHash) {

                bytes32 _newHash = keccak256(abi.encodePacked(_previousHash, keccak256(messages[messageIndex])));

                // avoid stack too deep
                {
                    Message memory parsedMessage = decodeMessage(
                        messages[messageIndex]
                    );

                    if (parsedMessage.destinationChainID != myChainID) {
                        revert InvalidDestinationChain();
                    }
                    emit pullMessageReceived(_previousHash, _newHash, requestChainID, parsedMessage.payloadID, parsedMessage.destinationChainID, parsedMessage.message);
                }

                _previousHash = _newHash;

                unchecked {
                    ++messageIndex;
                }
            }

            if (_previousHash != _targetHash) {
                revert InvalidResultHash();
            }

            latestReceivedFrom[requestChainID] = _previousHash;

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
