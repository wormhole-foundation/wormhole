// test/Messages.sol
// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.0;

import "../contracts/Implementation.sol";
import "../contracts/Setup.sol";
import "../contracts/Wormhole.sol";
import "../contracts/interfaces/IWormhole.sol";
import "forge-std/Test.sol";
import "forge-test/rv-helpers/TestUtils.sol";
import "../contracts/mock/MockImplementation.sol";

contract TestImplementation is TestUtils {
    event LogMessagePublished(
        address indexed sender,
        uint64 sequence,
        uint32 nonce,
        bytes payload,
        uint8 consistencyLevel
    );

    Wormhole proxy;
    Implementation impl;
    Setup setup;
    Setup proxiedSetup;
    IWormhole public proxied;

    uint256 public constant testGuardian =
        93941733246223705020089879371323733820373732307041878556247502674739205313440;
    uint16 public governanceChainId = 1;
    bytes32 public constant governanceContract =
        0x0000000000000000000000000000000000000000000000000000000000000004;
    bytes32 constant MESSAGEFEE_STORAGESLOT = bytes32(uint256(7));
    bytes32 constant SEQUENCES_SLOT = bytes32(uint256(4));

    uint256 constant testBadSigner1PK =
        61380885381456947260501717894649826485638944763666157704556612272461980735996;
    uint256 constant testSigner1 =
        93941733246223705020089879371323733820373732307041878556247502674739205313440;
    uint256 constant testSigner2 =
        62029033948131772461620424086954761227341731979036746506078649711513083917822;
    uint256 constant testSigner3 =
        61380885381456947260501717894649826485638944763666157704556612272461980735995;

    // "Core" (left padded)
    bytes32 constant core =
        0x00000000000000000000000000000000000000000000000000000000436f7265;
    uint8 actionContractUpgrade = 1;
    uint8 actionGuardianSetUpgrade = 2;
    uint8 actionMessageFee = 3;
    uint8 actionTransferFee = 4;
    uint8 actionRecoverChainId = 5;

    uint16 public testChainId = 2;
    uint256 public testEvmChainId = 1;

    uint16 fakeChainId = 1337;
    uint256 fakeEvmChainId = 10001;

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

        vm.chainId(testEvmChainId);
        proxiedSetup.setup({
            implementation: address(impl),
            initialGuardians: keys,
            chainId: testChainId,
            governanceChainId: 1,
            governanceContract: governanceContract,
            evmChainId: testEvmChainId
        });

        proxied = IWormhole(address(proxy));
    }

    function uint256Array(
        uint256 member
    ) internal pure returns (uint256[] memory arr) {
        arr = new uint256[](1);
        arr[0] = member;
    }

    function testPublishMessage(
        bytes32 storageSlot,
        uint256 messageFee,
        address alice,
        uint256 aliceBalance,
        uint32 nonce,
        bytes memory payload,
        uint8 consistencyLevel
    ) public unchangedStorage(address(proxied), storageSlot) {
        uint64 sequence = proxied.nextSequence(alice);
        bytes32 storageLocation = hashedLocation(alice, SEQUENCES_SLOT);

        vm.assume(aliceBalance >= messageFee);
        vm.assume(storageSlot != storageLocation);
        vm.assume(storageSlot != MESSAGEFEE_STORAGESLOT);

        vm.store(address(proxied), MESSAGEFEE_STORAGESLOT, bytes32(messageFee));
        vm.deal(address(alice), aliceBalance);

        vm.prank(alice);
        proxied.publishMessage{value: messageFee}(
            nonce,
            payload,
            consistencyLevel
        );

        assertEq(sequence + 1, proxied.nextSequence(alice));
    }

    function testPublishMessage_Emit(
        bytes32 storageSlot,
        uint256 messageFee,
        address alice,
        uint256 aliceBalance,
        uint32 nonce,
        bytes memory payload,
        uint8 consistencyLevel
    ) public unchangedStorage(address(proxied), storageSlot) {
        uint64 sequence = proxied.nextSequence(alice);
        bytes32 storageLocation = hashedLocation(alice, SEQUENCES_SLOT);

        vm.assume(aliceBalance >= messageFee);
        vm.assume(storageSlot != storageLocation);
        vm.assume(storageSlot != MESSAGEFEE_STORAGESLOT);

        vm.store(address(proxied), MESSAGEFEE_STORAGESLOT, bytes32(messageFee));
        vm.deal(address(alice), aliceBalance);

        vm.prank(alice);
        vm.expectEmit(true, true, true, true);
        emit LogMessagePublished(
            alice,
            sequence,
            nonce,
            payload,
            consistencyLevel
        );

        proxied.publishMessage{value: messageFee}(
            nonce,
            payload,
            consistencyLevel
        );
    }

    function testPublishMessage_Revert_InvalidFee(
        bytes32 storageSlot,
        uint256 messageFee,
        address alice,
        uint256 aliceBalance,
        uint256 aliceFee,
        uint32 nonce,
        bytes memory payload,
        uint8 consistencyLevel
    ) public unchangedStorage(address(proxied), storageSlot) {
        vm.assume(aliceBalance >= aliceFee);
        vm.assume(aliceFee != messageFee);
        vm.assume(storageSlot != MESSAGEFEE_STORAGESLOT);

        vm.store(address(proxied), MESSAGEFEE_STORAGESLOT, bytes32(messageFee));
        vm.deal(address(alice), aliceBalance);

        vm.prank(alice);
        vm.expectRevert("invalid fee");
        proxied.publishMessage{value: aliceFee}(
            nonce,
            payload,
            consistencyLevel
        );
    }

    /// forge-config: default.allow_internal_expect_revert = true
    function testPublishMessage_Revert_OutOfFunds(
        bytes32 storageSlot,
        uint256 messageFee,
        address alice,
        uint256 aliceBalance,
        uint32 nonce,
        bytes memory payload,
        uint8 consistencyLevel
    ) public unchangedStorage(address(proxied), storageSlot) {
        vm.assume(aliceBalance < messageFee);
        vm.assume(storageSlot != MESSAGEFEE_STORAGESLOT);

        vm.store(address(proxied), MESSAGEFEE_STORAGESLOT, bytes32(messageFee));
        vm.deal(address(alice), aliceBalance);

        vm.prank(alice);
        vm.expectRevert();
        proxied.publishMessage{value: messageFee}(
            nonce,
            payload,
            consistencyLevel
        );
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
        assertEq(chainId, testChainId, "Wrong Chain ID");

        // evm chain id
        uint256 evmChainId = proxied.evmChainId();
        assertEq(evmChainId, testEvmChainId, "Wrong EVM Chain ID");

        // governance
        uint16 readGovernanceChainId = proxied.governanceChainId();
        bytes32 readGovernanceContract = proxied.governanceContract();
        assertEq(
            readGovernanceChainId,
            governanceChainId,
            "Wrong governance chain ID"
        );
        assertEq(
            readGovernanceContract,
            governanceContract,
            "Wrong governance contract"
        );
    }

    function testShouldLogAPublishedMessageCorrectly() public {
        vm.expectEmit();
        emit LogMessagePublished(
            address(this),
            uint64(0),
            uint32(291),
            bytes(hex"123321"),
            uint8(32)
        );
        proxied.publishMessage(0x123, hex"123321", 32);
    }

    function testShouldIncreaseTheSequenceForAnAccount() public {
        proxied.publishMessage(0x1, hex"01", 32);
        uint64 sequence = proxied.publishMessage(0x1, hex"01", 32);
        assertEq(sequence, 1, "Sequence number didn't increase");
    }

    function signAndEncodeVMFixedIndex(
        uint32 timestamp,
        uint32 nonce,
        uint16 emitterChainId,
        bytes32 emitterAddress,
        uint64 sequence,
        bytes memory data,
        uint256[] memory signers,
        uint32 guardianSetIndex,
        uint8 consistencyLevel
    ) public pure returns (bytes memory signedMessage) {
        bytes memory body = abi.encodePacked(
            timestamp,
            nonce,
            emitterChainId,
            emitterAddress,
            sequence,
            consistencyLevel,
            data
        );
        bytes32 bodyHash = keccak256(abi.encodePacked(keccak256(body)));

        // Sign the hash with the devnet guardian private key
        IWormhole.Signature[] memory sigs = new IWormhole.Signature[](
            signers.length
        );
        for (uint256 i = 0; i < signers.length; i++) {
            (sigs[i].v, sigs[i].r, sigs[i].s) = vm.sign(signers[i], bodyHash);
            sigs[i].guardianIndex = 0;
        }

        signedMessage = abi.encodePacked(
            uint8(1),
            guardianSetIndex,
            uint8(sigs.length)
        );

        for (uint256 i = 0; i < signers.length; i++) {
            signedMessage = abi.encodePacked(
                signedMessage,
                uint8(0),
                sigs[i].r,
                sigs[i].s,
                sigs[i].v - 27
            );
        }

        signedMessage = abi.encodePacked(signedMessage, body);
    }

    function signAndEncodeVM(
        uint32 timestamp,
        uint32 nonce,
        uint16 emitterChainId,
        bytes32 emitterAddress,
        uint64 sequence,
        bytes memory data,
        uint256[] memory signers,
        uint32 guardianSetIndex,
        uint8 consistencyLevel
    ) public pure returns (bytes memory signedMessage) {
        bytes memory body = abi.encodePacked(
            timestamp,
            nonce,
            emitterChainId,
            emitterAddress,
            sequence,
            consistencyLevel,
            data
        );
        bytes32 bodyHash = keccak256(abi.encodePacked(keccak256(body)));

        // Sign the hash with the devnet guardian private key
        IWormhole.Signature[] memory sigs = new IWormhole.Signature[](
            signers.length
        );
        for (uint256 i = 0; i < signers.length; i++) {
            (sigs[i].v, sigs[i].r, sigs[i].s) = vm.sign(signers[i], bodyHash);
            sigs[i].guardianIndex = 0;
        }

        signedMessage = abi.encodePacked(
            uint8(1),
            guardianSetIndex,
            uint8(sigs.length)
        );

        for (uint256 i = 0; i < signers.length; i++) {
            signedMessage = abi.encodePacked(
                signedMessage,
                sigs[i].guardianIndex,
                sigs[i].r,
                sigs[i].s,
                sigs[i].v - 27
            );
        }

        signedMessage = abi.encodePacked(signedMessage, body);
    }

    function testParseVMsCorrectly() public {
        uint32 timestamp = 1000;
        uint32 nonce = 1001;
        uint16 emitterChainId = 11;
        bytes32 emitterAddress = 0x0000000000000000000000000000000000000000000000000000000000000eee;
        uint64 sequence = 0;
        uint8 consistencyLevel = 2;
        uint32 guardianSetIndex = 0;
        bytes memory data = hex"aaaaaa";

        bytes memory signedMessage = signAndEncodeVM(
            timestamp,
            nonce,
            emitterChainId,
            emitterAddress,
            sequence,
            data,
            uint256Array(testGuardian),
            guardianSetIndex,
            consistencyLevel
        );

        (IWormhole.VM memory parsed, bool valid, string memory reason) = proxied
            .parseAndVerifyVM(signedMessage);

        assertEq(parsed.version, 1, "Wrong VM version");
        assertEq(parsed.timestamp, timestamp, "Wrong VM timestamp");
        assertEq(parsed.nonce, nonce, "Wrong VM nonce");
        assertEq(
            parsed.emitterChainId,
            emitterChainId,
            "Wrong emitter chain id"
        );
        assertEq(
            parsed.emitterAddress,
            emitterAddress,
            "Wrong emitter address"
        );
        assertEq(parsed.payload, data, "Wrong VM payload");
        assertEq(parsed.guardianSetIndex, 0, "Wrong VM guardian set index");
        assertEq(parsed.sequence, sequence, "Wrong VM sequence");
        assertEq(
            parsed.consistencyLevel,
            consistencyLevel,
            "Wrong VM consistency level"
        );
        assertEq(valid, true, "Signed vaa not valid");
        assertEq(reason, "", "Wrong reason");
    }

    function testShouldFailQuorumOnVMsWithNoSigners() public {
        uint32 timestamp = 1000;
        uint32 nonce = 1001;
        uint16 emitterChainId = 11;
        bytes32 emitterAddress = 0x0000000000000000000000000000000000000000000000000000000000000eee;
        uint64 sequence = 0;
        uint8 consistencyLevel = 2;
        uint32 guardianSetIndex = 0;
        bytes memory data = hex"aaaaaa";

        bytes memory signedMessage = signAndEncodeVM(
            timestamp,
            nonce,
            emitterChainId,
            emitterAddress,
            sequence,
            data,
            new uint256[](0),
            guardianSetIndex,
            consistencyLevel
        );

        (, bool valid, string memory reason) = proxied.parseAndVerifyVM(
            signedMessage
        );

        assertEq(valid, false, "Signed vaa shouldn't be valid");
        assertEq(reason, "no quorum", "Wrong reason");
    }

    function testShouldFailToVerifyOnVMsWithBadSigner() public {
        uint32 timestamp = 1000;
        uint32 nonce = 1001;
        uint16 emitterChainId = 11;
        bytes32 emitterAddress = 0x0000000000000000000000000000000000000000000000000000000000000eee;
        uint64 sequence = 0;
        uint8 consistencyLevel = 2;
        uint32 guardianSetIndex = 0;
        bytes memory data = hex"aaaaaa";

        bytes memory signedMessage = signAndEncodeVM(
            timestamp,
            nonce,
            emitterChainId,
            emitterAddress,
            sequence,
            data,
            uint256Array(testBadSigner1PK),
            guardianSetIndex,
            consistencyLevel
        );

        (, bool valid, string memory reason) = proxied.parseAndVerifyVM(
            signedMessage
        );

        assertEq(valid, false, "Signed vaa shouldn't be valid");
        assertEq(reason, "VM signature invalid", "Wrong reason");
    }

    function testShouldErrorOnVMsWithInvalidGuardianSetIndex() public {
        uint32 timestamp = 1000;
        uint32 nonce = 1001;
        uint16 emitterChainId = 11;
        bytes32 emitterAddress = 0x0000000000000000000000000000000000000000000000000000000000000eee;
        uint64 sequence = 0;
        uint8 consistencyLevel = 2;
        uint32 guardianSetIndex = 200;
        bytes memory data = hex"aaaaaa";

        bytes memory signedMessage = signAndEncodeVM(
            timestamp,
            nonce,
            emitterChainId,
            emitterAddress,
            sequence,
            data,
            uint256Array(testGuardian),
            guardianSetIndex,
            consistencyLevel
        );

        (, bool valid, string memory reason) = proxied
            .parseAndVerifyVM(signedMessage);

        assertEq(valid, false, "Signed vaa shouldn't be valid");
        assertEq(reason, "invalid guardian set", "Wrong reason");
    }

    function testShouldRevertOnVMsWithDuplicateNonMonotonicSignatureIndexes()
        public
    {
        uint32 timestamp = 1000;
        uint32 nonce = 1001;
        uint16 emitterChainId = 11;
        bytes32 emitterAddress = 0x0000000000000000000000000000000000000000000000000000000000000eee;
        uint64 sequence = 0;
        uint8 consistencyLevel = 2;
        uint32 guardianSetIndex = 0;
        bytes memory data = hex"aaaaaa";
        uint256[] memory signers = new uint256[](3);
        signers[0] = testSigner1;
        signers[1] = testSigner2;
        signers[2] = testSigner3;
        bytes memory signedMessage = signAndEncodeVMFixedIndex(
            timestamp,
            nonce,
            emitterChainId,
            emitterAddress,
            sequence,
            data,
            signers,
            guardianSetIndex,
            consistencyLevel
        );

        vm.expectRevert("signature indices must be ascending");
        proxied.parseAndVerifyVM(signedMessage);
    }

    function testShouldSetAndEnforceFees() public {
        uint32 timestamp = 1000;
        uint32 nonce = 1001;
        uint256 messageFee = 1111;
        bytes memory data = abi.encodePacked(
            core,
            actionMessageFee,
            testChainId,
            messageFee
        );
        bytes memory vaa = signAndEncodeVM(
            timestamp,
            nonce,
            governanceChainId,
            governanceContract,
            0,
            data,
            uint256Array(testGuardian),
            0,
            2
        );

        uint256 before = proxied.messageFee();
        proxied.submitSetMessageFee(vaa);
        uint256 afterSettingFee = proxied.messageFee();
        assertTrue(before != afterSettingFee, "message fee did not update");
        assertEq(afterSettingFee, messageFee, "wrong message fee");
    }

    function testShouldTransferOutCollectedFees() public {
        address receiver = address(0x1234123412341234123412341234123412341234);

        uint32 timestamp = 1000;
        uint32 nonce = 1001;
        uint256 amount = 11;

        vm.deal(address(proxied), amount);
        bytes memory data = abi.encodePacked(
            core,
            actionTransferFee,
            testChainId,
            amount,
            addressToBytes32(receiver)
        );
        bytes memory vaa = signAndEncodeVM(
            timestamp,
            nonce,
            governanceChainId,
            governanceContract,
            0,
            data,
            uint256Array(testGuardian),
            0,
            2
        );

        uint256 receiverBefore = receiver.balance;
        uint256 whBefore = address(proxied).balance;
        proxied.submitTransferFees(vaa);
        uint256 receiverAfter = receiver.balance;
        uint256 whAfter = address(proxied).balance;
        assertEq(
            receiverAfter - receiverBefore,
            amount,
            "Receiver balance didn't change correctly"
        );
        assertEq(
            whBefore - whAfter,
            amount,
            "WH Core balance didn't change correctly"
        );
    }

    function testShouldRevertWhenSubmittingANewGuardianSetWithTheZeroAddress()
        public
    {
        uint32 timestamp = 1000;
        uint32 nonce = 1001;
        address zeroAddress = address(0x0);

        uint32 oldGuardianSetIndex = proxied.getCurrentGuardianSetIndex();

        bytes memory data = abi.encodePacked(
            core,
            actionGuardianSetUpgrade,
            testChainId,
            oldGuardianSetIndex + 1,
            uint8(3),
            vm.addr(testSigner1),
            vm.addr(testSigner2),
            zeroAddress
        );
        bytes memory vaa = signAndEncodeVM(
            timestamp,
            nonce,
            governanceChainId,
            governanceContract,
            0,
            data,
            uint256Array(testGuardian),
            0,
            2
        );

        vm.expectRevert("Invalid key");
        proxied.submitNewGuardianSet(vaa);
    }

    function testShouldAcceptANewGuardianSet() public {
        uint32 timestamp = 1000;
        uint32 nonce = 1001;

        uint32 oldGuardianSetIndex = proxied.getCurrentGuardianSetIndex();

        bytes memory data = abi.encodePacked(
            core,
            actionGuardianSetUpgrade,
            testChainId,
            oldGuardianSetIndex + 1,
            uint8(3),
            vm.addr(testSigner1),
            vm.addr(testSigner2),
            vm.addr(testSigner3)
        );
        bytes memory vaa = signAndEncodeVM(
            timestamp,
            nonce,
            governanceChainId,
            governanceContract,
            0,
            data,
            uint256Array(testGuardian),
            0,
            2
        );

        proxied.submitNewGuardianSet(vaa);

        uint32 newIndex = proxied.getCurrentGuardianSetIndex();
        assertEq(
            oldGuardianSetIndex + 1,
            newIndex,
            "New index is one more than old index"
        );

        IWormhole.GuardianSet memory guardianSet = proxied.getGuardianSet(
            newIndex
        );

        assertEq(guardianSet.expirationTime, 0, "Wrong expiration time");
        assertEq(guardianSet.keys[0], vm.addr(testSigner1), "Wrong guardian");
        assertEq(guardianSet.keys[1], vm.addr(testSigner2), "Wrong guardian");
        assertEq(guardianSet.keys[2], vm.addr(testSigner3), "Wrong guardian");

        IWormhole.GuardianSet memory oldGuardianSet = proxied.getGuardianSet(
            oldGuardianSetIndex
        );

        assertTrue(
            (oldGuardianSet.expirationTime > block.timestamp + 86000) &&
                (oldGuardianSet.expirationTime < block.timestamp + 88000),
            "Wrong expiration time"
        );
        assertEq(
            oldGuardianSet.keys[0],
            vm.addr(testGuardian),
            "Wrong guardian"
        );
    }

    function testShouldAcceptSmartContractUpgrades() public {
        uint32 timestamp = 1000;
        uint32 nonce = 1001;

        MockImplementation mock = new MockImplementation();

        bytes memory data = abi.encodePacked(
            core,
            actionContractUpgrade,
            testChainId,
            addressToBytes32(address(mock))
        );
        bytes memory vaa = signAndEncodeVM(
            timestamp,
            nonce,
            governanceChainId,
            governanceContract,
            0,
            data,
            uint256Array(testGuardian),
            0,
            2
        );

        bytes32 IMPLEMENTATION_STORAGE_SLOT = 0x360894a13ba1a3210667c828492db98dca3e2076cc3735a920a3ca505d382bbc;

        proxied.submitContractUpgrade(vaa);

        bytes32 afterUpgrade = vm.load(
            address(proxied),
            IMPLEMENTATION_STORAGE_SLOT
        );
        assertEq(afterUpgrade, addressToBytes32(address(mock)));
        assertEq(
            MockImplementation(payable(address(proxied)))
                .testNewImplementationActive(),
            true,
            "New implementation not active"
        );
    }

    function testShouldRevertRecoverChainIDGovernancePacketsOnCanonicalChainsNonFork()
        public
    {
        uint32 timestamp = 1000;
        uint32 nonce = 1001;

        bytes memory data = abi.encodePacked(
            core,
            actionRecoverChainId,
            testEvmChainId,
            testChainId
        );

        bytes memory vaa = signAndEncodeVM(
            timestamp,
            nonce,
            governanceChainId,
            governanceContract,
            0,
            data,
            uint256Array(testGuardian),
            0,
            2
        );

        vm.expectRevert("not a fork");
        proxied.submitRecoverChainId(vaa);
    }

    function testShouldRevertGovernancePacketsFromOldGuardianSet() public {
        uint32 timestamp = 1000;
        uint32 nonce = 1001;

        // upgrade guardian set
        bytes memory data = abi.encodePacked(
            core,
            actionGuardianSetUpgrade,
            testChainId,
            uint32(1),
            uint8(3),
            vm.addr(testSigner1),
            vm.addr(testSigner2),
            vm.addr(testSigner3)
        );
        bytes memory vaa = signAndEncodeVM(
            timestamp,
            nonce,
            governanceChainId,
            governanceContract,
            0,
            data,
            uint256Array(testGuardian),
            0,
            2
        );

        proxied.submitNewGuardianSet(vaa);

        data = abi.encodePacked(
            core,
            actionTransferFee,
            testChainId,
            uint256(1),
            address(0)
        );
        vaa = signAndEncodeVM(
            timestamp,
            nonce,
            governanceChainId,
            governanceContract,
            0,
            data,
            uint256Array(testGuardian),
            0,
            2
        );

        vm.expectRevert("not signed by current guardian set");
        proxied.submitTransferFees(vaa);
    }

    function testShouldTimeOutOldGuardians() public {
        uint32 timestamp = 1000;
        uint32 nonce = 1001;

        // upgrade guardian set
        bytes memory data = abi.encodePacked(
            core,
            actionGuardianSetUpgrade,
            testChainId,
            uint32(1),
            uint8(3),
            vm.addr(testSigner1),
            vm.addr(testSigner2),
            vm.addr(testSigner3)
        );
        bytes memory vaa = signAndEncodeVM(
            timestamp,
            nonce,
            governanceChainId,
            governanceContract,
            0,
            data,
            uint256Array(testGuardian),
            0,
            2
        );

        proxied.submitNewGuardianSet(vaa);

        data = hex"aaaaaa";
        vaa = signAndEncodeVM(
            timestamp,
            nonce,
            governanceChainId,
            governanceContract,
            0,
            data,
            uint256Array(testGuardian),
            0,
            2
        );

        (, bool valid, ) = proxied.parseAndVerifyVM(vaa);

        assertEq(valid, true, "Vaa should be valid");

        skip(100000);

        (, valid, ) = proxied.parseAndVerifyVM(vaa);

        assertEq(valid, false, "Vaa should be expired");
    }

    function testShouldRevertGovernancePacketsFromWrongGovernanceChain()
        public
    {
        uint32 timestamp = 1000;
        uint32 nonce = 1001;

        bytes memory data = abi.encodePacked(
            core,
            actionTransferFee,
            testChainId,
            uint256(1),
            address(0)
        );
        bytes memory vaa = signAndEncodeVM(
            timestamp,
            nonce,
            999,
            governanceContract,
            0,
            data,
            uint256Array(testGuardian),
            0,
            2
        );

        vm.expectRevert("wrong governance chain");
        proxied.submitTransferFees(vaa);
    }

    function testShouldRevertGovernancePacketsFromWrongGovernanceContract()
        public
    {
        uint32 timestamp = 1000;
        uint32 nonce = 1001;

        bytes memory data = abi.encodePacked(
            core,
            actionTransferFee,
            testChainId,
            uint256(1),
            address(0)
        );
        bytes memory vaa = signAndEncodeVM(
            timestamp,
            nonce,
            governanceChainId,
            core,
            0,
            data,
            uint256Array(testGuardian),
            0,
            2
        );

        vm.expectRevert("wrong governance contract");
        proxied.submitTransferFees(vaa);
    }

    function testShouldRevertGovernancePacketsThatAlreadyHaveBeenApplied()
        public
    {
        uint32 timestamp = 1000;
        uint32 nonce = 1001;

        uint256 amount = 1;
        vm.deal(address(proxied), amount);

        bytes memory data = abi.encodePacked(
            core,
            actionTransferFee,
            testChainId,
            amount,
            addressToBytes32(address(0))
        );
        bytes memory vaa = signAndEncodeVM(
            timestamp,
            nonce,
            governanceChainId,
            governanceContract,
            0,
            data,
            uint256Array(testGuardian),
            0,
            2
        );

        proxied.submitTransferFees(vaa);

        vm.expectRevert("governance action already consumed");
        proxied.submitTransferFees(vaa);
    }

    function addressToBytes32(address input) internal pure returns (bytes32 output) {
        return bytes32(uint256(uint160(input)));
    }

    function testShouldRejectSmartContractUpgradesOnForks() public {
        uint32 timestamp = 1000;
        uint32 nonce = 1001;

        // Perform a successful upgrade
        MockImplementation mock = new MockImplementation();

        bytes memory data = abi.encodePacked(
            core,
            actionContractUpgrade,
            testChainId,
            addressToBytes32(address(mock))
        );
        bytes memory vaa = signAndEncodeVM(
            timestamp,
            nonce,
            governanceChainId,
            governanceContract,
            0,
            data,
            uint256Array(testGuardian),
            0,
            2
        );

        bytes32 IMPLEMENTATION_STORAGE_SLOT = 0x360894a13ba1a3210667c828492db98dca3e2076cc3735a920a3ca505d382bbc;

        proxied.submitContractUpgrade(vaa);

        bytes32 afterUpgrade = vm.load(
            address(proxied),
            IMPLEMENTATION_STORAGE_SLOT
        );
        assertEq(afterUpgrade, addressToBytes32(address(mock)));
        assertEq(
            MockImplementation(payable(address(proxied)))
                .testNewImplementationActive(),
            true,
            "New implementation not active"
        );

        // Overwrite EVM Chain ID
        MockImplementation(payable(address(proxied))).testOverwriteEVMChainId(
            fakeChainId,
            fakeEvmChainId
        );
        assertEq(
            proxied.chainId(),
            fakeChainId,
            "Overwrite didn't work for chain ID"
        );
        assertEq(
            proxied.evmChainId(),
            fakeEvmChainId,
            "Overwrite didn't work for evm chain ID"
        );

        data = abi.encodePacked(
            core,
            actionContractUpgrade,
            testChainId,
            addressToBytes32(address(mock))
        );
        vaa = signAndEncodeVM(
            timestamp,
            nonce,
            governanceChainId,
            governanceContract,
            0,
            data,
            uint256Array(testGuardian),
            0,
            2
        );

        vm.expectRevert("invalid fork");
        proxied.submitContractUpgrade(vaa);
    }

    function testShouldAllowRecoverChainIDGovernancePacketsForks() public {
        uint32 timestamp = 1000;
        uint32 nonce = 1001;

        // Perform a successful upgrade
        MockImplementation mock = new MockImplementation();

        bytes memory data = abi.encodePacked(
            core,
            actionContractUpgrade,
            testChainId,
            addressToBytes32(address(mock))
        );
        bytes memory vaa = signAndEncodeVM(
            timestamp,
            nonce,
            governanceChainId,
            governanceContract,
            0,
            data,
            uint256Array(testGuardian),
            0,
            2
        );

        bytes32 IMPLEMENTATION_STORAGE_SLOT = 0x360894a13ba1a3210667c828492db98dca3e2076cc3735a920a3ca505d382bbc;

        proxied.submitContractUpgrade(vaa);

        bytes32 afterUpgrade = vm.load(
            address(proxied),
            IMPLEMENTATION_STORAGE_SLOT
        );
        assertEq(afterUpgrade, addressToBytes32(address(mock)));
        assertEq(
            MockImplementation(payable(address(proxied)))
                .testNewImplementationActive(),
            true,
            "New implementation not active"
        );

        // Overwrite EVM Chain ID
        MockImplementation(payable(address(proxied))).testOverwriteEVMChainId(
            fakeChainId,
            fakeEvmChainId
        );
        assertEq(
            proxied.chainId(),
            fakeChainId,
            "Overwrite didn't work for chain ID"
        );
        assertEq(
            proxied.evmChainId(),
            fakeEvmChainId,
            "Overwrite didn't work for evm chain ID"
        );

        // recover chain ID
        data = abi.encodePacked(
            core,
            actionRecoverChainId,
            testEvmChainId,
            testChainId
        );
        vaa = signAndEncodeVM(
            timestamp,
            nonce,
            governanceChainId,
            governanceContract,
            0,
            data,
            uint256Array(testGuardian),
            0,
            2
        );

        proxied.submitRecoverChainId(vaa);

        assertEq(
            proxied.chainId(),
            testChainId,
            "Recover didn't work for chain ID"
        );
        assertEq(
            proxied.evmChainId(),
            testEvmChainId,
            "Recover didn't work for evm chain ID"
        );
    }

    function testShouldAcceptSmartContractUpgradesAfterChainIdHasBeenRecovered()
        public
    {
        uint32 timestamp = 1000;
        uint32 nonce = 1001;

        // Perform a successful upgrade
        MockImplementation mock = new MockImplementation();

        bytes memory data = abi.encodePacked(
            core,
            actionContractUpgrade,
            testChainId,
            addressToBytes32(address(mock))
        );
        bytes memory vaa = signAndEncodeVM(
            timestamp,
            nonce,
            governanceChainId,
            governanceContract,
            0,
            data,
            uint256Array(testGuardian),
            0,
            2
        );

        bytes32 IMPLEMENTATION_STORAGE_SLOT = 0x360894a13ba1a3210667c828492db98dca3e2076cc3735a920a3ca505d382bbc;
        bytes32 before = vm.load(address(proxied), IMPLEMENTATION_STORAGE_SLOT);

        proxied.submitContractUpgrade(vaa);

        bytes32 afterUpgrade = vm.load(
            address(proxied),
            IMPLEMENTATION_STORAGE_SLOT
        );
        assertEq(afterUpgrade, addressToBytes32(address(mock)));
        assertEq(
            MockImplementation(payable(address(proxied)))
                .testNewImplementationActive(),
            true,
            "New implementation not active"
        );

        // Overwrite EVM Chain ID
        MockImplementation(payable(address(proxied))).testOverwriteEVMChainId(
            fakeChainId,
            fakeEvmChainId
        );
        assertEq(
            proxied.chainId(),
            fakeChainId,
            "Overwrite didn't work for chain ID"
        );
        assertEq(
            proxied.evmChainId(),
            fakeEvmChainId,
            "Overwrite didn't work for evm chain ID"
        );

        // recover chain ID
        data = abi.encodePacked(
            core,
            actionRecoverChainId,
            testEvmChainId,
            testChainId
        );
        vaa = signAndEncodeVM(
            timestamp,
            nonce,
            governanceChainId,
            governanceContract,
            0,
            data,
            uint256Array(testGuardian),
            0,
            2
        );

        proxied.submitRecoverChainId(vaa);

        assertEq(
            proxied.chainId(),
            testChainId,
            "Recover didn't work for chain ID"
        );
        assertEq(
            proxied.evmChainId(),
            testEvmChainId,
            "Recover didn't work for evm chain ID"
        );

        // Perform a successful upgrade
        mock = new MockImplementation();

        data = abi.encodePacked(
            core,
            actionContractUpgrade,
            testChainId,
            addressToBytes32(address(mock))
        );
        vaa = signAndEncodeVM(
            timestamp,
            nonce,
            governanceChainId,
            governanceContract,
            0,
            data,
            uint256Array(testGuardian),
            0,
            2
        );

        before = vm.load(address(proxied), IMPLEMENTATION_STORAGE_SLOT);

        proxied.submitContractUpgrade(vaa);

        afterUpgrade = vm.load(address(proxied), IMPLEMENTATION_STORAGE_SLOT);
        assertEq(afterUpgrade, addressToBytes32(address(mock)));
        assertEq(
            MockImplementation(payable(address(proxied)))
                .testNewImplementationActive(),
            true,
            "New implementation not active"
        );
    }
}
