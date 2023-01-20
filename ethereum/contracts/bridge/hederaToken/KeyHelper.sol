// SPDX-License-Identifier: Apache-2.0
pragma solidity >=0.5.0 <0.9.0;
pragma experimental ABIEncoderV2;

import "./HederaTokenService.sol";

abstract contract KeyHelper {
    using Bits for uint256;
    address supplyContract;

    mapping(KeyType => uint256) keyTypes;

    enum KeyType {
        ADMIN,
        KYC,
        FREEZE,
        WIPE,
        SUPPLY,
        FEE,
        PAUSE
    }
    enum KeyValueType {
        INHERIT_ACCOUNT_KEY,
        CONTRACT_ID,
        ED25519,
        SECP256K1,
        DELEGETABLE_CONTRACT_ID
    }

    constructor() {
        keyTypes[KeyType.ADMIN] = 1;
        keyTypes[KeyType.KYC] = 2;
        keyTypes[KeyType.FREEZE] = 4;
        keyTypes[KeyType.WIPE] = 8;
        keyTypes[KeyType.SUPPLY] = 16;
        keyTypes[KeyType.FEE] = 32;
        keyTypes[KeyType.PAUSE] = 64;
    }

    function getDefaultKeys() internal view returns (IHederaTokenService.TokenKey[] memory keys) {
        keys = new IHederaTokenService.TokenKey[](2);
        keys[0] = getSingleKey(KeyType.KYC, KeyValueType.CONTRACT_ID, '');
        keys[1] = IHederaTokenService.TokenKey(
            getDuplexKeyType(KeyType.SUPPLY, KeyType.PAUSE),
            getKeyValueType(KeyValueType.CONTRACT_ID, '')
        );
    }

    function getAllTypeKeys(KeyValueType keyValueType, bytes memory key)
        internal
        view
        returns (IHederaTokenService.TokenKey[] memory keys)
    {
        keys = new IHederaTokenService.TokenKey[](1);
        keys[0] = IHederaTokenService.TokenKey(getAllKeyTypes(), getKeyValueType(keyValueType, key));
    }

    function getCustomSingleTypeKeys(
        KeyType keyType,
        KeyValueType keyValueType,
        bytes memory key
    ) internal view returns (IHederaTokenService.TokenKey[] memory keys) {
        keys = new IHederaTokenService.TokenKey[](1);
        keys[0] = IHederaTokenService.TokenKey(getKeyType(keyType), getKeyValueType(keyValueType, key));
    }

    function getCustomDuplexTypeKeys(
        KeyType firstType,
        KeyType secondType,
        KeyValueType keyValueType,
        bytes memory key
    ) internal view returns (IHederaTokenService.TokenKey[] memory keys) {
        keys = new IHederaTokenService.TokenKey[](1);
        keys[0] = IHederaTokenService.TokenKey(
            getDuplexKeyType(firstType, secondType),
            getKeyValueType(keyValueType, key)
        );
    }

    function getSingleKey(
        KeyType keyType,
        KeyValueType keyValueType,
        bytes memory key
    ) internal view returns (IHederaTokenService.TokenKey memory tokenKey) {
        tokenKey = IHederaTokenService.TokenKey(getKeyType(keyType), getKeyValueType(keyValueType, key));
    }

    function getSingleKey(
        KeyType keyType,
        KeyValueType keyValueType,
        address key
    ) internal view returns (IHederaTokenService.TokenKey memory tokenKey) {
        tokenKey = IHederaTokenService.TokenKey(getKeyType(keyType), getKeyValueType(keyValueType, key));
    }

    function getSingleKey(
        KeyType firstType,
        KeyType secondType,
        KeyValueType keyValueType,
        bytes memory key
    ) internal view returns (IHederaTokenService.TokenKey memory tokenKey) {
        tokenKey = IHederaTokenService.TokenKey(
            getDuplexKeyType(firstType, secondType),
            getKeyValueType(keyValueType, key)
        );
    }

    function getDuplexKeyType(KeyType firstType, KeyType secondType) internal pure returns (uint256 keyType) {
        keyType = keyType.setBit(uint8(firstType));
        keyType = keyType.setBit(uint8(secondType));
    }

    function getAllKeyTypes() internal pure returns (uint256 keyType) {
        keyType = keyType.setBit(uint8(KeyType.ADMIN));
        keyType = keyType.setBit(uint8(KeyType.KYC));
        keyType = keyType.setBit(uint8(KeyType.FREEZE));
        keyType = keyType.setBit(uint8(KeyType.WIPE));
        keyType = keyType.setBit(uint8(KeyType.SUPPLY));
        keyType = keyType.setBit(uint8(KeyType.FEE));
        keyType = keyType.setBit(uint8(KeyType.PAUSE));
    }

    function getKeyType(KeyType keyType) internal view returns (uint256) {
        return keyTypes[keyType];
    }

    function getKeyValueType(KeyValueType keyValueType, bytes memory key)
        internal
        view
        returns (IHederaTokenService.KeyValue memory keyValue)
    {
        if (keyValueType == KeyValueType.INHERIT_ACCOUNT_KEY) {
            keyValue.inheritAccountKey = true;
        } else if (keyValueType == KeyValueType.CONTRACT_ID) {
            keyValue.contractId = supplyContract;
        } else if (keyValueType == KeyValueType.ED25519) {
            keyValue.ed25519 = key;
        } else if (keyValueType == KeyValueType.SECP256K1) {
            keyValue.ECDSA_secp256k1 = key;
        } else if (keyValueType == KeyValueType.DELEGETABLE_CONTRACT_ID) {
            keyValue.delegatableContractId = supplyContract;
        }
    }

    function getKeyValueType(KeyValueType keyValueType, address keyAddress)
        internal
        pure
        returns (IHederaTokenService.KeyValue memory keyValue)
    {
        if (keyValueType == KeyValueType.CONTRACT_ID) {
            keyValue.contractId = keyAddress;
        } else if (keyValueType == KeyValueType.DELEGETABLE_CONTRACT_ID) {
            keyValue.delegatableContractId = keyAddress;
        }
    }
}

library Bits {
    uint256 internal constant ONE = uint256(1);

    // Sets the bit at the given 'index' in 'self' to '1'.
    // Returns the modified value.
    function setBit(uint256 self, uint8 index) internal pure returns (uint256) {
        return self | (ONE << index);
    }
}
