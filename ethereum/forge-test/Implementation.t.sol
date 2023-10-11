// test/Messages.sol
// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.0;

import "../contracts/Implementation.sol";
import "../contracts/Setup.sol";
import "../contracts/Wormhole.sol";
import "../contracts/interfaces/IWormhole.sol";
import "forge-std/Test.sol";
import "forge-test/rv-helpers/TestUtils.sol";

contract TestImplementation is TestUtils {
    event LogMessagePublished(address indexed sender, uint64 sequence, uint32 nonce, bytes payload, uint8 consistencyLevel);

    Wormhole proxy;
    Implementation impl;
    Setup setup;
    Setup proxiedSetup;
    IWormhole proxied;

    uint256 constant testGuardian = 93941733246223705020089879371323733820373732307041878556247502674739205313440;
    bytes32 constant governanceContract = 0x0000000000000000000000000000000000000000000000000000000000000004;
    bytes32 constant MESSAGEFEE_STORAGESLOT = bytes32(uint256(7));
    bytes32 constant SEQUENCES_SLOT = bytes32(uint256(4));

    function setUp() public {
        // Deploy setup
        setup = new Setup();
        // Deploy implementation contract
        impl = new Implementation();
        // Deploy proxy
        proxy = new Wormhole(address(setup), bytes(""));

        address[] memory keys = new address[](1);
        keys[0] = 0xbeFA429d57cD18b7F8A4d91A2da9AB4AF05d0FBe;
        //keys[0] = vm.addr(testGuardian);

        //proxied setup
        proxiedSetup = Setup(address(proxy));

        vm.chainId(1);
        proxiedSetup.setup({
            implementation: address(impl),
            initialGuardians: keys,
            chainId: 2,
            governanceChainId: 1,
            governanceContract: governanceContract,
            evmChainId: 1
        });

        proxied = IWormhole(address(proxy));
    }

    function testPublishMessage(
        bytes32 storageSlot,
        uint256 messageFee,
        address alice,
        uint256 aliceBalance,
        uint32 nonce,
        bytes memory payload,
        uint8 consistencyLevel)
        public
        unchangedStorage(address(proxied), storageSlot)
    {
        uint64 sequence = proxied.nextSequence(alice);
        bytes32 storageLocation = hashedLocation(alice, SEQUENCES_SLOT); 

        vm.assume(aliceBalance >= messageFee);
        vm.assume(storageSlot != storageLocation);
        vm.assume(storageSlot != MESSAGEFEE_STORAGESLOT);

        vm.store(address(proxied), MESSAGEFEE_STORAGESLOT, bytes32(messageFee));
        vm.deal(address(alice),aliceBalance);

        vm.prank(alice);
        proxied.publishMessage{value: messageFee}(nonce, payload, consistencyLevel);

        assertEq(sequence + 1, proxied.nextSequence(alice));
    }

    function testPublishMessage_Emit(
        bytes32 storageSlot,
        uint256 messageFee,
        address alice,
        uint256 aliceBalance,
        uint32 nonce,
        bytes memory payload,
        uint8 consistencyLevel)
        public
        unchangedStorage(address(proxied), storageSlot)
    {
        uint64 sequence = proxied.nextSequence(alice);
        bytes32 storageLocation = hashedLocation(alice, SEQUENCES_SLOT); 

        vm.assume(aliceBalance >= messageFee);
        vm.assume(storageSlot != storageLocation);
        vm.assume(storageSlot != MESSAGEFEE_STORAGESLOT);

        vm.store(address(proxied), MESSAGEFEE_STORAGESLOT, bytes32(messageFee));
        vm.deal(address(alice),aliceBalance);

        vm.prank(alice);
        vm.expectEmit(true, true, true, true);
        emit LogMessagePublished(alice, sequence, nonce, payload, consistencyLevel);

        proxied.publishMessage{value: messageFee}(nonce, payload, consistencyLevel);
    }

    function testPublishMessage_Revert_InvalidFee(
        bytes32 storageSlot,
        uint256 messageFee,
        address alice,
        uint256 aliceBalance,
        uint256 aliceFee,
        uint32 nonce,
        bytes memory payload,
        uint8 consistencyLevel)
        public
        unchangedStorage(address(proxied), storageSlot)
    {
        vm.assume(aliceBalance >= aliceFee);
        vm.assume(aliceFee != messageFee);
        vm.assume(storageSlot != MESSAGEFEE_STORAGESLOT);

        vm.store(address(proxied), MESSAGEFEE_STORAGESLOT, bytes32(messageFee));
        vm.deal(address(alice),aliceBalance);

        vm.prank(alice);
        vm.expectRevert("invalid fee");
        proxied.publishMessage{value: aliceFee}(nonce, payload, consistencyLevel);
    }

    function testPublishMessage_Revert_OutOfFunds(
        bytes32 storageSlot,
        uint256 messageFee,
        address alice,
        uint256 aliceBalance,
        uint32 nonce,
        bytes memory payload,
        uint8 consistencyLevel)
        public
        unchangedStorage(address(proxied), storageSlot)
    {
        vm.assume(aliceBalance < messageFee);
        vm.assume(storageSlot != MESSAGEFEE_STORAGESLOT);

        vm.store(address(proxied), MESSAGEFEE_STORAGESLOT, bytes32(messageFee));
        vm.deal(address(alice),aliceBalance);

        vm.prank(alice);
        vm.expectRevert();
        proxied.publishMessage{value: messageFee}(nonce, payload, consistencyLevel);
    }

    function testShouldBeInitializedWithCorrectSignersAndValues() public {
        uint32 index = proxied.getCurrentGuardianSetIndex();
        IWormhole.GuardianSet memory set = proxied.getGuardianSet(index);

        // check set
        assertEq(set.keys.length, 1, "Guardian set length wrong");
        assertEq(set.keys[0], vm.addr(testGuardian), "Guardian wrong");

        // check expiration
        assertEq(set.expirationTime, 0);

        // chain id
        uint16 chainId = proxied.chainId();
        assertEq(chainId, 2, "Wrong Chain ID");

        // evm chain id
        uint256 evmChainId = proxied.evmChainId();
        assertEq(evmChainId, 1, "Wrong EVM Chain ID");

        // governance
        uint16 governanceChainId = proxied.governanceChainId();
        bytes32 governanceContract = proxied.governanceContract();
        assertEq(governanceChainId, 1, "Wrong governance chain ID");
        assertEq(governanceContract, 0x0000000000000000000000000000000000000000000000000000000000000004, "Wrong governance contract");
    }

    function testShouldLogAPublishedMessageCorrectly() public {
        vm.expectEmit();
        emit LogMessagePublished(address(this), uint64(0), uint32(291), bytes(hex"123321"), uint8(32));
        proxied.publishMessage(0x123, hex"123321", 32);
    }

    function testShouldIncreaseTheSequenceForAnAccount() public {
        proxied.publishMessage(0x1, hex"01", 32);
        uint64 sequence = proxied.publishMessage(0x1, hex"01", 32);
        assertEq(sequence, 1, "Sequence number didn't increase");
    }

    function testParseVMsCorrectly() public {
        uint32 timestamp = 1000;
        uint32 nonce = 1001;
        uint16 emitterChainId = 11;
        bytes32 emitterAddress = 0x0000000000000000000000000000000000000000000000000000000000000eee;
        uint64 sequence = 25;
        uint8 consistencyLevel = 32;
        bytes memory data = hex"aaaaaa";
        
        bytes memory body = abi.encodePacked(timestamp, nonce, emitterChainId, emitterAddress, sequence, consistencyLevel);
        bytes32 bodyHash = keccak256(abi.encodePacked(keccak256(body)));

        

    }
}
