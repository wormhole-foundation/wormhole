// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.19;

import "../../interfaces/relayer/TypedUnits.sol";

// -------------------------------------- Persistent Storage ---------------------------------------

//We have to hardcode the keccak256 values by hand rather than having them calculated because:
//  solc: TypeError: Only direct number constants and references to such constants are supported by
//          inline assembly.
//And presumably what they mean by "direct number constants" is number literals...

struct GovernanceState {
    // mapping of IWormhole.VM.hash of previously executed governance VMs
    mapping(bytes32 => bool) consumedGovernanceActions;
}

//keccak256("GovernanceState") - 1
bytes32 constant GOVERNANCE_STORAGE_SLOT =
    0x970ad24d4754c92e299cabb86552091f5df0a15abc0f1b71f37d3e30031585dc;

function getGovernanceState() pure returns (GovernanceState storage state) {
    assembly ("memory-safe") {
        state.slot := GOVERNANCE_STORAGE_SLOT
    }
}

struct DefaultDeliveryProviderState {
    // address of the default relay provider on this chain
    address defaultDeliveryProvider;
}

//keccak256("DefaultRelayProviderState") - 1
bytes32 constant DEFAULT_RELAY_PROVIDER_STORAGE_SLOT =
    0xebc28a1927f62765bfb7ada566eeab2d31a98c65dbd1e8cad64acae2a3ae45d4;

function getDefaultDeliveryProviderState()
    pure
    returns (DefaultDeliveryProviderState storage state)
{
    assembly ("memory-safe") {
        state.slot := DEFAULT_RELAY_PROVIDER_STORAGE_SLOT
    }
}

struct RegisteredWormholeRelayersState {
    // chainId => wormhole address mapping of relayer contracts on other chains
    mapping(uint16 => bytes32) registeredWormholeRelayers;
}

//keccak256("RegisteredCoreRelayersState") - 1
bytes32 constant REGISTERED_CORE_RELAYERS_STORAGE_SLOT =
    0x9e4e57806ba004485cfae8ca22fb13380f01c10b1b0ccf48c20464961643cf6d;

function getRegisteredWormholeRelayersState()
    pure
    returns (RegisteredWormholeRelayersState storage state)
{
    assembly ("memory-safe") {
        state.slot := REGISTERED_CORE_RELAYERS_STORAGE_SLOT
    }
}

// Replay Protection and Indexing

struct DeliverySuccessState {
    mapping(bytes32 => uint256) deliverySuccessBlock;
}

struct DeliveryFailureState {
    mapping(bytes32 => uint256) deliveryFailureBlock;
}

//keccak256("DeliverySuccessState") - 1
bytes32 constant DELIVERY_SUCCESS_STATE_STORAGE_SLOT =
    0x1b988580e74603c035f5a7f71f2ae4647578af97cd0657db620836b9955fd8f5;

//keccak256("DeliveryFailureState") - 1
bytes32 constant DELIVERY_FAILURE_STATE_STORAGE_SLOT =
    0x6c615753402911c4de18a758def0565f37c41834d6eff72b16cb37cfb697f2a5;

function getDeliverySuccessState() pure returns (DeliverySuccessState storage state) {
    assembly ("memory-safe") {
        state.slot := DELIVERY_SUCCESS_STATE_STORAGE_SLOT
    }
}

function getDeliveryFailureState() pure returns (DeliveryFailureState storage state) {
    assembly ("memory-safe") {
        state.slot := DELIVERY_FAILURE_STATE_STORAGE_SLOT
    }
}

struct ReentrancyGuardState {
    // if 0 address, no reentrancy guard is active
    // otherwise, the address of the contract that has locked the reentrancy guard (msg.sender)
    address lockedBy;
}

//keccak256("ReentrancyGuardState") - 1
bytes32 constant REENTRANCY_GUARD_STORAGE_SLOT =
    0x44dc27ebd67a87ad2af1d98fc4a5f971d9492fe12498e4c413ab5a05b7807a67;

function getReentrancyGuardState() pure returns (ReentrancyGuardState storage state) {
    assembly ("memory-safe") {
        state.slot := REENTRANCY_GUARD_STORAGE_SLOT
    }
}

struct DeliveryTmpState {
    // the refund chain for the in-progress delivery
    uint16 refundChain;
    // the refund address for the in-progress delivery
    bytes32 refundAddress;
}

//keccak256("DeliveryTmpState") - 1
bytes32 constant DELIVERY_TMP_STORAGE_SLOT =
    0x1a2a8eb52f1d00a1242a3f8cc031e30a32870ff64f69009c4e06f75bd842fd22;

function getDeliveryTmpState() pure returns (DeliveryTmpState storage state) {
    assembly ("memory-safe") {
        state.slot := DELIVERY_TMP_STORAGE_SLOT
    }
}