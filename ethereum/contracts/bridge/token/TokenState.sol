// contracts/State.sol
// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.0;

contract TokenStorage {
    struct State {
        string name;
        string symbol;

        uint64 metaLastUpdatedSequence;

        uint256 totalSupply;
        uint8 decimals;

        mapping(address => uint256) balances;

        mapping(address => mapping(address => uint256)) allowances;

        address owner;

        bool initialized;

        uint16 chainId;
        bytes32 nativeContract;
    }
}

contract TokenState {
    TokenStorage.State _state;
}