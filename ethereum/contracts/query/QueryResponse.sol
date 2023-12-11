// contracts/query/QueryResponse.sol
// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.0;

import {BytesParsing} from "../relayer/libraries/BytesParsing.sol";
import "../interfaces/IWormhole.sol";

// @dev ParsedQueryResponse is returned by QueryResponse.parseAndVerifyQueryResponse().
struct ParsedQueryResponse {
    uint8   version;
    uint16  senderChainId;
    uint32  nonce;
    bytes   requestId; // 65 byte sig for off-chain, 32 byte vaaHash for on-chain
    ParsedPerChainQueryResponse [] responses;
}

// @dev ParsedPerChainQueryResponse describes a single per-chain response.
struct ParsedPerChainQueryResponse {
    uint16 chainId;
    uint8 queryType;
    bytes request;
    bytes response;
}

// @dev EthCallQueryResponse describes an ETH call per-chain query.
struct EthCallQueryResponse {
    bytes requestBlockId;
    uint64 blockNum;
    uint64 blockTime;
    bytes32 blockHash;
    EthCallData [] result;
}

// @dev EthCallByTimestampQueryResponse describes an ETH call by timestamp per-chain query.
struct EthCallByTimestampQueryResponse {
    bytes requestTargetBlockIdHint;
    bytes requestFollowingBlockIdHint;
    uint64 requestTargetTimestamp;
    uint64 targetBlockNum;
    uint64 targetBlockTime;
    uint64 followingBlockNum;
    bytes32 targetBlockHash;
    bytes32 followingBlockHash;
    uint64 followingBlockTime;
    EthCallData [] result;
}

// @dev EthCallWithFinalityQueryResponse describes an ETH call with finality per-chain query.
struct EthCallWithFinalityQueryResponse {
    bytes requestBlockId;
    bytes requestFinality;
    uint64 blockNum;
    uint64 blockTime;
    bytes32 blockHash;
    EthCallData [] result;
}

// @dev EthCallData describes a single ETH call query / response pair.
struct EthCallData {
    address contractAddress;
    bytes callData;
    bytes result;
}

// Custom errors
error EmptyWormholeAddress();
error InvalidResponseVersion();
error VersionMismatch();
error ZeroQueries();
error NumberOfResponsesMismatch();
error ChainIdMismatch();
error RequestTypeMismatch();
error UnsupportedQueryType();
error UnexpectedNumberOfResults();
error InvalidPayloadLength(uint256 received, uint256 expected);
error InvalidContractAddress();
error InvalidFunctionSignature();
error InvalidChainId();
error StaleBlockNum();
error StaleBlockTime();

// @dev QueryResponse is a library that implements the parsing and verification of Cross Chain Query (CCQ) responses.
abstract contract QueryResponse {
    using BytesParsing for bytes;

    IWormhole public immutable wormhole;

    bytes public constant responsePrefix = bytes("query_response_0000000000000000000|");
    uint8 public constant VERSION = 1;
    uint8 public constant QT_ETH_CALL = 1;
    uint8 public constant QT_ETH_CALL_BY_TIMESTAMP = 2;
    uint8 public constant QT_ETH_CALL_WITH_FINALITY = 3;
    uint8 public constant QT_MAX = 4; // Keep this last

    constructor(address _wormhole) {
        if (_wormhole == address(0)) {
            revert EmptyWormholeAddress();
        }

        wormhole = IWormhole(_wormhole);
    }

    /// @dev getResponseHash computes the hash of the specified query response.
    function getResponseHash(bytes memory response) public pure returns (bytes32) {
        return keccak256(response);
    }

    /// @dev getResponseDigest computes the digest of the specified query response.
    function getResponseDigest(bytes memory response) public pure returns (bytes32) {
        return keccak256(abi.encodePacked(responsePrefix,getResponseHash(response)));
    }
    
    /// @dev parseAndVerifyQueryResponse verifies the query response and returns the parsed response.
    function parseAndVerifyQueryResponse(bytes memory response, IWormhole.Signature[] memory signatures) public view returns (ParsedQueryResponse memory r) {
        verifyQueryResponseSignatures(response, signatures);

        uint index = 0;
        
        (r.version, index) = response.asUint8Unchecked(index);
        if (r.version != VERSION) {
            revert InvalidResponseVersion();
        }

        (r.senderChainId, index) = response.asUint16Unchecked(index);

        // For off chain requests (chainID zero), the requestId is the 65 byte signature. For on chain requests, it is the 32 byte VAA hash.
        if (r.senderChainId == 0) {
            (r.requestId, index) = response.sliceUnchecked(index, 65);
        } else {
            (r.requestId, index) = response.sliceUnchecked(index, 32);
        }
        
        uint32 len;
        (len, index) = response.asUint32Unchecked(index); // query_request_len
        uint reqIdx = index;

        // Scope to avoid stack-too-deep error
        {
            uint8 version;
            (version, reqIdx) = response.asUint8Unchecked(reqIdx);
            if (version != r.version) {
                revert VersionMismatch();
            }
        }

        (r.nonce, reqIdx) = response.asUint32Unchecked(reqIdx);

        uint8 numPerChainQueries;
        (numPerChainQueries, reqIdx) = response.asUint8Unchecked(reqIdx);
        
        // A valid query request has at least one per chain query
        if (numPerChainQueries == 0) {
            revert ZeroQueries();
        }

        // The response starts after the request.
        uint respIdx = index + len;
        uint startOfResponse = respIdx;

        uint8 respNumPerChainQueries;
        (respNumPerChainQueries, respIdx) = response.asUint8Unchecked(respIdx);
        if (respNumPerChainQueries != numPerChainQueries) {
            revert NumberOfResponsesMismatch();
        }

        r.responses = new ParsedPerChainQueryResponse[](numPerChainQueries);

        // Walk through the requests and responses in lock step.
        for (uint idx = 0; idx < numPerChainQueries;) {
            (r.responses[idx].chainId, reqIdx) = response.asUint16Unchecked(reqIdx);
            uint16 respChainId;
            (respChainId, respIdx) = response.asUint16Unchecked(respIdx);
            if (respChainId != r.responses[idx].chainId) {
                revert ChainIdMismatch();
            }

            (r.responses[idx].queryType, reqIdx) = response.asUint8Unchecked(reqIdx);
            uint8 respQueryType;
            (respQueryType, respIdx) = response.asUint8Unchecked(respIdx);
            if (respQueryType != r.responses[idx].queryType) {
                revert RequestTypeMismatch();
            }
            
            if (r.responses[idx].queryType < QT_ETH_CALL || r.responses[idx].queryType >= QT_MAX) {
                revert UnsupportedQueryType();
            }

            (len, reqIdx) = response.asUint32Unchecked(reqIdx);
            (r.responses[idx].request, reqIdx) = response.sliceUnchecked(reqIdx, len);

            (len, respIdx) = response.asUint32Unchecked(respIdx);
            (r.responses[idx].response, respIdx) = response.sliceUnchecked(respIdx, len);

            unchecked { ++idx; }
        }

        // End of request body should align with start of response body
        if (startOfResponse != reqIdx) {
            revert InvalidPayloadLength(startOfResponse, reqIdx);
        }

        checkLength(response, respIdx);
        return r;
    }

    /// @dev parseEthCallQueryResponse parses a ParsedPerChainQueryResponse for an ETH call per-chain query.
    function parseEthCallQueryResponse(ParsedPerChainQueryResponse memory pcr) public pure returns (EthCallQueryResponse memory r) {
        if (pcr.queryType != QT_ETH_CALL) {
                revert UnsupportedQueryType();
        }

        uint reqIdx = 0;
        uint respIdx = 0;

        uint32 len;
        (len, reqIdx) = pcr.request.asUint32Unchecked(reqIdx); // block_id_len

        (r.requestBlockId, reqIdx) = pcr.request.sliceUnchecked(reqIdx, len);

        uint8 numBatchCallData;
        (numBatchCallData, reqIdx) = pcr.request.asUint8Unchecked(reqIdx);

        (r.blockNum, respIdx) = pcr.response.asUint64Unchecked(respIdx);

        (r.blockHash, respIdx) = pcr.response.asBytes32Unchecked(respIdx);

        (r.blockTime, respIdx) = pcr.response.asUint64Unchecked(respIdx);

        uint8 respNumResults;
        (respNumResults, respIdx) = pcr.response.asUint8Unchecked(respIdx);
        if (respNumResults != numBatchCallData) {
                revert UnexpectedNumberOfResults();
        }

        r.result = new EthCallData[](numBatchCallData);

        // Walk through the call data and results in lock step.
        for (uint idx = 0; idx < numBatchCallData;) {
            (r.result[idx].contractAddress, reqIdx) = pcr.request.asAddressUnchecked(reqIdx);

            (len, reqIdx) = pcr.request.asUint32Unchecked(reqIdx); // call_data_len
            (r.result[idx].callData, reqIdx) = pcr.request.sliceUnchecked(reqIdx, len);

            (len, respIdx) = pcr.response.asUint32Unchecked(respIdx); // result_len
            (r.result[idx].result, respIdx) = pcr.response.sliceUnchecked(respIdx, len);

            unchecked { ++idx; }
        }

        checkLength(pcr.request, reqIdx);
        checkLength(pcr.response, respIdx);
        return r;
    }

    /// @dev parseEthCallByTimestampQueryResponse parses a ParsedPerChainQueryResponse for an ETH call per-chain query.
    function parseEthCallByTimestampQueryResponse(ParsedPerChainQueryResponse memory pcr) public pure returns (EthCallByTimestampQueryResponse memory r) {
        if (pcr.queryType != QT_ETH_CALL_BY_TIMESTAMP) {
                revert UnsupportedQueryType();
        }

        uint reqIdx = 0;
        uint respIdx = 0;
        uint32 len;

        (r.requestTargetTimestamp, reqIdx) = pcr.request.asUint64Unchecked(reqIdx); // Request target_time_us

        (len, reqIdx) = pcr.request.asUint32Unchecked(reqIdx); // Request target_block_id_hint_len
        (r.requestTargetBlockIdHint, reqIdx) = pcr.request.sliceUnchecked(reqIdx, len); // Request target_block_id_hint
                
        (len, reqIdx) = pcr.request.asUint32Unchecked(reqIdx); // following_block_id_hint_len
        (r.requestFollowingBlockIdHint, reqIdx) = pcr.request.sliceUnchecked(reqIdx, len); // Request following_block_id_hint

        uint8 numBatchCallData;
        (numBatchCallData, reqIdx) = pcr.request.asUint8Unchecked(reqIdx); // Request num_batch_call_data

        (r.targetBlockNum, respIdx) = pcr.response.asUint64Unchecked(respIdx); // Response target_block_number
        (r.targetBlockHash, respIdx) = pcr.response.asBytes32Unchecked(respIdx); // Response target_block_hash
        (r.targetBlockTime, respIdx) = pcr.response.asUint64Unchecked(respIdx); // Response target_block_time_us

        (r.followingBlockNum, respIdx) = pcr.response.asUint64Unchecked(respIdx); // Response following_block_number
        (r.followingBlockHash, respIdx) = pcr.response.asBytes32Unchecked(respIdx); // Response following_block_hash
        (r.followingBlockTime, respIdx) = pcr.response.asUint64Unchecked(respIdx); // Response following_block_time_us

        uint8 respNumResults;
        (respNumResults, respIdx) = pcr.response.asUint8Unchecked(respIdx); // Response num_results
        if (respNumResults != numBatchCallData) {
                revert UnexpectedNumberOfResults();
        }

        r.result = new EthCallData[](numBatchCallData);

        // Walk through the call data and results in lock step.
        for (uint idx = 0; idx < numBatchCallData;) {
            (r.result[idx].contractAddress, reqIdx) = pcr.request.asAddressUnchecked(reqIdx);

            (len, reqIdx) = pcr.request.asUint32Unchecked(reqIdx); // call_data_len
            (r.result[idx].callData, reqIdx) = pcr.request.sliceUnchecked(reqIdx, len);

            (len, respIdx) = pcr.response.asUint32Unchecked(respIdx); // result_len
            (r.result[idx].result, respIdx) = pcr.response.sliceUnchecked(respIdx, len);

            unchecked { ++idx; }
        }

        checkLength(pcr.request, reqIdx);
        checkLength(pcr.response, respIdx);
    }

    /// @dev parseEthCallWithFinalityQueryResponse parses a ParsedPerChainQueryResponse for an ETH call per-chain query.
    function parseEthCallWithFinalityQueryResponse(ParsedPerChainQueryResponse memory pcr) public pure returns (EthCallWithFinalityQueryResponse memory r) {
        if (pcr.queryType != QT_ETH_CALL_WITH_FINALITY) {
                revert UnsupportedQueryType();
        }

        uint reqIdx = 0;
        uint respIdx = 0;
        uint32 len;

        (len, reqIdx) = pcr.request.asUint32Unchecked(reqIdx); // Request block_id_len
        (r.requestBlockId, reqIdx) = pcr.request.sliceUnchecked(reqIdx, len); // Request block_id

        (len, reqIdx) = pcr.request.asUint32Unchecked(reqIdx); // Request finality_len
        (r.requestFinality, reqIdx) = pcr.request.sliceUnchecked(reqIdx, len); // Request finality        

        uint8 numBatchCallData;
        (numBatchCallData, reqIdx) = pcr.request.asUint8Unchecked(reqIdx); // Request num_batch_call_data

        (r.blockNum, respIdx) = pcr.response.asUint64Unchecked(respIdx); // Response block_number

        (r.blockHash, respIdx) = pcr.response.asBytes32Unchecked(respIdx); // Response block_hash

        (r.blockTime, respIdx) = pcr.response.asUint64Unchecked(respIdx); // Response block_time_us

        uint8 respNumResults;
        (respNumResults, respIdx) = pcr.response.asUint8Unchecked(respIdx); // Response num_results
        if (respNumResults != numBatchCallData) {
                revert UnexpectedNumberOfResults();
        }

        r.result = new EthCallData[](numBatchCallData);

        // Walk through the call data and results in lock step.
        for (uint idx = 0; idx < numBatchCallData;) {
            (r.result[idx].contractAddress, reqIdx) = pcr.request.asAddressUnchecked(reqIdx);

            (len, reqIdx) = pcr.request.asUint32Unchecked(reqIdx); // call_data_len
            (r.result[idx].callData, reqIdx) = pcr.request.sliceUnchecked(reqIdx, len);

            (len, respIdx) = pcr.response.asUint32Unchecked(respIdx); // result_len
            (r.result[idx].result, respIdx) = pcr.response.sliceUnchecked(respIdx, len);

            unchecked { ++idx; }
        }

        checkLength(pcr.request, reqIdx);
        checkLength(pcr.response, respIdx);
    }

    /// @dev validateBlockTime validates that the parsed block time isn't stale
    /// @param _blockTime Wormhole block time in MICROseconds
    /// @param _minBlockTime Minium block time in seconds
    function validateBlockTime(uint64 _blockTime, uint256 _minBlockTime) public pure {
        uint256 blockTimeInSeconds = _blockTime / 1_000_000; // Rounds down
        
        if (blockTimeInSeconds < _minBlockTime) {
            revert StaleBlockTime();
        }
    }

    /// @dev validateBlockNum validates that the parsed blockNum isn't stale
    function validateBlockNum(uint64 _blockNum, uint256 _minBlockNum) public pure {
        if (_blockNum < _minBlockNum) {
            revert StaleBlockNum();
        }
    } 

    /// @dev validateChainId validates that the parsed chainId is one of an array of chainIds we expect
    function validateChainId(uint16 chainId, uint16[] memory _validChainIds) public pure {
        bool validChainId = false;

        uint256 numChainIds = _validChainIds.length;
        
        for (uint256 idx = 0; idx < numChainIds;) {
            if (chainId == _validChainIds[idx]) {
                validChainId = true;
                break;
            }

            unchecked { ++idx; }
        }

        if (!validChainId) revert InvalidChainId();
    } 

    /// @dev validateMutlipleEthCallData validates that each EthCallData in an array comes from a function signature and contract address we expect
    function validateMultipleEthCallData(EthCallData[] memory r, address[] memory _expectedContractAddresses, bytes4[] memory _expectedFunctionSignatures) public pure {
        uint256 callDatasLength = r.length;
        
        for (uint256 idx = 0; idx < callDatasLength;) {
            validateEthCallData(r[idx], _expectedContractAddresses, _expectedFunctionSignatures);

            unchecked { ++idx; }
        }
    }

    /// @dev validateEthCallData validates that EthCallData comes from a function signature and contract address we expect
    /// @dev An empty array means we accept all addresses/function signatures
    /// @dev Example 1: To accept signatures 0xaaaaaaaa and 0xbbbbbbbb from `address(abcd)` you'd pass in [0xaaaaaaaa, 0xbbbbbbbb], [address(abcd)]
    /// @dev Example 2: To accept any function signatures from `address(abcd)` or `address(efab)` you'd pass in [], [address(abcd), address(efab)]
    /// @dev Example 3: To accept function signature 0xaaaaaaaa from any address you'd pass in [0xaaaaaaaa], []
    /// @dev WARNING Example 4: If you want to accept signature 0xaaaaaaaa from `address(abcd)` and signature 0xbbbbbbbb from `address(efab)` the following input would be incorrect:
    /// @dev [0xaaaaaaaa, 0xbbbbbbbb], [address(abcd), address(efab)]
    /// @dev This would accept both 0xaaaaaaaa and 0xbbbbbbbb from `address(abcd)` AND `address(efab)`. Instead you should make 2 calls to this method
    /// @dev using the pattern in Example 1. [0xaaaaaaaa], [address(abcd)] OR [0xbbbbbbbb], [address(efab)]
    function validateEthCallData(EthCallData memory r, address[] memory _expectedContractAddresses, bytes4[] memory _expectedFunctionSignatures) public pure {
        bool validContractAddress = _expectedContractAddresses.length == 0 ? true : false;
        bool validFunctionSignature = _expectedFunctionSignatures.length == 0 ? true : false;
        
        uint256 contractAddressesLength = _expectedContractAddresses.length;
        
        // Check that the contract address called in the request is expected
        for (uint256 idx = 0; idx < contractAddressesLength;) {
            if (r.contractAddress == _expectedContractAddresses[idx]) {
                validContractAddress = true;
                break;
            }

            unchecked { ++idx; }
        }

        // Early exit to save gas
        if (!validContractAddress) {
            revert InvalidContractAddress();
        }

        uint256 functionSignaturesLength = _expectedFunctionSignatures.length;

        // Check that the function signature called is expected
        for (uint256 idx = 0; idx < functionSignaturesLength;) {
            (bytes4 funcSig,) = r.callData.asBytes4Unchecked(0);
            if (funcSig == _expectedFunctionSignatures[idx]) {
                validFunctionSignature = true;
                break;
            }

            unchecked { ++idx; }
        }

        if (!validFunctionSignature) {
            revert InvalidFunctionSignature();
        }
    }

    /**
     * @dev verifyQueryResponseSignatures verifies the signatures on a query response. It calls into the Wormhole contract.
     * IWormhole.Signature expects the last byte to be bumped by 27 
     * see https://github.com/wormhole-foundation/wormhole/blob/637b1ee657de7de05f783cbb2078dd7d8bfda4d0/ethereum/contracts/Messages.sol#L174
     */
    function verifyQueryResponseSignatures(bytes memory response, IWormhole.Signature[] memory signatures) public view {
        // It might be worth adding a verifyCurrentQuorum call on the core bridge so that there is only 1 cross call instead of 4.
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

    /// @dev checkLength verifies that the message was fully consumed.
    function checkLength(bytes memory encoded, uint256 expected) private pure {
        if (encoded.length != expected) {
            revert InvalidPayloadLength(encoded.length, expected);
        }
    }
}

