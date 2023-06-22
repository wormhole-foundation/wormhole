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

//keccak256("DefaultDeliveryProviderState") - 1
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

//keccak256("RegisteredWormholeRelayersState") - 1
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

// ---------------------------------- Temporary/Volatile Storage -----------------------------------

//Unlike proper persistent storage, everything below is only used for the lifetime of the current
//  transaction and is (i.e. must be) reset at the end.

struct ForwardInstruction {
    bytes encodedInstruction;
    LocalNative msgValue;
    LocalNative deliveryPrice;
    LocalNative paymentForExtraReceiverValue;
    address payable rewardAddress;
    uint8 consistencyLevel;
}

struct DeliveryTmpState {
    bool deliveryInProgress;
    // the target address that is currently being delivered to (0 for a simple refund)
    address deliveryTarget;
    // the target relay provider address for the in-progress delivery
    address deliveryProvider;
    // the refund chain for the in-progress delivery
    uint16 refundChain;
    // the refund address for the in-progress delivery
    bytes32 refundAddress;
    // Requests which will be forwarded from the current delivery.
    ForwardInstruction[] forwardInstructions;
}

//keccak256("DeliveryTmpState") - 1
bytes32 constant DELIVERY_TMP_STORAGE_SLOT =
    0x1a2a8eb52f1d00a1242a3f8cc031e30a32870ff64f69009c4e06f75bd842fd22;

function getDeliveryTmpState() pure returns (DeliveryTmpState storage state) {
    assembly ("memory-safe") {
        state.slot := DELIVERY_TMP_STORAGE_SLOT
    }
}
