// SPDX-License-Identifier: Apache 2
pragma solidity ^0.8.0;

import "./interfaces/IDelegatedManagerSet.sol";
import "../interfaces/IWormhole.sol";
import "../libraries/external/BytesLib.sol";

string constant DELEGATED_MANAGER_SET_VERSION = "DelegatedManagerSet-0.0.1";

/// @title DelegatedManagerSet
/// @author Wormhole Project Contributors.
/// @notice The DelegatedManagerSet contract is an immutable contract that tracks delegated manager sets per chain.
contract DelegatedManagerSet is IDelegatedManagerSet {
    using BytesLib for bytes;

    string public constant VERSION = DELEGATED_MANAGER_SET_VERSION;
    IWormhole public immutable WORMHOLE;

    // "DelegatedManager" (left padded)
    bytes32 constant MODULE = 0x0000000000000000000000000000000044656C6567617465644D616E61676572;

    mapping(bytes32 => bool) public consumedGovernanceActions;
    mapping(uint16 => uint32) private _currentDelegatedManagerSetIndexes;
    mapping(uint16 => mapping(uint32 => bytes)) private _delegatedManagerSets;

    error InvalidVAA(string reason);
    error InvalidGuardianSet();
    error InvalidGovernanceChain();
    error InvalidGovernanceContract();
    error InvalidModule();
    error InvalidAction();
    error InvalidChain();
    error InvalidIndex();
    error AlreadyConsumed();

    constructor(
        address _wormhole
    ) {
        WORMHOLE = IWormhole(_wormhole);
    }

    // ==================== Internal Functions ===============================================

    // borrowed from the Core bridge
    function verifyGovernanceVm(
        IWormhole.VM memory vm
    ) internal view {
        // Verify the VAA is valid
        (bool isValid, string memory reason) = WORMHOLE.verifyVM(vm);
        if (!isValid) {
            revert InvalidVAA(reason);
        }

        // only current guardianset can sign governance packets
        if (vm.guardianSetIndex != WORMHOLE.getCurrentGuardianSetIndex()) {
            revert InvalidGuardianSet();
        }

        // Verify the VAA is from the governance chain (Solana)
        if (uint16(vm.emitterChainId) != WORMHOLE.governanceChainId()) {
            revert InvalidGovernanceChain();
        }

        // Verify the emitter contract is the governance contract (0x4 left padded)
        if (vm.emitterAddress != WORMHOLE.governanceContract()) {
            revert InvalidGovernanceContract();
        }

        // Verify this governance action hasn't already been
        // consumed to prevent reentry and replay
        if (consumedGovernanceActions[vm.hash]) {
            revert AlreadyConsumed();
        }

        // The governance VAA/VM is valid
    }

    function parseManagerSetUpdate(
        bytes memory encoded
    ) public pure returns (ManagerSetUpdate memory update) {
        uint256 index = 0;

        // governance header

        update.module = encoded.toBytes32(index);
        index += 32;
        if (update.module != MODULE) {
            revert InvalidModule();
        }

        update.action = encoded.toUint8(index);
        index += 1;
        if (update.action != 1) {
            revert InvalidAction();
        }

        update.chainId = encoded.toUint16(index);
        index += 2;

        // payload

        update.managerChainId = encoded.toUint16(index);
        index += 2;

        update.managerSetIndex = encoded.toUint32(index);
        index += 4;

        update.managerSet = encoded.slice(index, encoded.length - index);
    }

    // ==================== External Interface ===============================================

    /// @inheritdoc IDelegatedManagerSet
    function submitNewManagerSet(
        bytes memory encodedVm
    ) external override {
        IWormhole.VM memory vm = WORMHOLE.parseVM(encodedVm);

        // Verify the VAA is valid before processing it
        verifyGovernanceVm(vm);

        ManagerSetUpdate memory update = parseManagerSetUpdate(vm.payload);

        // Verify the VAA is for this chain
        if ((update.chainId != 0) && (update.chainId != WORMHOLE.chainId())) {
            revert InvalidChain();
        }

        // Verify that the manager set index is incrementing via a predictable +1 pattern
        if (update.managerSetIndex != _currentDelegatedManagerSetIndexes[update.managerChainId] + 1)
        {
            revert InvalidIndex();
        }

        // Record the governance action as consumed to prevent reentry
        consumedGovernanceActions[vm.hash] = true;

        // Add the new manager set to _delegatedManagerSets
        _delegatedManagerSets[update.managerChainId][update.managerSetIndex] = update.managerSet;

        // Makes the new manager set effective
        _currentDelegatedManagerSetIndexes[update.managerChainId] = update.managerSetIndex;
    }

    /// @inheritdoc IDelegatedManagerSet
    function getManagerSet(
        uint16 chainId,
        uint32 index
    ) external view override returns (bytes memory) {
        return _delegatedManagerSets[chainId][index];
    }

    /// @inheritdoc IDelegatedManagerSet
    function getCurrentManagerSetIndex(
        uint16 chainId
    ) external view override returns (uint32) {
        return _currentDelegatedManagerSetIndexes[chainId];
    }

    /// @inheritdoc IDelegatedManagerSet
    function getCurrentManagerSet(
        uint16 chainId
    ) external view override returns (bytes memory) {
        return _delegatedManagerSets[chainId][_currentDelegatedManagerSetIndexes[chainId]];
    }
}
