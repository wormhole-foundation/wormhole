// contracts/query/QueryDemo.sol
// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.0;

import "../libraries/external/BytesLib.sol";
import "../interfaces/IWormhole.sol";
import "./QueryResponse.sol";

error InvalidOwner();
// @dev for the onlyOwner modifier
error InvalidCaller();
error InvalidCalldata();
error InvalidWormholeAddress();
error InvalidForeignChainID();
error ObsoleteUpdate();
error StaleUpdate();
error UnexpectedResultLength();
error UnexpectedResultMismatch();

/// @dev QueryDemo is an example of using the QueryResponse library to parse and verify Cross Chain Query (CCQ) responses.
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
    uint16 private immutable myChainID;
    mapping(uint16 => ChainEntry) private counters;
    uint16[] private foreignChainIDs;

    bytes4 GetMyCounter = bytes4(hex"916d5743");

    constructor(address _owner, address _wormhole, uint16 _myChainID) QueryResponse(_wormhole) {
        if (_owner == address(0)) {
            revert InvalidOwner();
        }
        owner = _owner;

        if (_wormhole == address(0)) {
            revert InvalidWormholeAddress();
        }

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

    // @notice Takes the cross chain query response for the other counters, stores the results for the other chains, and updates the counter for this chain.
    function updateCounters(bytes memory response, IWormhole.Signature[] memory signatures) public {
        uint256 adjustedBlockTime;
        ParsedQueryResponse memory r = parseAndVerifyQueryResponse(response, signatures);
        uint numResponses = r.responses.length;
        if (numResponses != foreignChainIDs.length) {
            revert UnexpectedResultLength();
        }

        for (uint i=0; i < numResponses;) {
            // Create a storage pointer for frequently read and updated data stored on the blockchain
            ChainEntry storage chainEntry = counters[r.responses[i].chainId];
            if (chainEntry.chainID != foreignChainIDs[i]) {
                revert InvalidForeignChainID();
            }

            EthCallQueryResponse memory eqr = parseEthCallQueryResponse(r.responses[i]);

            // Validate that update is not obsolete
            validateBlockNum(eqr.blockNum, chainEntry.blockNum, block.number);

            // Validate that update is not stale
            validateBlockTime(eqr.blockTime, block.timestamp - 300, block.timestamp);

            if (eqr.result.length != 1) {
                revert UnexpectedResultMismatch();
            }

            // Validate addresses and function signatures
            address[] memory validAddresses = new address[](1);
            bytes4[] memory validFunctionSignatures = new bytes4[](1);
            validAddresses[0] = chainEntry.contractAddress;
            validFunctionSignatures[0] = GetMyCounter;

            validateMultipleEthCallData(eqr.result, validAddresses, validFunctionSignatures);

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
