// contracts/State.sol
// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.19;

import "../../interfaces/relayer/TypedUnits.sol";

contract DeliveryProviderStorage {
    struct PriceData {
        // The price of purchasing 1 unit of gas on the target chain, denominated in target chain's wei.
        GasPrice gasPrice;
        // The price of the native currency denominated in USD * 10^6
        WeiPrice nativeCurrencyPrice;
    }

    struct AssetConversion {
        // The following two fields are a percentage buffer that is used to upcharge the user for the value attached to the message sent.
        // The cost of getting ‘targetAmount’ on the target chain for the receiverValue is
        // (denominator + buffer) / (denominator) * (the converted amount in source chain currency using the ‘quoteAssetPrice’ values)
        uint16 buffer;
        uint16 denominator;
    }

    struct State {
        // Wormhole chain id of this blockchain.
        uint16 chainId;
        // Current owner.
        address owner;
        // Pending target of ownership transfer.
        address pendingOwner;
        // Address that is allowed to modify pricing
        address pricingWallet;
        // Address of the core relayer contract.
        address coreRelayer;
        // Dictionary of implementation contract -> initialized flag
        mapping(address => bool) initializedImplementations;
        // Supported chains to deliver to
        mapping(uint16 => bool) supportedChains;
        // Contracts of this relay provider on other chains
        mapping(uint16 => bytes32) targetChainAddresses;
        // Dictionary of wormhole chain id -> price data
        mapping(uint16 => PriceData) data;
        // The delivery overhead gas required to deliver a message to targetChain, denominated in targetChain's gas.
        mapping(uint16 => Gas) deliverGasOverhead;
        // The maximum budget that is allowed for a delivery on target chain, denominated in the targetChain's wei.
        mapping(uint16 => TargetNative) maximumBudget;
        // Dictionary of wormhole chain id -> assetConversion
        mapping(uint16 => AssetConversion) assetConversion;
        // Reward address for the relayer. The WormholeRelayer contract transfers the reward for relaying messages here.
        address payable rewardAddress;
    }
}

contract DeliveryProviderState {
    DeliveryProviderStorage.State _state;
}
