// contracts/State.sol
// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.0;

contract NFTStorage {
    struct State {

        // Token name
        string name;

        // Token symbol
        string symbol;

        // Mapping from token ID to owner address
        mapping(uint256 => address) owners;

        // Mapping owner address to token count
        mapping(address => uint256) balances;

        // Mapping from token ID to approved address
        mapping(uint256 => address) tokenApprovals;

        // Mapping from token ID to URI
        mapping(uint256 => string) tokenURIs;

        // Mapping from owner to operator approvals
        mapping(address => mapping(address => bool)) operatorApprovals;

        address owner;

        bool initialized;

        uint16 chainId;
        bytes32 nativeContract;
    }
}

contract NFTState {
    NFTStorage.State _state;
}