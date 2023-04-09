// contracts/State.sol
// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.0;

contract RelayProviderStorage {
    struct PriceData {
        // The price of purchasing 1 unit of gas on targetChain, denominated in targetChain's wei.
        uint128 gasPrice;
        // The price of the native currency denominated in USD * 10^6
        uint128 nativeCurrencyPrice;
    }

    struct AssetConversion {
        // The following two fields are a percentage buffer that is used to upcharge the user for the value attached to the message sent.
        // The cost of getting ‘targetAmount’ on ‘targetChain’ for the receiverValue is
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
        // Address of the core relayer contract.
        address payable coreRelayer;
        // Dictionary of implementation contract -> initialized flag
        mapping(address => bool) initializedImplementations;
        // Supported chains to deliver to
        mapping(uint16 => bool) supportedChains;
        // Contracts of this relay provider on other chains
        mapping(uint16 => bytes32) targetChainAddresses;
        // Dictionary of wormhole chain id -> price data
        mapping(uint16 => PriceData) data;
        // The delivery overhead gas required to deliver a message to targetChain, denominated in targetChain's gas.
        mapping(uint16 => uint32) deliverGasOverhead;
        // The maximum budget that is allowed for a delivery on target chain, denominated in the targetChain's wei.
        mapping(uint16 => uint256) maximumBudget;
        // Dictionary of wormhole chain id -> assetConversion
        mapping(uint16 => AssetConversion) assetConversion;
        // Reward address for the relayer. The CoreRelayer contract transfers the reward for relaying messages here.
        address payable rewardAddress;
    }
}

contract RelayProviderState {
    RelayProviderStorage.State _state;
}
