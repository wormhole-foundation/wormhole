// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.19;

import "../../interfaces/relayer/TypedUnits.sol";

abstract contract DeliveryProviderStructs {
    struct UpdatePrice {
        /**
         * Wormhole chain id
         */
        uint16 chainId;
        /**
         * Gas price in ´chainId´ chain.
         */
        GasPrice gasPrice;
        /**
         * Price of the native currency in ´chainId´ chain.
         * Native currency is typically used to pay for gas.
         */
        WeiPrice nativeCurrencyPrice;
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
        Wei maximumTotalBudget;
    }

    struct DeliverGasOverheadUpdate {
        /**
         * Wormhole chain id
         */
        uint16 chainId;
        /**
         * The gas overhead for a delivery in ´chainId´ chain.
         */
        Gas newGasOverhead;
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
        // See DeliveryProviderState.AssetConversion
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
        // See DeliveryProviderState.AssetConversion
        uint16 buffer;
        uint16 bufferDenominator;
        // DeliverGasOverheadUpdate
        /**
         * The gas overhead for a delivery in ´chainId´ chain.
         */
        Gas newGasOverhead;
        // UpdatePrice
        /**
         * Gas price in ´chainId´ chain.
         */
        GasPrice gasPrice;
        /**
         * Price of the native currency in ´chainId´ chain.
         * Native currency is typically used to pay for gas.
         */
        WeiPrice nativeCurrencyPrice;
        // TargetChainUpdate
        /**
         * Wormhole address of the relay provider in the ´chainId´ chain.
         */
        bytes32 targetChainAddress;
        // MaximumBudgetUpdate
        /**
         * Maximum total budget for a delivery in ´chainId´ chain.
         */
        Wei maximumTotalBudget;
    }

    struct CoreConfig {
        bool updateWormholeRelayer;
        bool updateRewardAddress;
        /**
         * Address of the WormholeRelayer contract
         */
        address payable coreRelayer;
        /**
         * Address where rewards are sent for successful relays and sends
         */
        address payable rewardAddress;
    }
}
