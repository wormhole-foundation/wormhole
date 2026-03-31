// SPDX-License-Identifier: Apache 2
pragma solidity ^0.8.0;

import "../contracts/Implementation.sol";
import "../contracts/Setup.sol";
import "../contracts/Wormhole.sol";
import "../contracts/delegated_manager_set/DelegatedManagerSet.sol";
import "forge-std/Test.sol";
import "forge-test/rv-helpers/TestUtils.sol";

contract TestDelegatedManagerSet is TestUtils {
    uint16 constant CHAINID = 2;
    uint256 constant EVMCHAINID = 1;
    bytes32 constant CORE_MODULE = 0x00000000000000000000000000000000000000000000000000000000436f7265;
    // "DelegatedManager" left-padded
    bytes32 constant DELEGATED_MANAGER_MODULE = 0x0000000000000000000000000000000044656C6567617465644D616E61676572;
    bytes32 constant governanceContract = 0x0000000000000000000000000000000000000000000000000000000000000004;

    Wormhole proxy;
    Implementation impl;
    Setup setup;
    IWormhole wormhole;
    DelegatedManagerSet delegatedManagerSet;

    uint256 constant testGuardian = 93941733246223705020089879371323733820373732307041878556247502674739205313440;

    function setUp() public {
        // Deploy Wormhole
        setup = new Setup();
        impl = new Implementation();
        proxy = new Wormhole(address(setup), bytes(""));

        address[] memory keys = new address[](1);
        keys[0] = 0xbeFA429d57cD18b7F8A4d91A2da9AB4AF05d0FBe; // vm.addr(testGuardian)

        Setup proxiedSetup = Setup(address(proxy));

        vm.chainId(EVMCHAINID);
        proxiedSetup.setup({
            implementation: address(impl),
            initialGuardians: keys,
            chainId: CHAINID,
            governanceChainId: 1,
            governanceContract: governanceContract,
            evmChainId: EVMCHAINID
        });

        wormhole = IWormhole(address(proxy));

        // Deploy DelegatedManagerSet
        delegatedManagerSet = new DelegatedManagerSet(address(wormhole));
    }

    // ==================== Helper Functions ====================

    function createManagerSetUpdatePayload(
        bytes32 module,
        uint8 action,
        uint16 chainId,
        uint16 managerChainId,
        uint32 managerSetIndex,
        bytes memory managerSet
    ) internal pure returns (bytes memory) {
        return abi.encodePacked(
            module,
            action,
            chainId,
            managerChainId,
            managerSetIndex,
            managerSet
        );
    }

    function createSignedVAA(
        bytes memory payload
    ) internal view returns (bytes memory) {
        (bytes memory _vm, ) = validVm(
            0,          // guardianSetIndex
            uint32(block.timestamp),
            0,          // nonce
            1,          // emitterChainId (governance chain)
            governanceContract,
            0,          // sequence
            0,          // consistencyLevel
            payload,
            testGuardian
        );
        return _vm;
    }

    function createSignedVAAWithParams(
        uint32 guardianSetIndex,
        uint16 emitterChainId,
        bytes32 emitterAddress,
        bytes memory payload
    ) internal view returns (bytes memory) {
        (bytes memory _vm, ) = validVm(
            guardianSetIndex,
            uint32(block.timestamp),
            0,          // nonce
            emitterChainId,
            emitterAddress,
            0,          // sequence
            0,          // consistencyLevel
            payload,
            testGuardian
        );
        return _vm;
    }

    // ==================== Constructor Tests ====================

    function testConstructor() public {
        assertEq(address(delegatedManagerSet.WORMHOLE()), address(wormhole));
        assertEq(delegatedManagerSet.VERSION(), "DelegatedManagerSet-0.0.1");
    }

    // ==================== parseManagerSetUpdate Tests ====================

    function testParseManagerSetUpdate() public {
        bytes memory managerSet = hex"0102030405";
        bytes memory payload = createManagerSetUpdatePayload(
            DELEGATED_MANAGER_MODULE,
            1,      // action
            CHAINID,
            5,      // managerChainId
            1,      // managerSetIndex
            managerSet
        );

        IDelegatedManagerSet.ManagerSetUpdate memory update = delegatedManagerSet.parseManagerSetUpdate(payload);

        assertEq(update.module, DELEGATED_MANAGER_MODULE);
        assertEq(update.action, 1);
        assertEq(update.chainId, CHAINID);
        assertEq(update.managerChainId, 5);
        assertEq(update.managerSetIndex, 1);
        assertEq(update.managerSet, managerSet);
    }

    function testParseManagerSetUpdate_EmptyManagerSet() public {
        bytes memory payload = createManagerSetUpdatePayload(
            DELEGATED_MANAGER_MODULE,
            1,
            CHAINID,
            5,
            1,
            bytes("")
        );

        IDelegatedManagerSet.ManagerSetUpdate memory update = delegatedManagerSet.parseManagerSetUpdate(payload);
        assertEq(update.managerSet.length, 0);
    }

    function testParseManagerSetUpdate_Revert_InvalidModule() public {
        bytes32 invalidModule = bytes32(uint256(0x1234));
        bytes memory payload = createManagerSetUpdatePayload(
            invalidModule,
            1,
            CHAINID,
            5,
            1,
            hex"0102"
        );

        vm.expectRevert(DelegatedManagerSet.InvalidModule.selector);
        delegatedManagerSet.parseManagerSetUpdate(payload);
    }

    function testParseManagerSetUpdate_Revert_InvalidAction() public {
        bytes memory payload = createManagerSetUpdatePayload(
            DELEGATED_MANAGER_MODULE,
            2,      // invalid action (should be 1)
            CHAINID,
            5,
            1,
            hex"0102"
        );

        vm.expectRevert(DelegatedManagerSet.InvalidAction.selector);
        delegatedManagerSet.parseManagerSetUpdate(payload);
    }

    // ==================== submitNewManagerSet Tests ====================

    function testSubmitNewManagerSet() public {
        bytes memory managerSet = hex"0102030405060708091011121314151617181920";
        bytes memory payload = createManagerSetUpdatePayload(
            DELEGATED_MANAGER_MODULE,
            1,
            CHAINID,    // target this chain
            5,          // managerChainId
            1,          // managerSetIndex (must be current + 1 = 0 + 1 = 1)
            managerSet
        );

        bytes memory signedVAA = createSignedVAA(payload);

        delegatedManagerSet.submitNewManagerSet(signedVAA);

        // Verify the manager set was stored
        assertEq(delegatedManagerSet.getCurrentManagerSetIndex(5), 1);
        assertEq(delegatedManagerSet.getManagerSet(5, 1), managerSet);
        assertEq(delegatedManagerSet.getCurrentManagerSet(5), managerSet);
    }

    function testSubmitNewManagerSet_ChainIdZero() public {
        bytes memory managerSet = hex"aabbccdd";
        bytes memory payload = createManagerSetUpdatePayload(
            DELEGATED_MANAGER_MODULE,
            1,
            0,          // chainId = 0 means all chains
            7,          // managerChainId
            1,          // managerSetIndex
            managerSet
        );

        bytes memory signedVAA = createSignedVAA(payload);

        delegatedManagerSet.submitNewManagerSet(signedVAA);

        assertEq(delegatedManagerSet.getCurrentManagerSetIndex(7), 1);
        assertEq(delegatedManagerSet.getManagerSet(7, 1), managerSet);
    }

    function testSubmitNewManagerSet_MultipleUpdates() public {
        // First update: index 1
        bytes memory managerSet1 = hex"1111";
        bytes memory payload1 = createManagerSetUpdatePayload(
            DELEGATED_MANAGER_MODULE,
            1,
            CHAINID,
            10,     // managerChainId
            1,      // managerSetIndex
            managerSet1
        );
        delegatedManagerSet.submitNewManagerSet(createSignedVAA(payload1));

        // Second update: index 2
        bytes memory managerSet2 = hex"2222";
        bytes memory payload2 = createManagerSetUpdatePayload(
            DELEGATED_MANAGER_MODULE,
            1,
            CHAINID,
            10,     // same managerChainId
            2,      // managerSetIndex (must be 1 + 1 = 2)
            managerSet2
        );

        // Need a different VAA (different sequence/nonce to avoid replay)
        (bytes memory signedVAA2, ) = validVm(
            0,
            uint32(block.timestamp),
            1,      // different nonce
            1,
            governanceContract,
            1,      // different sequence
            0,
            payload2,
            testGuardian
        );

        delegatedManagerSet.submitNewManagerSet(signedVAA2);

        assertEq(delegatedManagerSet.getCurrentManagerSetIndex(10), 2);
        assertEq(delegatedManagerSet.getManagerSet(10, 1), managerSet1);
        assertEq(delegatedManagerSet.getManagerSet(10, 2), managerSet2);
        assertEq(delegatedManagerSet.getCurrentManagerSet(10), managerSet2);
    }

    function testSubmitNewManagerSet_Revert_InvalidChain() public {
        bytes memory payload = createManagerSetUpdatePayload(
            DELEGATED_MANAGER_MODULE,
            1,
            99,     // wrong chainId (not CHAINID and not 0)
            5,
            1,
            hex"0102"
        );

        bytes memory signedVAA = createSignedVAA(payload);

        vm.expectRevert(DelegatedManagerSet.InvalidChain.selector);
        delegatedManagerSet.submitNewManagerSet(signedVAA);
    }

    function testSubmitNewManagerSet_Revert_InvalidIndex_TooHigh() public {
        bytes memory payload = createManagerSetUpdatePayload(
            DELEGATED_MANAGER_MODULE,
            1,
            CHAINID,
            5,
            2,      // invalid: should be 1 (current 0 + 1)
            hex"0102"
        );

        bytes memory signedVAA = createSignedVAA(payload);

        vm.expectRevert(DelegatedManagerSet.InvalidIndex.selector);
        delegatedManagerSet.submitNewManagerSet(signedVAA);
    }

    function testSubmitNewManagerSet_Revert_InvalidIndex_Zero() public {
        bytes memory payload = createManagerSetUpdatePayload(
            DELEGATED_MANAGER_MODULE,
            1,
            CHAINID,
            5,
            0,      // invalid: should be 1 (current 0 + 1)
            hex"0102"
        );

        bytes memory signedVAA = createSignedVAA(payload);

        vm.expectRevert(DelegatedManagerSet.InvalidIndex.selector);
        delegatedManagerSet.submitNewManagerSet(signedVAA);
    }

    function testSubmitNewManagerSet_Revert_AlreadyConsumed() public {
        bytes memory managerSet = hex"0102030405";
        bytes memory payload = createManagerSetUpdatePayload(
            DELEGATED_MANAGER_MODULE,
            1,
            CHAINID,
            5,
            1,
            managerSet
        );

        bytes memory signedVAA = createSignedVAA(payload);

        // First submission should succeed
        delegatedManagerSet.submitNewManagerSet(signedVAA);

        // Second submission with same VAA should fail
        vm.expectRevert(DelegatedManagerSet.AlreadyConsumed.selector);
        delegatedManagerSet.submitNewManagerSet(signedVAA);
    }

    function testSubmitNewManagerSet_Revert_InvalidGuardianSet_NonExistent() public {
        bytes memory payload = createManagerSetUpdatePayload(
            DELEGATED_MANAGER_MODULE,
            1,
            CHAINID,
            5,
            1,
            hex"0102"
        );

        // Create VAA with wrong guardian set index (non-existent)
        bytes memory signedVAA = createSignedVAAWithParams(
            99,     // invalid guardian set index
            1,
            governanceContract,
            payload
        );

        vm.expectRevert(abi.encodeWithSelector(DelegatedManagerSet.InvalidVAA.selector, "invalid guardian set"));
        delegatedManagerSet.submitNewManagerSet(signedVAA);
    }

    function testSubmitNewManagerSet_Revert_InvalidGuardianSet_NotCurrent() public {
        // First, upgrade the guardian set so we have index 0 (old) and index 1 (current)
        // The old guardian set is still valid for 24 hours but shouldn't be accepted for governance

        address[] memory newGuardians = new address[](1);
        newGuardians[0] = vm.addr(12345); // new guardian with different key

        // Create guardian set upgrade payload (Core module action 2)
        bytes memory gsPayload = abi.encodePacked(
            CORE_MODULE,
            uint8(2),       // action: guardian set upgrade
            CHAINID,
            uint32(1),      // new guardian set index
            uint8(1),       // number of guardians
            newGuardians[0]
        );

        bytes memory gsVAA = createSignedVAA(gsPayload);
        wormhole.submitNewGuardianSet(gsVAA);

        // Verify guardian set was upgraded
        assertEq(wormhole.getCurrentGuardianSetIndex(), 1);

        // Now create a DelegatedManagerSet payload signed with OLD guardian set (index 0)
        // The old guardian set is still valid (not expired) but not current
        bytes memory payload = createManagerSetUpdatePayload(
            DELEGATED_MANAGER_MODULE,
            1,
            CHAINID,
            5,
            1,
            hex"0102"
        );

        // Sign with guardian set index 0 (old but still valid for 24h)
        bytes memory signedVAA = createSignedVAAWithParams(
            0,      // old guardian set index
            1,
            governanceContract,
            payload
        );

        // This should fail with InvalidGuardianSet because the VAA is valid
        // (passes verifyVM) but not signed by the CURRENT guardian set
        vm.expectRevert(DelegatedManagerSet.InvalidGuardianSet.selector);
        delegatedManagerSet.submitNewManagerSet(signedVAA);
    }

    function testSubmitNewManagerSet_Revert_InvalidGovernanceChain() public {
        bytes memory payload = createManagerSetUpdatePayload(
            DELEGATED_MANAGER_MODULE,
            1,
            CHAINID,
            5,
            1,
            hex"0102"
        );

        // Create VAA with wrong emitter chain
        bytes memory signedVAA = createSignedVAAWithParams(
            0,
            99,     // wrong governance chain
            governanceContract,
            payload
        );

        vm.expectRevert(DelegatedManagerSet.InvalidGovernanceChain.selector);
        delegatedManagerSet.submitNewManagerSet(signedVAA);
    }

    function testSubmitNewManagerSet_Revert_InvalidGovernanceContract() public {
        bytes memory payload = createManagerSetUpdatePayload(
            DELEGATED_MANAGER_MODULE,
            1,
            CHAINID,
            5,
            1,
            hex"0102"
        );

        // Create VAA with wrong emitter address
        bytes memory signedVAA = createSignedVAAWithParams(
            0,
            1,
            bytes32(uint256(0xdead)),   // wrong governance contract
            payload
        );

        vm.expectRevert(DelegatedManagerSet.InvalidGovernanceContract.selector);
        delegatedManagerSet.submitNewManagerSet(signedVAA);
    }

    function testSubmitNewManagerSet_Revert_InvalidVAA() public {
        // Create an invalid/malformed VAA
        bytes memory invalidVAA = hex"00010203";

        vm.expectRevert();
        delegatedManagerSet.submitNewManagerSet(invalidVAA);
    }

    // ==================== View Function Tests ====================

    function testGetManagerSet_NonExistent() public {
        bytes memory result = delegatedManagerSet.getManagerSet(999, 1);
        assertEq(result.length, 0);
    }

    function testGetCurrentManagerSetIndex_NonExistent() public {
        uint32 result = delegatedManagerSet.getCurrentManagerSetIndex(999);
        assertEq(result, 0);
    }

    function testGetCurrentManagerSet_NonExistent() public {
        bytes memory result = delegatedManagerSet.getCurrentManagerSet(999);
        assertEq(result.length, 0);
    }

    // ==================== Fuzz Tests ====================

    function testFuzz_ParseManagerSetUpdate(
        uint16 chainId,
        uint16 managerChainId,
        uint32 managerSetIndex,
        bytes calldata managerSet
    ) public {
        bytes memory payload = createManagerSetUpdatePayload(
            DELEGATED_MANAGER_MODULE,
            1,
            chainId,
            managerChainId,
            managerSetIndex,
            managerSet
        );

        IDelegatedManagerSet.ManagerSetUpdate memory update = delegatedManagerSet.parseManagerSetUpdate(payload);

        assertEq(update.module, DELEGATED_MANAGER_MODULE);
        assertEq(update.action, 1);
        assertEq(update.chainId, chainId);
        assertEq(update.managerChainId, managerChainId);
        assertEq(update.managerSetIndex, managerSetIndex);
        assertEq(update.managerSet, managerSet);
    }

    function testFuzz_SubmitNewManagerSet(
        uint16 managerChainId,
        bytes calldata managerSet
    ) public {
        bytes memory payload = createManagerSetUpdatePayload(
            DELEGATED_MANAGER_MODULE,
            1,
            CHAINID,
            managerChainId,
            1,
            managerSet
        );

        bytes memory signedVAA = createSignedVAA(payload);

        delegatedManagerSet.submitNewManagerSet(signedVAA);

        assertEq(delegatedManagerSet.getCurrentManagerSetIndex(managerChainId), 1);
        assertEq(delegatedManagerSet.getManagerSet(managerChainId, 1), managerSet);
        assertEq(delegatedManagerSet.getCurrentManagerSet(managerChainId), managerSet);
    }
}
