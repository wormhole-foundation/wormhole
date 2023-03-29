// SPDX-License-Identifier: Apache 2
pragma solidity ^0.8.0;

import "../libraries/external/BytesLib.sol";

library CoreRelayerLibrary {
    using BytesLib for bytes;

    error WrongModule(bytes32 module);
    error InvalidContractUpgradeAction(uint8 action);
    error InvalidContractUpgradeLength(uint256 length);
    error InvalidRegisterChainAction(uint8);
    error InvalidRegisterChainLength(uint256);
    error InvalidDefaultProviderAction(uint8);
    error InvalidDefaultProviderLength(uint256);

    function parseUpgrade(bytes memory encodedUpgrade, bytes32 module)
        public
        pure
        returns (ContractUpgrade memory cu)
    {
        uint256 index = 0;

        cu.module = encodedUpgrade.toBytes32(index);
        index += 32;

        if (cu.module != module) {
            revert WrongModule(cu.module);
        }

        cu.action = encodedUpgrade.toUint8(index);
        index += 1;

        if (cu.action != 1) {
            revert InvalidContractUpgradeAction(cu.action);
        }

        cu.chain = encodedUpgrade.toUint16(index);
        index += 2;

        cu.newContract = address(uint160(uint256(encodedUpgrade.toBytes32(index))));
        index += 32;

        if (encodedUpgrade.length != index) {
            revert InvalidContractUpgradeLength(encodedUpgrade.length);
        }
    }

    function parseRegisterChain(bytes memory encodedRegistration, bytes32 module)
        public
        pure
        returns (RegisterChain memory registerChain)
    {
        uint256 index = 0;

        registerChain.module = encodedRegistration.toBytes32(index);
        index += 32;

        if (registerChain.module != module) {
            revert WrongModule(registerChain.module);
        }

        registerChain.action = encodedRegistration.toUint8(index);
        index += 1;

        registerChain.chain = encodedRegistration.toUint16(index);
        index += 2;

        if (registerChain.action != 2) {
            revert InvalidRegisterChainAction(registerChain.action);
        }

        registerChain.emitterChain = encodedRegistration.toUint16(index);
        index += 2;

        registerChain.emitterAddress = encodedRegistration.toBytes32(index);
        index += 32;

        if (encodedRegistration.length != index) {
            revert InvalidRegisterChainLength(encodedRegistration.length);
        }
    }

    function parseUpdateDefaultProvider(bytes memory encodedDefaultProvider, bytes32 module)
        public
        pure
        returns (UpdateDefaultProvider memory defaultProvider)
    {
        uint256 index = 0;

        defaultProvider.module = encodedDefaultProvider.toBytes32(index);
        index += 32;

        if (defaultProvider.module != module) {
            revert WrongModule(defaultProvider.module);
        }

        defaultProvider.action = encodedDefaultProvider.toUint8(index);
        index += 1;

        if (defaultProvider.action != 3) {
            revert InvalidDefaultProviderAction(defaultProvider.action);
        }

        defaultProvider.chain = encodedDefaultProvider.toUint16(index);
        index += 2;

        defaultProvider.newProvider = address(uint160(uint256(encodedDefaultProvider.toBytes32(index))));
        index += 32;

        if (encodedDefaultProvider.length != index) {
            revert InvalidDefaultProviderLength(encodedDefaultProvider.length);
        }
    }

    struct ContractUpgrade {
        bytes32 module;
        uint8 action;
        uint16 chain;
        address newContract;
    }

    struct RegisterChain {
        bytes32 module;
        uint8 action;
        uint16 chain; //TODO Why is this on this object?
        uint16 emitterChain;
        bytes32 emitterAddress;
    }

    //This could potentially be combined with ContractUpgrade
    struct UpdateDefaultProvider {
        bytes32 module;
        uint8 action;
        uint16 chain;
        address newProvider;
    }
}
