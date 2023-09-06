// test/Messages.sol
// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.0;

import "../contracts/Messages.sol";
import "../contracts/Getters.sol";
import "../contracts/Structs.sol";
import "forge-test/rv-helpers/TestUtils.sol";

contract TestGetters is TestUtils {
    Getters getters;

    function setUp() public {
        getters = new Getters();
    }

    function testGetGuardianSetIndex(uint32 index, bytes32 storageSlot)
        public
        unchangedStorage(address(getters), storageSlot)
    {
        vm.assume(storageSlot != GUARDIANSETINDEX_STORAGE_INDEX);

        bytes32 mask = bytes32(0xffffffffffffffffffffffffffffffffffffffffffffffffffffffff00000000);
        bytes32 updatedStorage = storeWithMask(address(getters), GUARDIANSETINDEX_STORAGE_INDEX, bytes32(uint256(index)), mask);

        assertEq(index, getters.getCurrentGuardianSetIndex());
        assertEq(updatedStorage, vm.load(address(getters), GUARDIANSETINDEX_STORAGE_INDEX));
    }

    function testGetGuardianSetIndex_KEVM(uint32 index, bytes32 storageSlot)
        public
        symbolic(address(getters))
    {
        testGetGuardianSetIndex(index, storageSlot);
    }

    function testGetExpireGuardianSet(uint32 timestamp, uint32 index, bytes32 storageSlot)
        public
        unchangedStorage(address(getters), storageSlot)
    {
        bytes32 storageLocation = hashedLocationOffset(index,GUARDIANSETS_STORAGE_INDEX,1);
        vm.assume(storageSlot != storageLocation);

        bytes32 mask = bytes32(0xffffffffffffffffffffffffffffffffffffffffffffffffffffffff00000000);
        bytes32 updatedStorage = storeWithMask(address(getters), storageLocation, bytes32(uint256(timestamp)), mask);

        uint32 expirationTime = getters.getGuardianSet(index).expirationTime;

        assertEq(expirationTime, timestamp);
        assertEq(updatedStorage, vm.load(address(getters), storageLocation));
    }

    function testGetExpireGuardianSet_KEVM(uint32 timestamp, uint32 index, bytes32 storageSlot)
        public
        symbolic(address(getters))
    {
        testGetExpireGuardianSet(timestamp, index, storageSlot);
    }

    function testGetMessageFee(uint256 newFee, bytes32 storageSlot)
        public
        unchangedStorage(address(getters), storageSlot)
    {
        vm.assume(storageSlot != MESSAGEFEE_STORAGE_INDEX);

        vm.store(address(getters), MESSAGEFEE_STORAGE_INDEX, bytes32(newFee));

        assertEq(newFee, getters.messageFee());
        assertEq(bytes32(newFee), vm.load(address(getters), MESSAGEFEE_STORAGE_INDEX));
    }

    function testGetMessageFee_KEVM(uint256 newFee, bytes32 storageSlot)
        public
        symbolic(address(getters))
    {
        testGetMessageFee(newFee, storageSlot);
    }

    function testGetGovernanceContract(bytes32 newGovernanceContract, bytes32 storageSlot)
        public
        unchangedStorage(address(getters), storageSlot)
    {
        vm.assume(storageSlot != GOVERNANCECONTRACT_STORAGE_INDEX);

        vm.store(address(getters), GOVERNANCECONTRACT_STORAGE_INDEX, newGovernanceContract);

        assertEq(newGovernanceContract, getters.governanceContract());
        assertEq(newGovernanceContract, vm.load(address(getters), GOVERNANCECONTRACT_STORAGE_INDEX));
    }

    function testGetGovernanceContract_KEVM(bytes32 newGovernanceContract, bytes32 storageSlot)
        public
        symbolic(address(getters))
    {
        testGetGovernanceContract(newGovernanceContract, storageSlot);
    }

    function testIsInitialized(address newImplementation, uint8 initialized, bytes32 storageSlot)
        public
        unchangedStorage(address(getters), storageSlot)
    {
        bytes32 storageLocation = hashedLocation(newImplementation, INITIALIZEDIMPLEMENTATIONS_STORAGE_INDEX); 
        vm.assume(storageSlot != storageLocation);

        bytes32 mask = bytes32(0xffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff00);
        bytes32 updatedStorage = storeWithMask(address(getters), storageLocation, bytes32(uint256(initialized)), mask);

        assertEq(getters.isInitialized(newImplementation), initialized != 0);
        assertEq(updatedStorage, vm.load(address(getters), storageLocation));
    }

    function testIsInitialized_KEVM(address newImplementation, uint8 initialized, bytes32 storageSlot)
        public
        symbolic(address(getters))
    {
        testIsInitialized(newImplementation, initialized, storageSlot);
    }

    function testGetGovernanceActionConsumed(bytes32 hash, uint8 initialized, bytes32 storageSlot)
        public
        unchangedStorage(address(getters), storageSlot)
    {
        bytes32 storageLocation = hashedLocation(hash, CONSUMEDGOVACTIONS_STORAGE_INDEX);
        vm.assume(storageSlot != storageLocation);

        bytes32 mask = bytes32(0xffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff00);
        bytes32 updatedStorage = storeWithMask(address(getters), storageLocation, bytes32(uint256(initialized)), mask);

        assertEq(getters.governanceActionIsConsumed(hash), initialized != 0);
        assertEq(updatedStorage, vm.load(address(getters), storageLocation));
    }

    function testGetGovernanceActionConsumed_KEVM(bytes32 hash, uint8 initialized, bytes32 storageSlot)
        public
        symbolic(address(getters))
    {
        testGetGovernanceActionConsumed(hash, initialized, storageSlot);
    }

    function testChainId(uint16 newChainId, bytes32 storageSlot)
        public
        unchangedStorage(address(getters), storageSlot)
    {
        vm.assume(storageSlot != CHAINID_STORAGE_INDEX);

        bytes32 mask = bytes32(0xffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff0000);
        bytes32 updatedStorage = storeWithMask(address(getters), CHAINID_STORAGE_INDEX, bytes32(uint256(newChainId)), mask);

        assertEq(getters.chainId(), newChainId);
        assertEq(updatedStorage, vm.load(address(getters), CHAINID_STORAGE_INDEX));
    }

    function testChainId_KEVM(uint16 newChainId, bytes32 storageSlot)
        public
        symbolic(address(getters))
    {
        testChainId(newChainId, storageSlot);
    }

    function testGovernanceChainId(uint16 newChainId, bytes32 storageSlot)
        public
        unchangedStorage(address(getters), storageSlot)
    {
        vm.assume(storageSlot != CHAINID_STORAGE_INDEX);

        bytes32 mask = bytes32(0xffffffffffffffffffffffffffffffffffffffffffffffffffffffff0000ffff);
        bytes32 updatedStorage = storeWithMask(address(getters), CHAINID_STORAGE_INDEX, bytes32(uint256(newChainId)) << 16, mask);

        assertEq(getters.governanceChainId(), newChainId);
        assertEq(updatedStorage, vm.load(address(getters), CHAINID_STORAGE_INDEX));
    }

    function testGovernanceChainId_KEVM(uint16 newChainId, bytes32 storageSlot)
        public
        symbolic(address(getters))
    {
        testGovernanceChainId(newChainId, storageSlot);
    }

    function testNextSequence(address emitter, uint64 sequence, bytes32 storageSlot)
        public
        unchangedStorage(address(getters), storageSlot)
    {
        bytes32 storageLocation = hashedLocation(emitter, SEQUENCES_STORAGE_INDEX); 
        vm.assume(storageSlot != storageLocation);

        bytes32 mask = bytes32(0xffffffffffffffffffffffffffffffffffffffffffffffff0000000000000000);
        bytes32 updatedStorage = storeWithMask(address(getters), storageLocation, bytes32(uint256(sequence)), mask);

        assertEq(getters.nextSequence(emitter), sequence);
        assertEq(updatedStorage, vm.load(address(getters), storageLocation));
    }

    function testNextSequence_KEVM(address emitter, uint64 sequence, bytes32 storageSlot)
        public
        symbolic(address(getters))
    {
        testNextSequence(emitter, sequence, storageSlot);
    }

    function testEvmChainId(uint256 newEvmChainId, bytes32 storageSlot)
        public
        unchangedStorage(address(getters), storageSlot)
    {
        vm.assume(storageSlot != EVMCHAINID_STORAGE_INDEX);

        vm.store(address(getters), EVMCHAINID_STORAGE_INDEX, bytes32(newEvmChainId));

        assertEq(getters.evmChainId(), newEvmChainId);
        assertEq(bytes32(newEvmChainId), vm.load(address(getters), EVMCHAINID_STORAGE_INDEX));
    }

    function testEvmChainId_KEVM(uint256 newEvmChainId, bytes32 storageSlot)
        public
        symbolic(address(getters))
    {
        testEvmChainId(newEvmChainId, storageSlot);
    }
}
