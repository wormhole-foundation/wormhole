// contracts/query/QueryDemo.sol
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
error ObsoleteUpdate();
error StaleUpdate();
error UnexpectedCallData();
error UnexpectedResultLength();
error UnexpectedResultMismatch();

/// @dev QueryDemo is a library that implements the parsing and verification of Cross Chain Query (CCQ) responses.
contract QueryDemo is QueryResponse {
    using BytesLib for bytes;

    struct ChainEntry {
        uint16 chainID;
        address contractAddress;
        uint256 counter;
        uint256 blockNum;
        uint256 blockTime;
    }

    address private immutable owner;
    address private immutable wormhole;
    uint16 private immutable myChainID;
    mapping(uint16 => ChainEntry) private counters;
    uint16[] private foreignChainIDs;

    bytes4 GetMyCounter = bytes4(hex"916d5743");

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
        counters[_myChainID] = ChainEntry(_myChainID, address(this), 0, 0, 0);
    }

    // updateRegistration should be used to add the other chains and to set / update contract addresses.
    function updateRegistration(uint16 _chainID, address _contractAddress) public onlyOwner {
        if (counters[_chainID].chainID == 0) {
            foreignChainIDs.push(_chainID);
            counters[_chainID].chainID = _chainID;
        }

        counters[_chainID].contractAddress = _contractAddress;
    }

    // getMyCounter (call signature 916d5743) returns the counter value for this chain. It is meant to be used in a cross chain query.
    function getMyCounter() public view returns (uint256) {
        return counters[myChainID].counter;
    }

    // getState() returns this chain's view of all the counters. It is meant to be used in the front end.
    function getState() public view returns (ChainEntry[] memory) {
        ChainEntry[] memory ret = new ChainEntry[](foreignChainIDs.length + 1);
        ret[0] = counters[myChainID];
        uint256 length = foreignChainIDs.length;

        for (uint256 i=0; i < length;) {
            ret[i+1] = counters[foreignChainIDs[i]];
            unchecked {
                ++i;
            }
        }

        return ret;
    }

    // @notice Takes the cross chain query response for the two other counters, stores the results for the other chains, and updates the counter for this chain.
    function updateCounters(bytes memory response, IWormhole.Signature[] memory signatures) public {
        uint256 adjustedBlockTime;
        ParsedQueryResponse memory r = parseAndVerifyQueryResponse(address(wormhole), response, signatures);
        if (r.responses.length != foreignChainIDs.length) {
            revert UnexpectedResultLength();
        }

        for (uint i=0; i < r.responses.length;) {
            // Create a storage pointer for frequently read and updated data stored on the blockchain
            ChainEntry storage chainEntry = counters[r.responses[i].chainId];
            if (chainEntry.chainID != foreignChainIDs[i]) {
                revert InvalidForeignChainID();
            }

            EthCallQueryResponse memory eqr = parseEthCallQueryResponse(r.responses[i]);
            if (eqr.blockNum <= chainEntry.blockNum) {
                revert ObsoleteUpdate();
            }

            // wormhole time is in microseconds, timestamp is in seconds
            adjustedBlockTime = eqr.blockTime / 1_000_000;
            if (adjustedBlockTime <= block.timestamp - 300) {
                revert StaleUpdate();
            }

            if (eqr.result.length != 1) {
                revert UnexpectedResultMismatch();
            }

            if (eqr.result[0].contractAddress != chainEntry.contractAddress) {
                revert InvalidContractAddress();
            }

            // TODO: Is there an easier way to verify that the call data is correct!
            bytes memory callData = eqr.result[0].callData;
            bytes4 result;
            assembly {
                    result := mload(add(callData, 32))
            }
            if (result != GetMyCounter) {
                revert UnexpectedCallData();
            }

            require(eqr.result[0].result.length == 32, "result is not a uint256");

            chainEntry.blockNum = eqr.blockNum;
            chainEntry.blockTime = adjustedBlockTime;
            chainEntry.counter = abi.decode(eqr.result[0].result, (uint256));

            unchecked {
                ++i;
            }
        }

        counters[myChainID].blockNum = block.number;
        counters[myChainID].blockTime = block.timestamp;
        counters[myChainID].counter += 1;
    }

    modifier onlyOwner() {
        if (owner != msg.sender) {
            revert InvalidOwner();
        }
        _;
    }
}
