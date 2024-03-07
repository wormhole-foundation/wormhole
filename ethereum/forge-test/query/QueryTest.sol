// SPDX-License-Identifier: Apache 2

// forge test --match-contract QueryTest

pragma solidity ^0.8.4;

// @dev QueryTest is a library to build Cross Chain Query (CCQ) responses for testing purposes.
library QueryTest {
    // Custom errors
    error SolanaTooManyPDAs();
    error SolanaTooManySeeds();
    error SolanaSeedTooLong();

    //
    // Query Request stuff
    //

    /// @dev buildOffChainQueryRequestBytes builds an off chain query request from the specified fields.
    function buildOffChainQueryRequestBytes(
        uint8 _version,
        uint32 _nonce,
        uint8 _numPerChainQueries,
        bytes memory _perChainQueries
    ) internal pure returns (bytes memory){
        return abi.encodePacked(
            _version,
            _nonce,
            _numPerChainQueries,
            _perChainQueries // Each created by buildPerChainRequestBytes()
        );
    }

    /// @dev buildPerChainRequestBytes builds a per chain request from the specified fields.
    function buildPerChainRequestBytes(
        uint16 _chainId,
        uint8 _queryType,
        bytes memory _queryBytes
    ) internal pure returns (bytes memory){
        return abi.encodePacked(
            _chainId,
            _queryType,
            uint32(_queryBytes.length),
            _queryBytes
        );
    }

    /// @dev buildEthCallRequestBytes builds an eth_call query request from the specified fields.
    function buildEthCallRequestBytes(
        bytes memory _blockId,
        uint8 _numCallData,
        bytes memory _callData // Created with buildEthCallDataBytes()
    ) internal pure returns (bytes memory){
        return abi.encodePacked(
            uint32(_blockId.length),
            _blockId,
            _numCallData,
            _callData
        );
    }

    /// @dev buildEthCallByTimestampRequestBytes builds an eth_call_by_timestamp query request from the specified fields.
    function buildEthCallByTimestampRequestBytes(
        uint64 _targetTimeUs,
        bytes memory _targetBlockHint,
        bytes memory _followingBlockHint,        
        uint8 _numCallData,
        bytes memory _callData // Created with buildEthCallDataBytes()
    ) internal pure returns (bytes memory){
        return abi.encodePacked(
            _targetTimeUs,
            uint32(_targetBlockHint.length),
            _targetBlockHint,
            uint32(_followingBlockHint.length),
            _followingBlockHint,
            _numCallData,
            _callData
        );
    }

    /// @dev buildEthCallWithFinalityRequestBytes builds an eth_call_with_finality query request from the specified fields.
    function buildEthCallWithFinalityRequestBytes(
        bytes memory _blockId,
        bytes memory _finality,        
        uint8 _numCallData,
        bytes memory _callData // Created with buildEthCallDataBytes()
    ) internal pure returns (bytes memory){
        return abi.encodePacked(
            uint32(_blockId.length),
            _blockId,
            uint32(_finality.length),
            _finality,            
            _numCallData,
            _callData
        );
    }

    /// @dev buildEthCallDataBytes builds the call data associated with one of the eth_call family of queries.
    function buildEthCallDataBytes(
        address _contractAddress,
        bytes memory _callData
    ) internal pure returns (bytes memory){
        return abi.encodePacked(
            _contractAddress,
            uint32(_callData.length),
            _callData
        );
    }
    
    /// @dev buildSolanaAccountRequestBytes builds a sol_account query request from the specified fields.
    function buildSolanaAccountRequestBytes(
        bytes memory _commitment,
        uint64 _minContextSlot,
        uint64 _dataSliceOffset,
        uint64 _dataSliceLength,
        uint8 _numAccounts,
        bytes memory _accounts // Each account is 32 bytes.
    ) internal pure returns (bytes memory){
        return abi.encodePacked(
            uint32(_commitment.length),
            _commitment,
            _minContextSlot,
            _dataSliceOffset,            
            _dataSliceLength,
            _numAccounts,
            _accounts
        );
    }
        
    /// @dev buildSolanaPdaRequestBytes builds a sol_pda query request from the specified fields.
    function buildSolanaPdaRequestBytes(
        bytes memory _commitment,
        uint64 _minContextSlot,
        uint64 _dataSliceOffset,
        uint64 _dataSliceLength,
        bytes[] memory _pdas // Created with multiple calls to buildSolanaPdaEntry()
    ) internal pure returns (bytes memory){
        uint numPdas = _pdas.length;
        if (numPdas > 255) {
            revert SolanaTooManyPDAs();
        }
        bytes memory result = abi.encodePacked(
            uint32(_commitment.length),
            _commitment,
            _minContextSlot,
            _dataSliceOffset,            
            _dataSliceLength,
            uint8(numPdas)
        );

        for (uint idx; idx < numPdas;) {
            result = abi.encodePacked(
                result,
                _pdas[idx]
            );

            unchecked { ++idx; }
        }

        return result;
    }

    /// @dev buildSolanaPdaEntry builds a PDA entry for a sol_pda query.
    function buildSolanaPdaEntry(
        bytes32 _programId,
        uint8 _numSeeds,
        bytes memory _seeds // Created with buildSolanaPdaSeedBytes()
    ) internal pure returns (bytes memory){
        if (_numSeeds > SolanaMaxSeeds) {
            revert SolanaTooManySeeds();
        }
        return abi.encodePacked(
            _programId,
            _numSeeds,
            _seeds
        );
    }

    // According to the spec, there may be at most 16 seeds.
    // https://github.com/gagliardetto/solana-go/blob/6fe3aea02e3660d620433444df033fc3fe6e64c1/keys.go#L559
    uint public constant SolanaMaxSeeds = 16;

    // According to the spec, a seed may be at most 32 bytes.
    // https://github.com/gagliardetto/solana-go/blob/6fe3aea02e3660d620433444df033fc3fe6e64c1/keys.go#L557
    uint public constant SolanaMaxSeedLen = 32;

    /// @dev buildSolanaPdaSeedBytes packs the seeds for a PDA entry into an array of bytes.
    function buildSolanaPdaSeedBytes(
        bytes[] memory _seeds
    ) internal pure returns (bytes memory, uint8){
        uint numSeeds = _seeds.length;
        if (numSeeds > SolanaMaxSeeds) {
            revert SolanaTooManySeeds();
        }

        bytes memory result;

        for (uint idx; idx < numSeeds;) {
            uint seedLen = _seeds[idx].length;
            if (seedLen > SolanaMaxSeedLen) {
                revert SolanaSeedTooLong();
            }
            result = abi.encodePacked(
                result,
                abi.encodePacked(
                    uint32(seedLen),
                    _seeds[idx]
                )
            );

            unchecked { ++idx; }
        }

        return (result, uint8(numSeeds));
    }

    //
    // Query Response stuff
    //

    /// @dev buildQueryResponseBytes builds a query response from the specified fields.
    function buildQueryResponseBytes(
        uint8 _version,
        uint16 _senderChainId,
        bytes memory _signature,
        bytes memory _queryRequest,
        uint8 _numPerChainResponses,
        bytes memory _perChainResponses
    ) internal pure returns (bytes memory){
        return abi.encodePacked(
            _version,
            _senderChainId,
            _signature,
            uint32(_queryRequest.length),
            _queryRequest,
            _numPerChainResponses,
            _perChainResponses // Each created by buildPerChainResponseBytes()
        );
    }

    /// @dev buildPerChainResponseBytes builds a per chain response from the specified fields.
    function buildPerChainResponseBytes(
        uint16 _chainId,
        uint8 _queryType,
        bytes memory _responseBytes
    ) internal pure returns (bytes memory){
        return abi.encodePacked(
            _chainId,
            _queryType,
            uint32(_responseBytes.length),
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
        bytes memory _result
    ) internal pure returns (bytes memory){
        return abi.encodePacked(
            uint32(_result.length),
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

    /// @dev buildSolanaPdaResponseBytes builds a sol_pda response from the specified fields.
    function buildSolanaPdaResponseBytes(
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
