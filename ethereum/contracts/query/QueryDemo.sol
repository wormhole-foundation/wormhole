// contracts/query/QueryDemo.sol
// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.0;

import "../libraries/external/BytesLib.sol";
import "../interfaces/IWormhole.sol";
import "@openzeppelin/contracts/access/Ownable.sol";
import "@openzeppelin/contracts/utils/Context.sol";
import "../../contracts/query/QueryResponse.sol";

/// @dev QueryDemo is a library that implements the parsing and verification of Cross Chain Query (CCQ) responses.
contract QueryDemo is Context {
    using BytesLib for bytes;

    struct ChainEntry {
        uint16 chainID;
        address contractAddress;
        uint256 counter;
        uint64 blockNum;
        uint64 blockTime;
    }

    address owner;
    address wormhole;
    uint16 myChainID;
    mapping(uint16 => ChainEntry) counters;
    uint16[] chainIDs;

    bytes4 GetMyCounter = bytes4(hex"916d5743");

    function setup(address _owner, address _wormhole, uint16 _myChainID, address _myContractAddress) public {
        owner = _owner;
        wormhole = _wormhole;  
        myChainID = _myChainID;
        counters[myChainID] = ChainEntry(myChainID, _myContractAddress, 0, 0, 0);
        chainIDs.push(myChainID);
    }

    // updateRegistration should be used to add the other chains and to set / update contract addresses.
    function updateRegistration(uint16 _chainID, address _contractAddress) public onlyOwner {
        if (counters[_chainID].chainID == 0) {
            chainIDs.push(_chainID);
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
        ChainEntry[] memory ret = new ChainEntry[](chainIDs.length);
        for (uint idx=0; idx<chainIDs.length; idx++) {
            ret[idx] = counters[chainIDs[idx]];
        }        
        return ret;
    }

    // updateCounters takes the cross chain query response for the two other counters, stores the results for the other chains, and updates the counter for this chain.
    function updateCounters(bytes memory response, IWormhole.Signature[] memory signatures) public {
        QueryResponse.ParsedQueryResponse memory r = QueryResponse.parseAndVerifyQueryResponse(address(wormhole), response, signatures);
        require(r.responses.length == chainIDs.length - 1, "unexpected number of results");
        for (uint idx=0; idx<r.responses.length; idx++) {
            require(counters[r.responses[idx].chainId].chainID != myChainID, "cannot update self");
            require(counters[r.responses[idx].chainId].chainID != 0, "invalid chainID");
            QueryResponse.EthCallQueryResponse memory eqr = QueryResponse.parseEthCallQueryResponse(r.responses[idx]);
            require(eqr.blockNum > counters[r.responses[idx].chainId].blockNum, "update is obsolete");
            require(eqr.blockNum == counters[r.responses[idx].chainId].blockNum, "update is redundant"); // This also prevents multiple entries for the same chain.
            require(eqr.blockTime > block.timestamp - 300, "update is stale");
            require(eqr.result.length == 1, "result mismatch");
            require(eqr.result[0].contractAddress == counters[r.responses[idx].chainId].contractAddress, "contract address is wrong");

            // TODO: Is there an easier way to verify that the call data is correct!
            bytes memory callData = eqr.result[0].callData;
            bytes4 result;
            assembly {
                    result := mload(add(callData, 32))
                }
            require(result == GetMyCounter, "unexpected callData");

            require(eqr.result[0].result.length == 32, "result is not a uint256");
            counters[r.responses[idx].chainId].blockNum = eqr.blockNum;
            counters[r.responses[idx].chainId].blockTime = eqr.blockTime;
            counters[r.responses[idx].chainId].counter = abi.decode(eqr.result[0].result, (uint256));
        }

        counters[myChainID].counter += 1;
    }

    modifier onlyOwner() {
        require(owner == _msgSender(), "caller is not the owner");
        _;
    }
}
