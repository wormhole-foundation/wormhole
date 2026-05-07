// SPDX-License-Identifier: Apache 2
pragma solidity ^0.8.0;

import "../contracts/Implementation.sol";
import "../contracts/Setup.sol";
import "../contracts/Wormhole.sol";
import "../contracts/wormhole_pauser/WormholePauser.sol";
import "../contracts/wormhole_pauser/interfaces/IWormholePauser.sol";
import "forge-std/Test.sol";
import "forge-test/rv-helpers/TestUtils.sol";

/// @dev Mock target used to verify proposal execution. The pause selector must match what we expect
///      a real protocol's pause() to look like.
contract MockPausable {
    bool public paused;
    address public lastCaller;
    bytes public lastPayload;
    bool public shouldRevert;

    function pause() external {
        if (shouldRevert) revert("forced revert");
        lastCaller = msg.sender;
        paused = true;
    }

    function setShouldRevert(bool b) external {
        shouldRevert = b;
    }

    /// @dev Echoes a payload so we can assert the call was forwarded as-is.
    function echo(bytes calldata payload) external {
        lastPayload = payload;
        lastCaller = msg.sender;
    }
}

contract TestWormholePauser is TestUtils {
    uint16 constant CHAINID = 2;
    uint256 constant EVMCHAINID = 1;
    bytes32 constant CORE_MODULE =
        0x00000000000000000000000000000000000000000000000000000000436f7265;
    // "DelegatedPauser" left-padded
    bytes32 constant DELEGATED_PAUSER_MODULE =
        0x000000000000000000000000000000000044656C656761746564506175736572;
    bytes32 constant governanceContract =
        0x0000000000000000000000000000000000000000000000000000000000000004;
    uint8 constant SET_CONFIG_EVM_ACTION = 1;
    uint8 constant SET_CONFIG_SOLANA_ACTION = 2;

    Wormhole proxy;
    Implementation impl;
    Setup setup;
    IWormhole wormhole;
    WormholePauser pauser;
    MockPausable target;

    uint256 constant testGuardian =
        93941733246223705020089879371323733820373732307041878556247502674739205313440;
    address signerA = address(0xA1);
    address signerB = address(0xB2);
    address signerC = address(0xC3);
    address outsider = address(0xDEAD);

    function setUp() public {
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
        pauser = new WormholePauser(address(wormhole));
        target = new MockPausable();
    }

    // ============================ Helpers ============================

    function _encodeSetConfigPayload(
        bytes32 module,
        uint8 action,
        uint16 chain,
        uint16 index,
        uint8 threshold,
        uint64 expiryDuration,
        address[] memory signers
    ) internal pure returns (bytes memory) {
        bytes memory body = abi.encodePacked(index, threshold, expiryDuration, uint8(signers.length));
        for (uint256 i = 0; i < signers.length; i++) {
            body = abi.encodePacked(body, signers[i]);
        }
        return abi.encodePacked(module, action, chain, body);
    }

    function _signedVAA(bytes memory payload, uint64 sequence) internal view returns (bytes memory) {
        (bytes memory _vm, ) = validVm(
            0,
            uint32(block.timestamp),
            0,
            1,
            governanceContract,
            sequence,
            0,
            payload,
            testGuardian
        );
        return _vm;
    }

    function _signedVAAWithParams(
        uint32 guardianSetIndex,
        uint16 emitterChainId,
        bytes32 emitterAddress,
        bytes memory payload
    ) internal view returns (bytes memory) {
        (bytes memory _vm, ) = validVm(
            guardianSetIndex,
            uint32(block.timestamp),
            0,
            emitterChainId,
            emitterAddress,
            0,
            0,
            payload,
            testGuardian
        );
        return _vm;
    }

    function _defaultSigners() internal view returns (address[] memory s) {
        s = new address[](3);
        s[0] = signerA;
        s[1] = signerB;
        s[2] = signerC;
    }

    /// @dev Apply a default config (3 signers, threshold 2, expiry 1 hour). Used by most tests.
    function _applyDefaultConfig() internal {
        bytes memory payload = _encodeSetConfigPayload(
            DELEGATED_PAUSER_MODULE,
            SET_CONFIG_EVM_ACTION,
            CHAINID,
            1, // index
            2, // threshold
            3600, // expiry
            _defaultSigners()
        );
        pauser.submitConfig(_signedVAA(payload, 0));
    }

    function _pauseCalldata() internal pure returns (bytes memory) {
        return abi.encodeWithSignature("pause()");
    }

    // ============================ Constructor ============================

    function testConstructor() public {
        assertEq(address(pauser.WORMHOLE()), address(wormhole));
        assertEq(pauser.VERSION(), "WormholePauser-0.0.1");
        assertEq(pauser.configIndex(), 0);
        assertEq(pauser.threshold(), 0);
        assertEq(pauser.expiryDuration(), 0);
        assertEq(pauser.nextProposalId(), 0);
    }

    // ============================ submitConfig ============================

    function testSubmitConfig_Success() public {
        _applyDefaultConfig();

        assertEq(pauser.configIndex(), 1);
        assertEq(pauser.threshold(), 2);
        assertEq(pauser.expiryDuration(), 3600);
        assertTrue(pauser.isSigner(signerA));
        assertTrue(pauser.isSigner(signerB));
        assertTrue(pauser.isSigner(signerC));
        assertFalse(pauser.isSigner(outsider));
    }

    function testSubmitConfig_Revert_InvalidModule() public {
        bytes32 wrong = bytes32(uint256(0xdead));
        bytes memory payload = _encodeSetConfigPayload(wrong, 1, CHAINID, 1, 1, 1, _defaultSigners());
        bytes memory vaa = _signedVAA(payload, 0);
        vm.expectRevert(WormholePauser.InvalidModule.selector);
        pauser.submitConfig(vaa);
    }

    function testSubmitConfig_Revert_InvalidAction_Solana() public {
        bytes memory payload = _encodeSetConfigPayload(
            DELEGATED_PAUSER_MODULE,
            SET_CONFIG_SOLANA_ACTION, // not allowed on EVM
            CHAINID,
            1,
            1,
            1,
            _defaultSigners()
        );
        vm.expectRevert(WormholePauser.InvalidAction.selector);
        pauser.submitConfig(_signedVAA(payload, 0));
    }

    function testSubmitConfig_Revert_InvalidChain() public {
        bytes memory payload = _encodeSetConfigPayload(
            DELEGATED_PAUSER_MODULE,
            SET_CONFIG_EVM_ACTION,
            99, // wrong chain
            1,
            1,
            1,
            _defaultSigners()
        );
        vm.expectRevert(WormholePauser.InvalidChain.selector);
        pauser.submitConfig(_signedVAA(payload, 0));
    }

    function testSubmitConfig_Revert_InvalidIndex_TooHigh() public {
        bytes memory payload = _encodeSetConfigPayload(
            DELEGATED_PAUSER_MODULE,
            SET_CONFIG_EVM_ACTION,
            CHAINID,
            2, // initial must be 1
            1,
            1,
            _defaultSigners()
        );
        vm.expectRevert(WormholePauser.InvalidIndex.selector);
        pauser.submitConfig(_signedVAA(payload, 0));
    }

    function testSubmitConfig_Revert_InvalidIndex_Zero() public {
        bytes memory payload = _encodeSetConfigPayload(
            DELEGATED_PAUSER_MODULE,
            SET_CONFIG_EVM_ACTION,
            CHAINID,
            0,
            1,
            1,
            _defaultSigners()
        );
        vm.expectRevert(WormholePauser.InvalidIndex.selector);
        pauser.submitConfig(_signedVAA(payload, 0));
    }

    function testSubmitConfig_Revert_InvalidThreshold_Zero() public {
        bytes memory payload = _encodeSetConfigPayload(
            DELEGATED_PAUSER_MODULE,
            SET_CONFIG_EVM_ACTION,
            CHAINID,
            1,
            0, // threshold == 0
            1,
            _defaultSigners()
        );
        vm.expectRevert(WormholePauser.InvalidThreshold.selector);
        pauser.submitConfig(_signedVAA(payload, 0));
    }

    function testSubmitConfig_Revert_InvalidThreshold_GreaterThanSigners() public {
        bytes memory payload = _encodeSetConfigPayload(
            DELEGATED_PAUSER_MODULE,
            SET_CONFIG_EVM_ACTION,
            CHAINID,
            1,
            5, // threshold > 3
            1,
            _defaultSigners()
        );
        vm.expectRevert(WormholePauser.InvalidThreshold.selector);
        pauser.submitConfig(_signedVAA(payload, 0));
    }

    function testSubmitConfig_Revert_InvalidExpiryDuration() public {
        bytes memory payload = _encodeSetConfigPayload(
            DELEGATED_PAUSER_MODULE,
            SET_CONFIG_EVM_ACTION,
            CHAINID,
            1,
            1,
            0, // expiry == 0
            _defaultSigners()
        );
        vm.expectRevert(WormholePauser.InvalidExpiryDuration.selector);
        pauser.submitConfig(_signedVAA(payload, 0));
    }

    function testSubmitConfig_Revert_EmptySignerSet() public {
        address[] memory s = new address[](0);
        bytes memory payload = _encodeSetConfigPayload(
            DELEGATED_PAUSER_MODULE,
            SET_CONFIG_EVM_ACTION,
            CHAINID,
            1,
            1,
            1,
            s
        );
        vm.expectRevert(WormholePauser.EmptySignerSet.selector);
        pauser.submitConfig(_signedVAA(payload, 0));
    }

    function testSubmitConfig_Revert_ZeroSigner() public {
        address[] memory s = new address[](2);
        s[0] = signerA;
        s[1] = address(0);
        bytes memory payload = _encodeSetConfigPayload(
            DELEGATED_PAUSER_MODULE,
            SET_CONFIG_EVM_ACTION,
            CHAINID,
            1,
            1,
            1,
            s
        );
        vm.expectRevert(WormholePauser.ZeroSigner.selector);
        pauser.submitConfig(_signedVAA(payload, 0));
    }

    function testSubmitConfig_Revert_DuplicateSigner() public {
        address[] memory s = new address[](2);
        s[0] = signerA;
        s[1] = signerA;
        bytes memory payload = _encodeSetConfigPayload(
            DELEGATED_PAUSER_MODULE,
            SET_CONFIG_EVM_ACTION,
            CHAINID,
            1,
            1,
            1,
            s
        );
        vm.expectRevert(WormholePauser.DuplicateSigner.selector);
        pauser.submitConfig(_signedVAA(payload, 0));
    }

    function testSubmitConfig_Revert_InvalidPayloadLength() public {
        // Construct a payload with one extra trailing byte after the signer list.
        bytes memory body = abi.encodePacked(uint16(1), uint8(1), uint64(1), uint8(1), signerA, hex"ff");
        bytes memory payload = abi.encodePacked(
            DELEGATED_PAUSER_MODULE,
            SET_CONFIG_EVM_ACTION,
            CHAINID,
            body
        );
        vm.expectRevert(WormholePauser.InvalidPayloadLength.selector);
        pauser.submitConfig(_signedVAA(payload, 0));
    }

    function testSubmitConfig_Revert_AlreadyConsumed() public {
        bytes memory payload = _encodeSetConfigPayload(
            DELEGATED_PAUSER_MODULE,
            SET_CONFIG_EVM_ACTION,
            CHAINID,
            1,
            2,
            3600,
            _defaultSigners()
        );
        bytes memory vaa = _signedVAA(payload, 0);
        pauser.submitConfig(vaa);
        vm.expectRevert(WormholePauser.AlreadyConsumed.selector);
        pauser.submitConfig(vaa);
    }

    function testSubmitConfig_Revert_InvalidGovernanceChain() public {
        bytes memory payload = _encodeSetConfigPayload(
            DELEGATED_PAUSER_MODULE,
            SET_CONFIG_EVM_ACTION,
            CHAINID,
            1,
            1,
            1,
            _defaultSigners()
        );
        bytes memory vaa = _signedVAAWithParams(0, 99, governanceContract, payload);
        vm.expectRevert(WormholePauser.InvalidGovernanceChain.selector);
        pauser.submitConfig(vaa);
    }

    function testSubmitConfig_Revert_InvalidGovernanceContract() public {
        bytes memory payload = _encodeSetConfigPayload(
            DELEGATED_PAUSER_MODULE,
            SET_CONFIG_EVM_ACTION,
            CHAINID,
            1,
            1,
            1,
            _defaultSigners()
        );
        bytes memory vaa = _signedVAAWithParams(0, 1, bytes32(uint256(0xdeadbeef)), payload);
        vm.expectRevert(WormholePauser.InvalidGovernanceContract.selector);
        pauser.submitConfig(vaa);
    }

    function testSubmitConfig_Revert_InvalidGuardianSet_NonExistent() public {
        bytes memory payload = _encodeSetConfigPayload(
            DELEGATED_PAUSER_MODULE,
            SET_CONFIG_EVM_ACTION,
            CHAINID,
            1,
            1,
            1,
            _defaultSigners()
        );
        bytes memory vaa = _signedVAAWithParams(99, 1, governanceContract, payload);
        vm.expectRevert(abi.encodeWithSelector(WormholePauser.InvalidVAA.selector, "invalid guardian set"));
        pauser.submitConfig(vaa);
    }

    function testSubmitConfig_Revert_InvalidGuardianSet_NotCurrent() public {
        // Roll the guardian set forward; the prior set is still valid (within 24h) but not current.
        address[] memory newGuardians = new address[](1);
        newGuardians[0] = vm.addr(12345);
        bytes memory gsPayload = abi.encodePacked(
            CORE_MODULE,
            uint8(2),
            uint16(0),
            uint32(1),
            uint8(1),
            newGuardians[0]
        );
        wormhole.submitNewGuardianSet(_signedVAA(gsPayload, 0));
        assertEq(wormhole.getCurrentGuardianSetIndex(), 1);

        bytes memory payload = _encodeSetConfigPayload(
            DELEGATED_PAUSER_MODULE,
            SET_CONFIG_EVM_ACTION,
            CHAINID,
            1,
            1,
            1,
            _defaultSigners()
        );
        bytes memory vaa = _signedVAAWithParams(0, 1, governanceContract, payload);
        vm.expectRevert(WormholePauser.InvalidGuardianSet.selector);
        pauser.submitConfig(vaa);
    }

    function testSubmitConfig_Revert_InvalidVAA() public {
        bytes memory invalidVaa = hex"00010203";
        vm.expectRevert();
        pauser.submitConfig(invalidVaa);
    }

    function testSubmitConfig_MultipleUpdates() public {
        _applyDefaultConfig();
        assertEq(pauser.configIndex(), 1);

        // Second update with index 2
        address[] memory s = new address[](1);
        s[0] = outsider;
        bytes memory payload = _encodeSetConfigPayload(
            DELEGATED_PAUSER_MODULE,
            SET_CONFIG_EVM_ACTION,
            CHAINID,
            2,
            1,
            60,
            s
        );
        (bytes memory vaa2, ) = validVm(0, uint32(block.timestamp), 1, 1, governanceContract, 1, 0, payload, testGuardian);
        pauser.submitConfig(vaa2);

        assertEq(pauser.configIndex(), 2);
        assertEq(pauser.threshold(), 1);
        assertEq(pauser.expiryDuration(), 60);
        assertTrue(pauser.isSigner(outsider));
        // Old signers are no longer in the current set.
        assertFalse(pauser.isSigner(signerA));
        assertFalse(pauser.isSigner(signerB));
        assertFalse(pauser.isSigner(signerC));
    }

    // ============================ propose / approve / execute ============================

    function testPropose_Revert_NotSigner() public {
        _applyDefaultConfig();
        vm.expectRevert(WormholePauser.NotSigner.selector);
        vm.prank(outsider);
        pauser.propose(address(target), _pauseCalldata());
    }

    function testPropose_AutoApproves_DoesNotExecute_BelowThreshold() public {
        _applyDefaultConfig(); // threshold = 2

        vm.prank(signerA);
        uint256 id = pauser.propose(address(target), _pauseCalldata());

        IWormholePauser.Proposal memory p = pauser.getProposal(id);
        assertTrue(p.exists);
        assertFalse(p.executed);
        assertEq(p.approvalCount, 1);
        assertEq(p.target, address(target));
        assertEq(p.configIndex, 1);
        assertTrue(pauser.hasApproved(id, signerA));
        assertFalse(target.paused());
    }

    function testPropose_ThresholdOne_ExecutesImmediately() public {
        // Threshold-1 config
        address[] memory s = new address[](1);
        s[0] = signerA;
        bytes memory payload = _encodeSetConfigPayload(
            DELEGATED_PAUSER_MODULE,
            SET_CONFIG_EVM_ACTION,
            CHAINID,
            1,
            1,
            3600,
            s
        );
        pauser.submitConfig(_signedVAA(payload, 0));

        vm.prank(signerA);
        uint256 id = pauser.propose(address(target), _pauseCalldata());

        IWormholePauser.Proposal memory p = pauser.getProposal(id);
        assertTrue(p.executed);
        assertTrue(target.paused());
        assertEq(target.lastCaller(), address(pauser));
    }

    function testApprove_ReachesThreshold_Executes() public {
        _applyDefaultConfig();

        vm.prank(signerA);
        uint256 id = pauser.propose(address(target), _pauseCalldata());

        vm.prank(signerB);
        pauser.approve(id);

        IWormholePauser.Proposal memory p = pauser.getProposal(id);
        assertTrue(p.executed);
        assertTrue(target.paused());
    }

    function testApprove_Revert_NotSigner() public {
        _applyDefaultConfig();
        vm.prank(signerA);
        uint256 id = pauser.propose(address(target), _pauseCalldata());

        vm.expectRevert(WormholePauser.NotSigner.selector);
        vm.prank(outsider);
        pauser.approve(id);
    }

    function testApprove_Revert_ProposalDoesNotExist() public {
        _applyDefaultConfig();
        vm.expectRevert(WormholePauser.ProposalDoesNotExist.selector);
        vm.prank(signerA);
        pauser.approve(999);
    }

    function testApprove_Revert_AlreadyApproved() public {
        _applyDefaultConfig();
        vm.prank(signerA);
        uint256 id = pauser.propose(address(target), _pauseCalldata());

        vm.expectRevert(WormholePauser.AlreadyApproved.selector);
        vm.prank(signerA);
        pauser.approve(id);
    }

    function testApprove_Revert_ProposalAlreadyExecuted() public {
        _applyDefaultConfig();
        vm.prank(signerA);
        uint256 id = pauser.propose(address(target), _pauseCalldata());
        vm.prank(signerB);
        pauser.approve(id);

        vm.expectRevert(WormholePauser.ProposalAlreadyExecuted.selector);
        vm.prank(signerC);
        pauser.approve(id);
    }

    function testApprove_Revert_ProposalExpired() public {
        _applyDefaultConfig();
        vm.prank(signerA);
        uint256 id = pauser.propose(address(target), _pauseCalldata());

        vm.warp(block.timestamp + 4000);
        vm.expectRevert(WormholePauser.ProposalExpired.selector);
        vm.prank(signerB);
        pauser.approve(id);
    }

    function testApprove_Revert_ProposalConfigRotated() public {
        _applyDefaultConfig();
        vm.prank(signerA);
        uint256 id = pauser.propose(address(target), _pauseCalldata());

        // Rotate config; the prior proposal is now stale.
        address[] memory s = new address[](1);
        s[0] = signerA;
        bytes memory payload = _encodeSetConfigPayload(
            DELEGATED_PAUSER_MODULE,
            SET_CONFIG_EVM_ACTION,
            CHAINID,
            2,
            1,
            3600,
            s
        );
        (bytes memory vaa2, ) = validVm(0, uint32(block.timestamp), 1, 1, governanceContract, 1, 0, payload, testGuardian);
        pauser.submitConfig(vaa2);

        vm.expectRevert(WormholePauser.ProposalConfigRotated.selector);
        vm.prank(signerA);
        pauser.approve(id);
    }

    function testApprove_ExecutionRevert_RollsBackEverything() public {
        _applyDefaultConfig();
        target.setShouldRevert(true);

        vm.prank(signerA);
        uint256 id = pauser.propose(address(target), _pauseCalldata());

        // The threshold-meeting approve should revert because target.pause() reverts.
        vm.prank(signerB);
        vm.expectRevert(); // ExecutionFailed with returnData; selector match is sufficient
        pauser.approve(id);

        // Approval count for signerB (the one whose tx reverted) is rolled back; signerA's persisted approval remains.
        IWormholePauser.Proposal memory p = pauser.getProposal(id);
        assertFalse(p.executed);
        assertEq(p.approvalCount, 1);
        assertTrue(pauser.hasApproved(id, signerA));
        assertFalse(pauser.hasApproved(id, signerB));

        // Fix the target and retry through signerB — proposal can recover.
        target.setShouldRevert(false);
        vm.prank(signerB);
        pauser.approve(id);
        IWormholePauser.Proposal memory p2 = pauser.getProposal(id);
        assertTrue(p2.executed);
        assertTrue(target.paused());
    }

    function testApprove_PayloadEchoedExactly() public {
        _applyDefaultConfig();
        bytes memory call = abi.encodeWithSignature("echo(bytes)", hex"deadbeef");

        vm.prank(signerA);
        uint256 id = pauser.propose(address(target), call);
        vm.prank(signerB);
        pauser.approve(id);

        assertEq(target.lastPayload(), hex"deadbeef");
        assertEq(target.lastCaller(), address(pauser));
    }

    // ============================ cancelApproval ============================

    function testCancelApproval_Success() public {
        _applyDefaultConfig();
        vm.prank(signerA);
        uint256 id = pauser.propose(address(target), _pauseCalldata());
        vm.prank(signerA);
        pauser.cancelApproval(id);

        IWormholePauser.Proposal memory p = pauser.getProposal(id);
        assertEq(p.approvalCount, 0);
        assertFalse(pauser.hasApproved(id, signerA));
    }

    function testCancelApproval_Revert_NotSigner() public {
        _applyDefaultConfig();
        vm.prank(signerA);
        uint256 id = pauser.propose(address(target), _pauseCalldata());
        vm.expectRevert(WormholePauser.NotSigner.selector);
        vm.prank(outsider);
        pauser.cancelApproval(id);
    }

    function testCancelApproval_Revert_NotApproved() public {
        _applyDefaultConfig();
        vm.prank(signerA);
        uint256 id = pauser.propose(address(target), _pauseCalldata());
        vm.expectRevert(WormholePauser.NotApproved.selector);
        vm.prank(signerB);
        pauser.cancelApproval(id);
    }

    function testCancelApproval_Revert_ProposalDoesNotExist() public {
        _applyDefaultConfig();
        vm.expectRevert(WormholePauser.ProposalDoesNotExist.selector);
        vm.prank(signerA);
        pauser.cancelApproval(999);
    }

    function testCancelApproval_Revert_ProposalAlreadyExecuted() public {
        _applyDefaultConfig();
        vm.prank(signerA);
        uint256 id = pauser.propose(address(target), _pauseCalldata());
        vm.prank(signerB);
        pauser.approve(id);

        vm.expectRevert(WormholePauser.ProposalAlreadyExecuted.selector);
        vm.prank(signerA);
        pauser.cancelApproval(id);
    }

    function testCancelApproval_Revert_ProposalExpired() public {
        _applyDefaultConfig();
        vm.prank(signerA);
        uint256 id = pauser.propose(address(target), _pauseCalldata());
        vm.warp(block.timestamp + 4000);
        vm.expectRevert(WormholePauser.ProposalExpired.selector);
        vm.prank(signerA);
        pauser.cancelApproval(id);
    }

    function testCancelApproval_Revert_ProposalConfigRotated() public {
        _applyDefaultConfig();
        vm.prank(signerA);
        uint256 id = pauser.propose(address(target), _pauseCalldata());

        address[] memory s = new address[](1);
        s[0] = signerA;
        bytes memory payload = _encodeSetConfigPayload(
            DELEGATED_PAUSER_MODULE,
            SET_CONFIG_EVM_ACTION,
            CHAINID,
            2,
            1,
            3600,
            s
        );
        (bytes memory vaa2, ) = validVm(0, uint32(block.timestamp), 1, 1, governanceContract, 1, 0, payload, testGuardian);
        pauser.submitConfig(vaa2);

        vm.expectRevert(WormholePauser.ProposalConfigRotated.selector);
        vm.prank(signerA);
        pauser.cancelApproval(id);
    }

    function testCancelApproval_AfterCancel_AnotherSignerCanReapproveAndExecute() public {
        _applyDefaultConfig();
        vm.prank(signerA);
        uint256 id = pauser.propose(address(target), _pauseCalldata());
        vm.prank(signerA);
        pauser.cancelApproval(id);

        vm.prank(signerB);
        pauser.approve(id);
        vm.prank(signerC);
        pauser.approve(id);

        assertTrue(target.paused());
    }

    // ============================ proposalId monotonicity ============================

    function testPropose_AssignsMonotonicIds() public {
        _applyDefaultConfig();
        vm.prank(signerA);
        uint256 id1 = pauser.propose(address(target), _pauseCalldata());
        vm.prank(signerB);
        uint256 id2 = pauser.propose(address(target), _pauseCalldata());

        assertEq(id1, 0);
        assertEq(id2, 1);
        assertEq(pauser.nextProposalId(), 2);
    }

    // ============================ View getters on missing data ============================

    function testGetProposal_NonExistent() public {
        IWormholePauser.Proposal memory p = pauser.getProposal(123);
        assertFalse(p.exists);
        assertEq(p.target, address(0));
    }

    function testHasApproved_NonExistent() public {
        assertFalse(pauser.hasApproved(123, signerA));
    }

    // ============================ Fuzz ============================

    function testFuzz_SubmitConfig(uint8 t, uint64 expiry, uint8 numSigners) public {
        vm.assume(numSigners > 0 && numSigners <= 50);
        vm.assume(t > 0 && t <= numSigners);
        vm.assume(expiry > 0);

        address[] memory s = new address[](numSigners);
        for (uint8 i = 0; i < numSigners; i++) {
            s[i] = address(uint160(i + 1)); // distinct, non-zero
        }
        bytes memory payload = _encodeSetConfigPayload(
            DELEGATED_PAUSER_MODULE,
            SET_CONFIG_EVM_ACTION,
            CHAINID,
            1,
            t,
            expiry,
            s
        );
        pauser.submitConfig(_signedVAA(payload, 0));
        assertEq(pauser.threshold(), t);
        assertEq(pauser.expiryDuration(), expiry);
    }
}
