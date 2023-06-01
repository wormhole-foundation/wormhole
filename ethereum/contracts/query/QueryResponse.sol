// contracts/query/QueryResponse.sol
// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.0;

import "../libraries/external/BytesLib.sol";
import "../interfaces/IWormhole.sol";

// TODO: move functions to library
contract QueryResponse {
    using BytesLib for bytes;
    struct EthCallResponse {
        // Sender
        uint16  senderChainId;
        bytes   requestId; // 65 byte sig for off-chain, 32 byte vaaHash for on-chain
        // Request
        uint8   requestType;
        uint16  requestChainId;
        uint32  requestNonce;
        address requestTo;
        bytes   requestData;
        bytes   requestBlock;
        // Response
        uint64  blockNumber;
        bytes32 blockHash;
        uint32  blockTime;
        bytes   result;
    }

    bytes public constant responsePrefix = bytes("query_response_0000000000000000000|");
    IWormhole public immutable wormhole;

    constructor (address _wormhole) {
        wormhole = IWormhole(_wormhole);
    }

    function getResponseHash(bytes memory response) public pure returns (bytes32) {
        return keccak256(response);
    }

    function getResponseDigest(bytes memory response) public pure returns (bytes32) {
        return keccak256(abi.encodePacked(responsePrefix,getResponseHash(response)));
    }

    /**
     * @dev verifyQueryResponse serves to 
     * IWormhole.Signature expects the last byte to be bumped by 27 
     * see https://github.com/wormhole-foundation/wormhole/blob/637b1ee657de7de05f783cbb2078dd7d8bfda4d0/ethereum/contracts/Messages.sol#L174
     */
    function verifyQueryResponse(bytes memory response, IWormhole.Signature[] memory signatures) public view {
        // TODO: make a verifyCurrentQuorum call on the core bridge so that there is only 1 cross call instead of 4
        uint32 gsi = wormhole.getCurrentGuardianSetIndex();
        IWormhole.GuardianSet memory guardianSet = wormhole.getGuardianSet(gsi);

        bytes32 responseHash = getResponseDigest(response);

       /**
        * @dev Checks whether the guardianSet has zero keys
        * WARNING: This keys check is critical to ensure the guardianSet has keys present AND to ensure
        * that guardianSet key size doesn't fall to zero and negatively impact quorum assessment.  If guardianSet
        * key length is 0 and vm.signatures length is 0, this could compromise the integrity of both vm and
        * signature verification.
        */
        if(guardianSet.keys.length == 0){
            revert("invalid guardian set");
        }

       /**
        * @dev We're using a fixed point number transformation with 1 decimal to deal with rounding.
        *   WARNING: This quorum check is critical to assessing whether we have enough Guardian signatures to validate a VM
        *   if making any changes to this, obtain additional peer review. If guardianSet key length is 0 and
        *   vm.signatures length is 0, this could compromise the integrity of both vm and signature verification.
        */
        if (signatures.length < wormhole.quorum(guardianSet.keys.length)){
            revert("no quorum");
        }

        /// @dev Verify the proposed vm.signatures against the guardianSet
        (bool signaturesValid, string memory invalidReason) = wormhole.verifySignatures(responseHash, signatures, guardianSet);
        if(!signaturesValid){
            revert(invalidReason);
        }

        /// If we are here, we've validated the VM is a valid multi-sig that matches the current guardianSet.
    }

    function parseEthCallResponse(bytes memory response) internal pure returns (EthCallResponse memory r) {
        uint index = 0;

        r.senderChainId = response.toUint16(index);
        index += 2;

        if (r.senderChainId == 0) {
            r.requestId = response.slice(index, 65);
            index += 65;
        } else {
            r.requestId = response.slice(index, 32);
            index += 32;
        }

        r.requestType = response.toUint8(index);
        index += 1;

        require(r.requestType == 1, "invalid request type");

        r.requestChainId = response.toUint16(index);
        index += 2;

        r.requestNonce = response.toUint32(index);
        index += 4;

        r.requestTo = response.toAddress(index);
        index += 20;

        uint32 len = response.toUint32(index);
        index += 4;
        r.requestData = response.slice(index, len);
        index += len;

        len = response.toUint32(index);
        index += 4;
        r.requestBlock = response.slice(index, len);
        index += len;

        r.blockNumber = response.toUint64(index);
        index += 8;

        r.blockHash = response.toBytes32(index);
        index += 32;

        r.blockTime = response.toUint32(index);
        index += 4;

        len = response.toUint32(index);
        index += 4;
        r.result = response.slice(index, len);
        index += len;

        require(response.length == index, "invalid response");
    }

    function processStringResult(bytes memory response, IWormhole.Signature[] memory signatures) public view returns (string memory result) {
        verifyQueryResponse(response, signatures);
        EthCallResponse memory parsed = parseEthCallResponse(response);
        // Polygon
        require(parsed.requestChainId == 5, "invalid request chain");
        // WMATIC
        require(parsed.requestTo == 0x0d500B1d8E8eF31E21C99d1Db9A6444d3ADf1270, "invalid request to");
        // Name
        require(parsed.requestData.equal(abi.encodeWithSignature("name()")), "invalid request data");
        (result) = abi.decode(parsed.result, (string));
    }
}
