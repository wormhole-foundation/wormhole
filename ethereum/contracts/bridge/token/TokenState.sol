// contracts/State.sol
// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.0;

import "@openzeppelin/contracts/utils/Counters.sol";

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
    using Counters for Counters.Counter;

    TokenStorage.State _state;

    // EIP712
    // Cache the domain separator as an immutable value, but also store the chain id that it corresponds to, in order to
    // invalidate the cached domain separator if the chain id changes.
    string _version;
    bytes32 _cachedDomainSeparator;
    uint256 _cachedChainId;
    address _cachedThis;

    bytes32 _hashedTokenChain;
    bytes32 _hashedNativeContract;
    bytes32 _hashedVersion;
    bytes32 _typeHash;

    // ERC20Permit draft
    mapping(address => Counters.Counter) _nonces;

    // solhint-disable-next-line var-name-mixedcase
    bytes32 constant _PERMIT_TYPEHASH =
        keccak256("Permit(address owner,address spender,uint256 value,uint256 nonce,uint256 deadline)");

    /**
     * @dev In previous versions `_PERMIT_TYPEHASH` was declared as `immutable`.
     * However, to ensure consistency with the upgradeable transpiler, we will continue
     * to reserve a slot.
     * @custom:oz-renamed-from _PERMIT_TYPEHASH
     */
    // solhint-disable-next-line var-name-mixedcase
    bytes32 _PERMIT_TYPEHASH_DEPRECATED_SLOT;

    /**
     * @dev See {IERC20Permit-nonces}.
     */
    function nonces(address owner_) public view returns (uint256) {
        return _nonces[owner_].current();
    }

    /**
     * @dev "Consume a nonce": return the current value and increment.
     *
     * _Available since v4.1._
     */
    function _useNonce(address owner_) internal returns (uint256 current) {
        Counters.Counter storage nonce = _nonces[owner_];
        current = nonce.current();
        nonce.increment();
    }
}