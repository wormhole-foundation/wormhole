// contracts/State.sol
// SPDX-License-Identifier: Apache 2

pragma solidity >=0.6.0 <0.9.0;

import "../../../node_modules/@openzeppelin/contracts/utils/Counters.sol";

contract HederaTokenStorage {
    struct State {
        string name;
        string symbol;

        uint64 metaLastUpdatedSequence;

        uint256 totalSupply;
        uint256 decimals;

        mapping(address => uint256) balances;

        mapping(address => mapping(address => uint256)) allowances;

        address owner;

        bool initialized;

        uint16 chainId;
        bytes32 nativeContract;

        // EIP712
        // Cache the domain separator and salt, but also store the chain id that 
        // it corresponds to, in order to invalidate the cached domain separator
        // if the chain id changes.
        bytes32 cachedDomainSeparator;
        uint256 cachedChainId;
        address cachedThis;
        bytes32 cachedSalt;
        bytes32 cachedHashedName;

        // ERC20Permit draft
        mapping(address => Counters.Counter) nonces;

        address hederaEVMAddress;
    }
}

contract HederaTokenState {
    using Counters for Counters.Counter;

    HederaTokenStorage.State _state;

    /**
     * @dev See {IERC20Permit-nonces}.
     */
    function nonces(address owner_) public view returns (uint256) {
        return _state.nonces[owner_].current();
    }

    /**
     * @dev "Consume a nonce": return the current value and increment.
     */
    function _useNonce(address owner_) internal returns (uint256 current) {
        Counters.Counter storage nonce = _state.nonces[owner_];
        current = nonce.current();
        nonce.increment();
    }
}
