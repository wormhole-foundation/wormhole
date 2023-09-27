// contracts/query/QueryResponse.sol
// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.0;

import "../libraries/external/BytesLib.sol";
import "../interfaces/IWormhole.sol";

/// @dev QueryResponse is a library that implements the parsing and verification of Cross Chain Query (CCQ) responses.
abstract contract QueryResponse {
    using BytesLib for bytes;

    // Custom errors
    error InvalidResponseVersion();
    error VersionMismatch();
    error NumberOfResponsesMismatch();
    error ChainIdMismatch();
    error RequestTypeMismatch();
    error UnsupportedQueryType();
    error UnexpectedNumberOfResults();
       
    /// @dev ParsedQueryResponse is returned by parseAndVerifyQueryResponse().
    struct ParsedQueryResponse {
        uint8   version;
        uint16  senderChainId;
        bytes   requestId; // 65 byte sig for off-chain, 32 byte vaaHash for on-chain
        uint32  nonce;
        ParsedPerChainQueryResponse [] responses;
    }

    /// @dev ParsedPerChainQueryResponse describes a single per-chain response.
    struct ParsedPerChainQueryResponse {
        uint16 chainId;
        uint8 queryType;
        bytes request;
        bytes response;
    }

    /// @dev EthCallQueryResponse describes an ETH call per-chain query.
    struct EthCallQueryResponse {
        bytes requestBlockId;
        uint64 blockNum;
        bytes32 blockHash;
        uint64 blockTime;
        EthCallData [] result;
    }

    /// @dev EthCallData describes a single ETH call query / response pair.
    struct EthCallData {
        address contractAddress;
        bytes callData;
        bytes result;
    }    

    bytes public constant responsePrefix = bytes("query_response_0000000000000000000|");

    /// @dev getResponseHash computes the hash of the specified query response.
    function getResponseHash(bytes memory response) public pure returns (bytes32) {
        return keccak256(response);
    }

    /// @dev getResponseDigest computes the digest of the specified query response.
    function getResponseDigest(bytes memory response) public pure returns (bytes32) {
        return keccak256(abi.encodePacked(responsePrefix,getResponseHash(response)));
    }
    
    /// @dev parseAndVerifyQueryResponse verifies the query response and returns the parsed response.
    function parseAndVerifyQueryResponse(address wormhole, bytes memory response, IWormhole.Signature[] memory signatures) public view returns (ParsedQueryResponse memory r) {
        verifyQueryResponseSignatures(wormhole, response, signatures);

        uint index = 0;
        
        r.version = response.toUint8(index);
        if (r.version != 1) {
            revert InvalidResponseVersion();
        }
        index += 1;

        r.senderChainId = response.toUint16(index);
        index += 2;

        if (r.senderChainId == 0) {
            r.requestId = response.slice(index, 65);
            index += 65;
        } else {
            r.requestId = response.slice(index, 32);
            index += 32;
        }
        
        uint32 len = response.toUint32(index); // query_request_len
        index += 4;
        uint reqIdx = index;

        if (response.toUint8(reqIdx) != r.version) {
            revert VersionMismatch();
        }
        reqIdx += 1;

        r.nonce = response.toUint32(reqIdx);
        reqIdx += 4;

        uint8 numPerChainQueries = response.toUint8(reqIdx);
        reqIdx += 1;

        // The response starts after the request.
        uint respIdx = index + len;

        if (response.toUint8(respIdx) != numPerChainQueries) {
            revert NumberOfResponsesMismatch();
        }
        respIdx += 1;

        r.responses = new ParsedPerChainQueryResponse[](numPerChainQueries);

        // Walk through the requests and responses in lock step.
        for (uint idx = 0; idx < numPerChainQueries;) {
            r.responses[idx].chainId = response.toUint16(reqIdx);
            if (response.toUint16(respIdx) != r.responses[idx].chainId) {
                revert ChainIdMismatch();
            }
            reqIdx += 2;
            respIdx += 2;

            r.responses[idx].queryType = response.toUint8(reqIdx);
            if (response.toUint8(respIdx) != r.responses[idx].queryType) {
                revert RequestTypeMismatch();
            }
            reqIdx += 1;
            respIdx += 1;
            
            if (r.responses[idx].queryType != 1) {
                revert UnsupportedQueryType();
            }

            len = response.toUint32(reqIdx);
            reqIdx += 4;
            r.responses[idx].request = response.slice(reqIdx, len);
            reqIdx += len;

            len = response.toUint32(respIdx);
            respIdx += 4;
            r.responses[idx].response = response.slice(respIdx, len);
            respIdx += len;

            unchecked { idx += 1; }
        }

        return r;
    }

    /// @dev parseEthCallQueryResponse parses a ParsedPerChainQueryResponse for an ETH call per-chain query.
    function parseEthCallQueryResponse(ParsedPerChainQueryResponse memory pcr) public pure returns (EthCallQueryResponse memory r) {
        if (pcr.queryType != 1) {
                revert UnsupportedQueryType();
        }

        uint reqIdx = 0;
        uint respIdx = 0;

        uint32 len = pcr.request.toUint32(reqIdx); // block_id_len
        reqIdx += 4;

        r.requestBlockId = pcr.request.slice(reqIdx, len);
        reqIdx += len;

        uint8 numBatchCallData = pcr.request.toUint8(reqIdx);
        reqIdx += 1;

        r.blockNum = pcr.response.toUint64(respIdx);
        respIdx += 8;

        r.blockHash = pcr.response.toBytes32(respIdx);
        respIdx += 32;

        r.blockTime = pcr.response.toUint64(respIdx);
        respIdx += 8;

        if (pcr.response.toUint8(respIdx) != numBatchCallData) {
                revert UnexpectedNumberOfResults();
        }
        respIdx += 1;

        r.result = new EthCallData[](numBatchCallData);

        // Walk through the call data and results in lock step.
        for (uint idx = 0; idx < numBatchCallData;) {
            r.result[idx].contractAddress = pcr.request.toAddress(reqIdx);
            reqIdx += 20;

            len = pcr.request.toUint32(reqIdx); // call_data_len
            reqIdx += 4;
            r.result[idx].callData = pcr.request.slice(reqIdx, len);
            reqIdx += len;

            len = pcr.response.toUint32(respIdx); // result_len
            respIdx += 4;
            r.result[idx].result = pcr.response.slice(respIdx, len);
            respIdx += len;

            unchecked { idx += 1; }
        }

        return r;
    }

    /**
     * @dev verifyQueryResponseSignatures verifies the signatures on a query response. It calls into the Wormhole contract.
     * IWormhole.Signature expects the last byte to be bumped by 27 
     * see https://github.com/wormhole-foundation/wormhole/blob/637b1ee657de7de05f783cbb2078dd7d8bfda4d0/ethereum/contracts/Messages.sol#L174
     */
    function verifyQueryResponseSignatures(address _wormhole, bytes memory response, IWormhole.Signature[] memory signatures) public view {
        IWormhole wormhole = IWormhole(_wormhole);
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
}

