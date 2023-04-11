// contracts/Structs.sol
// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.0;

abstract contract RelayProviderStructs {
    struct UpdatePrice {
        /**
         * Wormhole chain id
         */
        uint16 chainId;
        /**
         * Gas price in ´chainId´ chain.
         */
        uint128 gasPrice;
        /**
         * Price of the native currency in ´chainId´ chain.
         * Native currency is typically used to pay for gas.
         */
        uint128 nativeCurrencyPrice;
    }

    struct TargetChainUpdate {
        /**
         * Wormhole chain id
         */
        uint16 chainId;
        /**
         * Wormhole address of the relay provider in the ´chainId´ chain.
         */
        bytes32 targetChainAddress;
    }

    struct MaximumBudgetUpdate {
        /**
         * Wormhole chain id
         */
        uint16 chainId;
        /**
         * Maximum total budget for a delivery in ´chainId´ chain.
         */
        uint256 maximumTotalBudget;
    }

    struct DeliverGasOverheadUpdate {
        /**
         * Wormhole chain id
         */
        uint16 chainId;
        /**
         * The gas overhead for a delivery in ´chainId´ chain.
         */
        uint32 newGasOverhead;
    }

    struct SupportedChainUpdate {
        /**
         * Wormhole chain id
         */
        uint16 chainId;
        /**
         * True if the chain is supported.
         */
        bool isSupported;
    }

    struct AssetConversionBufferUpdate {
        /**
         * Wormhole chain id
         */
        uint16 chainId;
        // See RelayProviderState.AssetConversion
        uint16 buffer;
        uint16 bufferDenominator;
    }

    struct Update {
        // Update flags
        bool updateAssetConversionBuffer;
        bool updateDeliverGasOverhead;
        bool updatePrice;
        bool updateTargetChainAddress;
        bool updateMaximumBudget;
        bool updateSupportedChain;
        // SupportedChainUpdate
        bool isSupported;
        /**
         * Wormhole chain id
         */
        uint16 chainId;
        // AssetConversionBufferUpdate
        // See RelayProviderState.AssetConversion
        uint16 buffer;
        uint16 bufferDenominator;
        // DeliverGasOverheadUpdate
        /**
         * The gas overhead for a delivery in ´chainId´ chain.
         */
        uint32 newGasOverhead;
        // UpdatePrice
        /**
         * Gas price in ´chainId´ chain.
         */
        uint128 gasPrice;
        /**
         * Price of the native currency in ´chainId´ chain.
         * Native currency is typically used to pay for gas.
         */
        uint128 nativeCurrencyPrice;
        // TargetChainUpdate
        /**
         * Wormhole address of the relay provider in the ´chainId´ chain.
         */
        bytes32 targetChainAddress;
        // MaximumBudgetUpdate
        /**
         * Maximum total budget for a delivery in ´chainId´ chain.
         */
        uint256 maximumTotalBudget;
    }

    struct CoreConfig {
        bool updateCoreRelayer;
        bool updateRewardAddress;
        /**
         * Address of the CoreRelayer contract
         */
        address payable coreRelayer;
        /**
         * Address where rewards are sent for successful relays and sends
         */
        address payable rewardAddress;
    }
}
