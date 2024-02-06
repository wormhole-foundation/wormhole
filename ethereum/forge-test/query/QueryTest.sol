// SPDX-License-Identifier: Apache 2

// forge test --match-contract QueryTest

pragma solidity ^0.8.4;

// @dev QueryTest is a library to build Cross Chain Query (CCQ) responses for testing purposes.
abstract contract QueryTest {

    /// @dev buildQueryResponseBytes builds a query response from the specified fields.
    function buildQueryResponseBytes(
        uint8 _version,
        uint16 _senderChainId,
        bytes memory _signature,
        uint32 _queryRequestLen,
        bytes memory _queryRequest,
        uint8 _numPerChainResponses,
        bytes memory _perChainResponses
    ) internal pure returns (bytes memory){
        return abi.encodePacked(
            _version,
            _senderChainId,
            _signature,
            _queryRequestLen,
            _queryRequest,
            _numPerChainResponses,
            _perChainResponses // Each created by buildPerChainResponseBytes()
        );
    }

    /// @dev buildPerChainResponseBytes builds a per chain response from the specified fields.
    function buildPerChainResponseBytes(
        uint16 _chainId,
        uint8 _queryType,
        uint32 _responseLen,
        bytes memory _responseBytes
    ) internal pure returns (bytes memory){
        return abi.encodePacked(
            _chainId,
            _queryType,
            _responseLen,
            _responseBytes
        );
    }

    /// @dev buildEthCallResponseBytes builds an eth_call response from the specified fields.
    function buildEthCallResponseBytes(
        uint64 _blockNumber,
        bytes32 _blockHash,
        uint64 _blockTimeUs,
        uint8 _numResults,
        bytes memory _results // Created with buildEthCallResultBytes()
    ) internal pure returns (bytes memory){
        return abi.encodePacked(
            _blockNumber,
            _blockHash,
            _blockTimeUs,
            _numResults,
            _results
        );
    }

    /// @dev buildEthCallByTimestampResponseBytes builds an eth_call_by_timestamp response from the specified fields.
    function buildEthCallByTimestampResponseBytes(
        uint64 _targetBlockNumber,
        bytes32 _targetBlockHash,
        uint64 _targetBlockTimeUs,
        uint64 _followingBlockNumber,
        bytes32 _followingBlockHash,
        uint64 _followingBlockTimeUs,        
        uint8 _numResults,
        bytes memory _results // Created with buildEthCallResultBytes()
    ) internal pure returns (bytes memory){
        return abi.encodePacked(
            _targetBlockNumber,
            _targetBlockHash,
            _targetBlockTimeUs,
            _followingBlockNumber,
            _followingBlockHash,
            _followingBlockTimeUs,            
            _numResults,
            _results
        );
    }

    /// @dev buildEthCallWithFinalityResponseBytes builds an eth_call_with_finality response from the specified fields. Note that it is currently the same as buildEthCallResponseBytes.
    function buildEthCallWithFinalityResponseBytes(
        uint64 _blockNumber,
        bytes32 _blockHash,
        uint64 _blockTimeUs,
        uint8 _numResults,
        bytes memory _results // Created with buildEthCallResultBytes()
    ) internal pure returns (bytes memory){
        return abi.encodePacked(
            _blockNumber,
            _blockHash,
            _blockTimeUs,
            _numResults,
            _results
        );
    }    

    /// @dev buildEthCallResultBytes builds an eth_call result from the specified fields.
    function buildEthCallResultBytes(
        uint32 _resultLen,
        bytes memory _result
    ) internal pure returns (bytes memory){
        return abi.encodePacked(
            _resultLen,
            _result
        );
    }

    /// @dev buildSolanaAccountResponseBytes builds a sol_account response from the specified fields.
    function buildSolanaAccountResponseBytes(
        uint64 _slotNumber,
        uint64 _blockTimeUs,
        bytes32 _blockHash,
        uint8 _numResults,
        bytes memory _results // Created with buildEthCallResultBytes()
    ) internal pure returns (bytes memory){
        return abi.encodePacked(
            _slotNumber,
            _blockTimeUs,            
            _blockHash,
            _numResults,
            _results
        );
    } 
}
