// test/Messages.sol
// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.0;

import "forge-test/rv-helpers/MySetters.sol";
import "forge-test/rv-helpers/TestUtils.sol";

contract TestSetters is TestUtils {
    MySetters setters;

    function setUp() public {
        setters = new MySetters();
    }

    function testUpdateGuardianSetIndex(uint32 index, bytes32 storageSlot)
        public
        unchangedStorage(address(setters), storageSlot)
    {
        vm.assume(storageSlot != GUARDIANSETINDEX_STORAGE_INDEX);

        bytes32 originalSlot = vm.load(address(setters), GUARDIANSETINDEX_STORAGE_INDEX);

        setters.updateGuardianSetIndex_external(index);

        bytes32 updatedSlot = vm.load(address(setters), GUARDIANSETINDEX_STORAGE_INDEX);
        bytes32 mask = bytes32(0xffffffffffffffffffffffffffffffffffffffffffffffffffffffff00000000);
        bytes32 expectedSlot = bytes32(uint256(index)) | (mask & originalSlot);

        assertEq(updatedSlot, expectedSlot);
    }

    function testUpdateGuardianSetIndex_KEVM(uint32 index, bytes32 storageSlot)
        public
        symbolic(address(setters))
    {
        testUpdateGuardianSetIndex(index, storageSlot);
    }

    function testExpireGuardianSet(uint32 timestamp, uint32 index, bytes32 storageSlot)
        public
        unchangedStorage(address(setters), storageSlot)
    {
        bytes32 storageLocation = hashedLocationOffset(index,GUARDIANSETS_STORAGE_INDEX,1);
        vm.assume(storageSlot != storageLocation);
        vm.assume(timestamp <= MAX_UINT32 - 86400);

        bytes32 originalSlot = vm.load(address(setters), storageLocation);

        vm.warp(timestamp);

        setters.expireGuardianSet_external(index);

        bytes32 updatedSlot = vm.load(address(setters), storageLocation);
        bytes32 mask = bytes32(0xffffffffffffffffffffffffffffffffffffffffffffffffffffffff00000000);
        bytes32 expectedSlot = bytes32(uint256(timestamp + 86400)) | (mask & originalSlot);

        assertEq(updatedSlot, expectedSlot);
    }

    function testExpireGuardianSet_KEVM(uint32 timestamp, uint32 index, bytes32 storageSlot)
        public
        symbolic(address(setters))
    {
        testExpireGuardianSet(timestamp, index, storageSlot);
    }

    function testSetInitialized(address newImplementation, bytes32 storageSlot)
        public
        unchangedStorage(address(setters), storageSlot)
    {
        bytes32 storageLocation = hashedLocation(newImplementation, INITIALIZEDIMPLEMENTATIONS_STORAGE_INDEX);
        vm.assume(storageSlot != storageLocation);

        bytes32 originalSlot = vm.load(address(setters), storageLocation);

        setters.setInitialized_external(newImplementation);

        bytes32 updatedSlot = vm.load(address(setters), storageLocation);
        bytes32 mask = bytes32(0xffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff00);
        bytes32 expectedSlot = bytes32(uint256(uint8(0x01))) | (mask & originalSlot);

        assertEq(updatedSlot, expectedSlot);
    }

    function testSetInitialized_KEVM(address newImplementation, bytes32 storageSlot)
        public
        symbolic(address(setters))
    {
        testSetInitialized(newImplementation, storageSlot);
    }

    function testSetGovernanceActionConsumed(bytes32 hash, bytes32 storageSlot)
        public
        unchangedStorage(address(setters), storageSlot)
    {
        bytes32 storageLocation = hashedLocation(hash, CONSUMEDGOVACTIONS_STORAGE_INDEX);
        vm.assume(storageSlot != storageLocation);

        bytes32 originalSlot = vm.load(address(setters), storageLocation);

        setters.setGovernanceActionConsumed_external(hash);

        bytes32 updatedSlot = vm.load(address(setters), storageLocation);
        bytes32 mask = bytes32(0xffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff00);
        bytes32 expectedSlot =  bytes32(uint256(uint8(0x01))) | (mask & originalSlot);

        assertEq(updatedSlot, expectedSlot);
    }

    function testSetGovernanceActionConsumed_KEVM(bytes32 hash, bytes32 storageSlot)
        public
        symbolic(address(setters))
    {
        testSetGovernanceActionConsumed(hash, storageSlot);
    }

    function testSetChainId(uint16 newChainId, bytes32 storageSlot)
        public
        unchangedStorage(address(setters), storageSlot)
    {
        vm.assume(storageSlot != CHAINID_STORAGE_INDEX);

        bytes32 originalSlot = vm.load(address(setters), CHAINID_STORAGE_INDEX);

        setters.setChainId_external(newChainId);

        bytes32 updatedSlot = vm.load(address(setters), CHAINID_STORAGE_INDEX);
        bytes32 mask = bytes32(0xffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff0000);
        bytes32 expectedSlot = bytes32(uint256(newChainId)) | (mask & originalSlot);

        assertEq(updatedSlot, expectedSlot);
    }

    function testSetChainId_KEVM(uint16 newChainId, bytes32 storageSlot)
        public
        symbolic(address(setters))
    {
        testSetChainId(newChainId, storageSlot);
    }

    function testSetGovernanceChainId(uint16 newChainId, bytes32 storageSlot)
        public
        unchangedStorage(address(setters), storageSlot)
    {
        vm.assume(storageSlot != CHAINID_STORAGE_INDEX);

        bytes32 originalSlot = vm.load(address(setters), CHAINID_STORAGE_INDEX);

        setters.setGovernanceChainId_external(newChainId);

        bytes32 updatedSlot = vm.load(address(setters), CHAINID_STORAGE_INDEX);
        bytes32 mask = bytes32(0xffffffffffffffffffffffffffffffffffffffffffffffffffffffff0000ffff);
        bytes32 expectedSlot = bytes32(uint256(newChainId) << 16) | (mask & originalSlot);

        assertEq(updatedSlot, expectedSlot);
    }

    function testSetGovernanceChainId_KEVM(uint16 newChainId, bytes32 storageSlot)
        public
        symbolic(address(setters))
    {
        testSetGovernanceChainId(newChainId, storageSlot);
    }

    function testSetGovernanceContract(bytes32 newGovernanceContract, bytes32 storageSlot)
        public
        unchangedStorage(address(setters), storageSlot)
    {
        vm.assume(storageSlot != GOVERNANCECONTRACT_STORAGE_INDEX);

        setters.setGovernanceContract_external(newGovernanceContract);

        assertEq(newGovernanceContract, vm.load(address(setters), GOVERNANCECONTRACT_STORAGE_INDEX));
    }

    function testSetGovernanceContract_KEVM(bytes32 newGovernanceContract, bytes32 storageSlot)
        public
        symbolic(address(setters))
    {
        testSetGovernanceContract(newGovernanceContract, storageSlot);
    }

    function testSetMessageFee(uint256 newFee, bytes32 storageSlot)
        public
        unchangedStorage(address(setters), storageSlot)
    {
        vm.assume(storageSlot != MESSAGEFEE_STORAGE_INDEX);

        setters.setMessageFee_external(newFee);

        bytes32 updatedSlot = vm.load(address(setters), MESSAGEFEE_STORAGE_INDEX);
        bytes32 expectedSlot = bytes32(newFee);

        assertEq(updatedSlot, expectedSlot);
    }

    function testSetMessageFee_KEVM(uint256 newFee, bytes32 storageSlot)
        public
        symbolic(address(setters))
    {
        testSetMessageFee(newFee, storageSlot);
    }

    function testSetNextSequence(address emitter, uint64 sequence, bytes32 storageSlot)
        public
        unchangedStorage(address(setters), storageSlot)
    {
        bytes32 storageLocation = hashedLocation(emitter, SEQUENCES_STORAGE_INDEX);
        vm.assume(storageSlot != storageLocation);

        bytes32 originalSlot = vm.load(address(setters), storageLocation);

        setters.setNextSequence_external(emitter, sequence);

        bytes32 updatedSlot = vm.load(address(setters), storageLocation);
        bytes32 mask = bytes32(0xffffffffffffffffffffffffffffffffffffffffffffffff0000000000000000);
        bytes32 expectedSlot = bytes32(uint256(sequence)) | (mask & originalSlot);

        assertEq(updatedSlot, expectedSlot);
    }

    function testSetNextSequence_KEVM(address emitter, uint64 sequence, bytes32 storageSlot)
        public
        symbolic(address(setters))
    {
        testSetNextSequence(emitter, sequence, storageSlot);
    }

    function testSetEvmChainId_Success(uint256 newEvmChainId, bytes32 storageSlot)
        public
        unchangedStorage(address(setters), storageSlot)
    {
        vm.assume(storageSlot != EVMCHAINID_STORAGE_INDEX);
        vm.assume(newEvmChainId < 2 ** 64);

        vm.chainId(newEvmChainId);

        setters.setEvmChainId_external(newEvmChainId);

        assertEq(bytes32(newEvmChainId), vm.load(address(setters), EVMCHAINID_STORAGE_INDEX));
    }

    function testSetEvmChainId_Success_KEVM(uint256 newEvmChainId, bytes32 storageSlot)
        public
        symbolic(address(setters))
    {
        testSetEvmChainId_Success(newEvmChainId, storageSlot);
    }

    function testSetEvmChainId_Revert(uint256 newEvmChainId, bytes32 storageSlot)
        public
        unchangedStorage(address(setters), storageSlot)
    {
        vm.assume(newEvmChainId < 2 ** 64);
        vm.assume(newEvmChainId != block.chainid);

        vm.expectRevert("invalid evmChainId");
        setters.setEvmChainId_external(newEvmChainId);
    }

    function testSetEvmChainId_Revert_KEVM(uint256 newEvmChainId, bytes32 storageSlot)
        public
        symbolic(address(setters))
    {
        testSetEvmChainId_Revert(newEvmChainId, storageSlot);
    }
}
