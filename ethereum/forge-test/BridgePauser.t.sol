// SPDX-License-Identifier: Apache 2
pragma solidity ^0.8.0;

import "../contracts/bridge/BridgeSetup.sol";
import "../contracts/bridge/BridgeImplementation.sol";
import "../contracts/bridge/BridgePauserStorage.sol";
import "../contracts/bridge/TokenBridge.sol";
import "../contracts/bridge/interfaces/ITokenBridge.sol";
import "../contracts/bridge/token/TokenImplementation.sol";
import "../contracts/bridge/mock/MockWETH9.sol";
import "../contracts/interfaces/IWormhole.sol";
import "forge-std/Test.sol";
import "./Implementation.t.sol";

contract TestBridgePauser is Test {
    BridgeSetup bridgeSetup;
    BridgeImplementation bridgeImpl;
    ITokenBridge bridge;
    IWormhole wormhole;
    TestImplementation implementationTest;
    TokenImplementation tokenImpl;
    IERC20 weth;

    uint16 testChainId;
    uint256 testEvmChainId;
    uint16 governanceChainId;
    bytes32 governanceContract;
    uint8 constant finality = 15;

    // "TokenBridge" left-padded
    bytes32 constant tokenBridgeModule =
        0x000000000000000000000000000000000000000000546f6b656e427269646765;
    uint8 constant ACTION_SET_PAUSER_ADDRESSES = 4;
    uint8 constant EVM_ADDR_LEN = 20;

    address constant PAUSER = address(0xCAFE);
    address constant FREEZER = address(0xF00D);
    address constant UNPAUSER = address(0xBEEF);

    // 5 days, in seconds — matches `Bridge.PAUSE_DURATION`.
    uint64 constant PAUSE_DURATION = 5 days;
    uint64 constant MAX_TIMESTAMP = type(uint64).max;

    // Re-declared so vm.expectEmit can match (Solidity 0.8.4 doesn't allow IName.Event references).
    event PauserAddressesSet(address indexed pauser, address indexed freezer, address indexed unpauser);
    event Paused(address indexed by, uint256 pauseExpiry);
    event Frozen(address indexed by, uint256 pauseExpiry);
    event Unpaused(address indexed by);
    event UnpauseExpired(address indexed by);

    uint256 constant testGuardian =
        93941733246223705020089879371323733820373732307041878556247502674739205313440;

    function setUp() public {
        implementationTest = new TestImplementation();
        implementationTest.setUp();
        wormhole = IWormhole(address(implementationTest.proxied()));

        bridgeSetup = new BridgeSetup();
        bridgeImpl = new BridgeImplementation();
        tokenImpl = new TokenImplementation();
        weth = IERC20(address(new MockWETH9()));

        testChainId = implementationTest.testChainId();
        testEvmChainId = implementationTest.testEvmChainId();
        vm.chainId(testEvmChainId);
        governanceChainId = implementationTest.governanceChainId();
        governanceContract = implementationTest.governanceContract();

        bytes memory setupAbi = abi.encodeWithSelector(
            BridgeSetup.setup.selector,
            address(bridgeImpl),
            testChainId,
            address(wormhole),
            governanceChainId,
            governanceContract,
            address(tokenImpl),
            address(weth),
            finality,
            testEvmChainId
        );
        bridge = ITokenBridge(address(new TokenBridge(address(bridgeSetup), setupAbi)));

        // Start at a non-zero time so `pauseExpiry` arithmetic and `unpauseExpired` boundaries are
        // meaningful (a fresh foundry chain starts at block.timestamp == 1).
        vm.warp(1_000_000);
    }

    // ============================ Helpers ============================

    function _signAndEncodeVM(bytes memory data, uint64 sequence) internal pure returns (bytes memory) {
        return _signAndEncodeVMFrom(data, sequence, uint16(1), bytes32(uint256(0x4)));
    }

    /// @dev Variant that lets a test override the emitter chain id and emitter address — used to
    ///      exercise the `WrongGovernanceChain` and `WrongGovernanceContract` revert paths.
    function _signAndEncodeVMFrom(
        bytes memory data,
        uint64 sequence,
        uint16 emitterChainId,
        bytes32 emitterAddress
    ) internal pure returns (bytes memory) {
        bytes memory body = abi.encodePacked(
            uint32(0), uint32(0), emitterChainId, emitterAddress, sequence, uint8(0), data
        );
        bytes32 bodyHash = keccak256(abi.encodePacked(keccak256(body)));
        (uint8 v, bytes32 r, bytes32 s) = Vm(address(uint160(uint256(keccak256("hevm cheat code"))))).sign(testGuardian, bodyHash);
        bytes memory header = abi.encodePacked(uint8(1), uint32(0), uint8(1), uint8(0), r, s, v - 27);
        return abi.encodePacked(header, body);
    }

    /// @dev Encode a SetPauserAddresses payload using the length-prefixed wire format described in
    ///      whitepapers/0003_token_bridge.md. Three roles in wire order pauser, freezer, unpauser;
    ///      each address is preceded by its length (20 on EVM, or 0 to leave the role unassigned).
    function _setPauserAddressesPayload(
        uint16 chain_,
        address pauser_,
        address freezer_,
        address unpauser_
    ) internal pure returns (bytes memory) {
        return abi.encodePacked(
            tokenBridgeModule,
            ACTION_SET_PAUSER_ADDRESSES,
            chain_,
            EVM_ADDR_LEN,
            pauser_,
            EVM_ADDR_LEN,
            freezer_,
            EVM_ADDR_LEN,
            unpauser_
        );
    }

    /// @dev Encode a SetPauserAddresses payload where each of the three roles may independently be
    ///      marked as unassigned (length 0, zero-byte body).
    function _setPauserAddressesPayloadOptional(
        uint16 chain_,
        bool hasPauser,
        address pauser_,
        bool hasFreezer,
        address freezer_,
        bool hasUnpauser,
        address unpauser_
    ) internal pure returns (bytes memory) {
        bytes memory pauserField = hasPauser
            ? abi.encodePacked(EVM_ADDR_LEN, pauser_)
            : abi.encodePacked(uint8(0));
        bytes memory freezerField = hasFreezer
            ? abi.encodePacked(EVM_ADDR_LEN, freezer_)
            : abi.encodePacked(uint8(0));
        bytes memory unpauserField = hasUnpauser
            ? abi.encodePacked(EVM_ADDR_LEN, unpauser_)
            : abi.encodePacked(uint8(0));
        return abi.encodePacked(
            tokenBridgeModule,
            ACTION_SET_PAUSER_ADDRESSES,
            chain_,
            pauserField,
            freezerField,
            unpauserField
        );
    }

    // ============================ submitSetPauserAddresses ============================

    function testSubmitSetPauserAddresses_Success() public {
        bytes memory payload = _setPauserAddressesPayload(testChainId, PAUSER, FREEZER, UNPAUSER);
        bytes memory vaa = _signAndEncodeVM(payload, 0);

        vm.expectEmit(true, true, true, true);
        emit PauserAddressesSet(PAUSER, FREEZER, UNPAUSER);
        bridge.submitSetPauserAddresses(vaa);

        assertEq(bridge.pauser(), PAUSER);
        assertEq(bridge.freezer(), FREEZER);
        assertEq(bridge.unpauser(), UNPAUSER);
        assertFalse(bridge.paused());
    }

    function testSubmitSetPauserAddresses_Revert_WrongChain() public {
        bytes memory payload = _setPauserAddressesPayload(uint16(99), PAUSER, FREEZER, UNPAUSER);
        vm.expectRevert(ITokenBridge.WrongChainId.selector);
        bridge.submitSetPauserAddresses(_signAndEncodeVM(payload, 0));
    }

    function testSubmitSetPauserAddresses_Revert_UnknownAction() public {
        // Action 5 is not defined for `SetPauserAddresses`; the single action 4 is length-prefixed
        // and covers every runtime. Any other action must be rejected.
        bytes memory payload = abi.encodePacked(
            tokenBridgeModule,
            uint8(5),
            testChainId,
            EVM_ADDR_LEN,
            PAUSER,
            EVM_ADDR_LEN,
            FREEZER,
            EVM_ADDR_LEN,
            UNPAUSER
        );
        vm.expectRevert(ITokenBridge.WrongAction.selector);
        bridge.submitSetPauserAddresses(_signAndEncodeVM(payload, 0));
    }

    function testSubmitSetPauserAddresses_Revert_WrongModule() public {
        bytes memory payload = abi.encodePacked(
            bytes32(uint256(0xdeadbeef)),
            ACTION_SET_PAUSER_ADDRESSES,
            testChainId,
            EVM_ADDR_LEN,
            PAUSER,
            EVM_ADDR_LEN,
            FREEZER,
            EVM_ADDR_LEN,
            UNPAUSER
        );
        vm.expectRevert(ITokenBridge.WrongModule.selector);
        bridge.submitSetPauserAddresses(_signAndEncodeVM(payload, 0));
    }

    function testSubmitSetPauserAddresses_Revert_InvalidPauserLength() public {
        // 32 is the Solana native size but must be rejected on EVM (native size = 20).
        bytes memory payload = abi.encodePacked(
            tokenBridgeModule,
            ACTION_SET_PAUSER_ADDRESSES,
            testChainId,
            uint8(32),
            bytes32(uint256(uint160(PAUSER))),
            EVM_ADDR_LEN,
            FREEZER,
            EVM_ADDR_LEN,
            UNPAUSER
        );
        vm.expectRevert(ITokenBridge.InvalidAddressLength.selector);
        bridge.submitSetPauserAddresses(_signAndEncodeVM(payload, 0));
    }

    function testSubmitSetPauserAddresses_Revert_InvalidFreezerLength() public {
        bytes memory payload = abi.encodePacked(
            tokenBridgeModule,
            ACTION_SET_PAUSER_ADDRESSES,
            testChainId,
            EVM_ADDR_LEN,
            PAUSER,
            uint8(32),
            bytes32(uint256(uint160(FREEZER))),
            EVM_ADDR_LEN,
            UNPAUSER
        );
        vm.expectRevert(ITokenBridge.InvalidAddressLength.selector);
        bridge.submitSetPauserAddresses(_signAndEncodeVM(payload, 0));
    }

    function testSubmitSetPauserAddresses_Revert_InvalidUnpauserLength() public {
        bytes memory payload = abi.encodePacked(
            tokenBridgeModule,
            ACTION_SET_PAUSER_ADDRESSES,
            testChainId,
            EVM_ADDR_LEN,
            PAUSER,
            EVM_ADDR_LEN,
            FREEZER,
            uint8(21),
            UNPAUSER,
            uint8(0xff)
        );
        vm.expectRevert(ITokenBridge.InvalidAddressLength.selector);
        bridge.submitSetPauserAddresses(_signAndEncodeVM(payload, 0));
    }

    function testSubmitSetPauserAddresses_AllUnassigned_ZeroLength() public {
        // Length 0 on all roles is the canonical "unassigned" encoding.
        bytes memory payload = _setPauserAddressesPayloadOptional(
            testChainId, false, address(0), false, address(0), false, address(0)
        );
        vm.expectEmit(true, true, true, true);
        emit PauserAddressesSet(address(0), address(0), address(0));
        bridge.submitSetPauserAddresses(_signAndEncodeVM(payload, 0));
        assertEq(bridge.pauser(), address(0));
        assertEq(bridge.freezer(), address(0));
        assertEq(bridge.unpauser(), address(0));
    }

    function testSubmitSetPauserAddresses_MiddleUnassigned_FreezerOmitted() public {
        // Freezer (middle field) unassigned; pauser and unpauser set.
        bytes memory payload = _setPauserAddressesPayloadOptional(
            testChainId, true, PAUSER, false, address(0), true, UNPAUSER
        );
        bridge.submitSetPauserAddresses(_signAndEncodeVM(payload, 0));
        assertEq(bridge.pauser(), PAUSER);
        assertEq(bridge.freezer(), address(0));
        assertEq(bridge.unpauser(), UNPAUSER);
    }

    function testSubmitSetPauserAddresses_AllZero20ByteAddress_IsUnassigned() public {
        // An all-zero 20-byte address must be treated as equivalent to a zero-length field —
        // i.e., the role ends up unassigned and the entry point reverts before comparing msg.sender.
        bytes memory payload = _setPauserAddressesPayload(testChainId, address(0), address(0), address(0));
        bridge.submitSetPauserAddresses(_signAndEncodeVM(payload, 0));
        assertEq(bridge.pauser(), address(0));
        assertEq(bridge.freezer(), address(0));
        assertEq(bridge.unpauser(), address(0));

        vm.expectRevert(ITokenBridge.NotPauser.selector);
        vm.prank(address(0));
        bridge.pause();

        vm.expectRevert(ITokenBridge.NotFreezer.selector);
        vm.prank(address(0));
        bridge.freeze();
    }

    function testPause_Revert_PauserUnassigned() public {
        // No prior governance message → pauser defaults to address(0) → pause() must revert before
        // comparing msg.sender.
        vm.expectRevert(ITokenBridge.NotPauser.selector);
        vm.prank(PAUSER);
        bridge.pause();
    }

    function testFreeze_Revert_FreezerUnassigned() public {
        vm.expectRevert(ITokenBridge.NotFreezer.selector);
        vm.prank(FREEZER);
        bridge.freeze();
    }

    function testUnpause_Revert_UnpauserUnassigned() public {
        // Configure only pauser, then pause; unpause must remain stuck because unpauser is unassigned.
        bytes memory payload = _setPauserAddressesPayloadOptional(
            testChainId, true, PAUSER, false, address(0), false, address(0)
        );
        bridge.submitSetPauserAddresses(_signAndEncodeVM(payload, 0));

        vm.prank(PAUSER);
        bridge.pause();
        assertTrue(bridge.paused());

        vm.expectRevert(ITokenBridge.NotUnpauser.selector);
        vm.prank(UNPAUSER);
        bridge.unpause();

        // Recovery path: governance assigns an unpauser, then unpause succeeds.
        bytes memory recovery = _setPauserAddressesPayload(testChainId, PAUSER, FREEZER, UNPAUSER);
        bridge.submitSetPauserAddresses(_signAndEncodeVM(recovery, 1));
        vm.prank(UNPAUSER);
        bridge.unpause();
        assertFalse(bridge.paused());
    }

    function testSubmitSetPauserAddresses_Revert_AlreadyConsumed() public {
        bytes memory payload = _setPauserAddressesPayload(testChainId, PAUSER, FREEZER, UNPAUSER);
        bytes memory vaa = _signAndEncodeVM(payload, 0);
        bridge.submitSetPauserAddresses(vaa);
        vm.expectRevert(ITokenBridge.GovernanceActionConsumed.selector);
        bridge.submitSetPauserAddresses(vaa);
    }

    function testSubmitSetPauserAddresses_CanRotate() public {
        bytes memory v1 = _signAndEncodeVM(_setPauserAddressesPayload(testChainId, PAUSER, FREEZER, UNPAUSER), 0);
        bridge.submitSetPauserAddresses(v1);

        address newPauser = address(0xAAAA);
        address newFreezer = address(0xCCCC);
        address newUnpauser = address(0xBBBB);
        bytes memory v2 = _signAndEncodeVM(
            _setPauserAddressesPayload(testChainId, newPauser, newFreezer, newUnpauser),
            1
        );
        bridge.submitSetPauserAddresses(v2);
        assertEq(bridge.pauser(), newPauser);
        assertEq(bridge.freezer(), newFreezer);
        assertEq(bridge.unpauser(), newUnpauser);
    }

    // ============================ pause ============================

    function testPause_Success() public {
        _configureRoles();

        uint64 expectedExpiry = uint64(block.timestamp) + PAUSE_DURATION;
        vm.expectEmit(true, true, true, true);
        emit Paused(PAUSER, expectedExpiry);
        vm.prank(PAUSER);
        bridge.pause();
        assertTrue(bridge.paused());
        assertEq(bridge.pauseExpiry(), expectedExpiry);
    }

    function testPause_Revert_NotPauser() public {
        _configureRoles();
        vm.expectRevert(ITokenBridge.NotPauser.selector);
        bridge.pause();
    }

    function testPause_PushesExpiryForward() public {
        _configureRoles();
        vm.prank(PAUSER);
        bridge.pause();
        uint64 firstExpiry = bridge.pauseExpiry();

        // Advance time and re-pause: expiry moves forward (not idempotent).
        vm.warp(block.timestamp + 1 days);
        vm.prank(PAUSER);
        bridge.pause();
        uint64 secondExpiry = bridge.pauseExpiry();

        assertGt(secondExpiry, firstExpiry);
        assertEq(secondExpiry, uint64(block.timestamp) + PAUSE_DURATION);
    }

    function testPause_DoesNotReduceFreezeExpiry() public {
        _configureRoles();
        // Freeze sets max expiry.
        vm.prank(FREEZER);
        bridge.freeze();
        assertEq(bridge.pauseExpiry(), MAX_TIMESTAMP);

        // A subsequent pause must NOT pull the expiry down to now + 5d.
        vm.prank(PAUSER);
        bridge.pause();
        assertEq(bridge.pauseExpiry(), MAX_TIMESTAMP);
        assertTrue(bridge.paused());
    }

    // ============================ freeze ============================

    function testFreeze_Success() public {
        _configureRoles();

        vm.expectEmit(true, true, true, true);
        emit Frozen(FREEZER, MAX_TIMESTAMP);
        vm.prank(FREEZER);
        bridge.freeze();
        assertTrue(bridge.paused());
        assertEq(bridge.pauseExpiry(), MAX_TIMESTAMP);
    }

    function testFreeze_Revert_NotFreezer() public {
        _configureRoles();
        vm.expectRevert(ITokenBridge.NotFreezer.selector);
        bridge.freeze();
    }

    function testFreeze_Idempotent() public {
        _configureRoles();
        vm.prank(FREEZER);
        bridge.freeze();
        assertEq(bridge.pauseExpiry(), MAX_TIMESTAMP);
        // Freezing again is a no-op effect.
        vm.prank(FREEZER);
        bridge.freeze();
        assertEq(bridge.pauseExpiry(), MAX_TIMESTAMP);
        assertTrue(bridge.paused());
    }

    // ============================ unpause ============================

    function testUnpause_Success() public {
        _configureRoles();
        vm.prank(PAUSER);
        bridge.pause();

        vm.expectEmit(true, true, true, true);
        emit Unpaused(UNPAUSER);
        vm.prank(UNPAUSER);
        bridge.unpause();
        assertFalse(bridge.paused());
        // Expiry brought down to now.
        assertEq(bridge.pauseExpiry(), uint64(block.timestamp));
    }

    function testUnpause_Revert_NotUnpauser() public {
        _configureRoles();
        vm.prank(PAUSER);
        bridge.pause();
        vm.expectRevert(ITokenBridge.NotUnpauser.selector);
        bridge.unpause();
    }

    function testUnpause_Revert_WhenNotPaused() public {
        _configureRoles();
        // Not paused — unpause must revert with NotPaused.
        vm.expectRevert(ITokenBridge.NotPaused.selector);
        vm.prank(UNPAUSER);
        bridge.unpause();
    }

    function testUnpause_LiftsFreeze() public {
        _configureRoles();
        vm.prank(FREEZER);
        bridge.freeze();
        assertTrue(bridge.paused());

        // The unpauser can lift a freeze early.
        vm.prank(UNPAUSER);
        bridge.unpause();
        assertFalse(bridge.paused());
        assertEq(bridge.pauseExpiry(), uint64(block.timestamp));
    }

    function testUnpause_AfterFreezeThenPauseWorks() public {
        // freeze -> unpause (expiry=now) -> pause must set a normal 5-day expiry (the stale max
        // expiry must not linger and block the pauser).
        _configureRoles();
        vm.prank(FREEZER);
        bridge.freeze();
        vm.prank(UNPAUSER);
        bridge.unpause();
        assertEq(bridge.pauseExpiry(), uint64(block.timestamp));

        vm.prank(PAUSER);
        bridge.pause();
        assertTrue(bridge.paused());
        assertEq(bridge.pauseExpiry(), uint64(block.timestamp) + PAUSE_DURATION);
    }

    // ============================ unpauseExpired (permissionless) ============================

    function testUnpauseExpired_AfterExpiry() public {
        _configureRoles();
        vm.prank(PAUSER);
        bridge.pause();
        uint64 expiry = bridge.pauseExpiry();

        // Advance past expiry; anyone (here: an arbitrary address) can unpause.
        vm.warp(uint256(expiry) + 1);
        vm.prank(address(0xD00D));
        bridge.unpauseExpired();
        assertFalse(bridge.paused());
        assertEq(bridge.pauseExpiry(), uint64(block.timestamp));
    }

    function testUnpauseExpired_AtExactExpiry() public {
        // Boundary: block.timestamp == pauseExpiry must succeed (guard is `<`).
        _configureRoles();
        vm.prank(PAUSER);
        bridge.pause();
        uint64 expiry = bridge.pauseExpiry();

        vm.warp(uint256(expiry));
        vm.prank(address(0xD00D));
        bridge.unpauseExpired();
        assertFalse(bridge.paused());
    }

    function testUnpauseExpired_Revert_BeforeExpiry() public {
        _configureRoles();
        vm.prank(PAUSER);
        bridge.pause();

        // Still within the window.
        vm.warp(block.timestamp + 1 days);
        vm.expectRevert(ITokenBridge.NotExpired.selector);
        bridge.unpauseExpired();
    }

    function testUnpauseExpired_Revert_WhenNotPaused() public {
        _configureRoles();
        // Not paused — must revert with NotPaused (even though now >= expiry == 0).
        vm.expectRevert(ITokenBridge.NotPaused.selector);
        bridge.unpauseExpired();
    }

    function testUnpauseExpired_CannotLiftFreeze() public {
        _configureRoles();
        vm.prank(FREEZER);
        bridge.freeze();

        // Even far in the future, now < MAX expiry, so a freeze is never permissionlessly liftable.
        vm.warp(uint256(MAX_TIMESTAMP) - 1);
        vm.expectRevert(ITokenBridge.NotExpired.selector);
        bridge.unpauseExpired();
    }

    // ============================ notPaused on entry points ============================

    function testNotPaused_AttestToken_RevertsWhenPaused() public {
        _configureRoles();
        vm.prank(PAUSER);
        bridge.pause();

        vm.expectRevert(ITokenBridge.BridgePaused.selector);
        bridge.attestToken(address(weth), 0);
    }

    function testNotPaused_TransferTokens_RevertsWhenPaused() public {
        _configureRoles();
        vm.prank(PAUSER);
        bridge.pause();

        vm.expectRevert(ITokenBridge.BridgePaused.selector);
        bridge.transferTokens(address(weth), 1, 2, bytes32(0), 0, 0);
    }

    function testNotPaused_TransferTokensWithPayload_RevertsWhenPaused() public {
        _configureRoles();
        vm.prank(PAUSER);
        bridge.pause();
        vm.expectRevert(ITokenBridge.BridgePaused.selector);
        bridge.transferTokensWithPayload(address(weth), 1, 2, bytes32(0), 0, hex"");
    }

    function testNotPaused_WrapAndTransferETH_RevertsWhenPaused() public {
        _configureRoles();
        vm.prank(PAUSER);
        bridge.pause();
        vm.expectRevert(ITokenBridge.BridgePaused.selector);
        bridge.wrapAndTransferETH(2, bytes32(0), 0, 0);
    }

    function testNotPaused_WrapAndTransferETHWithPayload_RevertsWhenPaused() public {
        _configureRoles();
        vm.prank(PAUSER);
        bridge.pause();
        vm.expectRevert(ITokenBridge.BridgePaused.selector);
        bridge.wrapAndTransferETHWithPayload(2, bytes32(0), 0, hex"");
    }

    function testNotPaused_CompleteTransfer_RevertsWhenPaused() public {
        _configureRoles();
        vm.prank(PAUSER);
        bridge.pause();
        vm.expectRevert(ITokenBridge.BridgePaused.selector);
        bridge.completeTransfer(hex"");
    }

    function testNotPaused_CompleteTransferAndUnwrapETH_RevertsWhenPaused() public {
        _configureRoles();
        vm.prank(PAUSER);
        bridge.pause();
        vm.expectRevert(ITokenBridge.BridgePaused.selector);
        bridge.completeTransferAndUnwrapETH(hex"");
    }

    function testNotPaused_CompleteTransferWithPayload_RevertsWhenPaused() public {
        _configureRoles();
        vm.prank(PAUSER);
        bridge.pause();
        vm.expectRevert(ITokenBridge.BridgePaused.selector);
        bridge.completeTransferWithPayload(hex"");
    }

    function testNotPaused_CompleteTransferAndUnwrapETHWithPayload_RevertsWhenPaused() public {
        _configureRoles();
        vm.prank(PAUSER);
        bridge.pause();
        vm.expectRevert(ITokenBridge.BridgePaused.selector);
        bridge.completeTransferAndUnwrapETHWithPayload(hex"");
    }

    function testNotPaused_CreateWrapped_RevertsWhenPaused() public {
        _configureRoles();
        vm.prank(PAUSER);
        bridge.pause();
        vm.expectRevert(ITokenBridge.BridgePaused.selector);
        bridge.createWrapped(hex"");
    }

    function testNotPaused_UpdateWrapped_RevertsWhenPaused() public {
        _configureRoles();
        vm.prank(PAUSER);
        bridge.pause();
        vm.expectRevert(ITokenBridge.BridgePaused.selector);
        bridge.updateWrapped(hex"");
    }

    function testNotPaused_RevertsWhenFrozen() public {
        // A freeze paused the bridge; user entry points must revert too.
        _configureRoles();
        vm.prank(FREEZER);
        bridge.freeze();
        vm.expectRevert(ITokenBridge.BridgePaused.selector);
        bridge.transferTokens(address(weth), 1, 2, bytes32(0), 0, 0);
    }

    // ============================ Governance handlers while paused ============================
    //
    // The whitepaper specifies that every governance handler must remain callable while the bridge
    // is paused (only user-facing entry points are gated by `notPaused`).

    function testNotPaused_SetPauserAddresses_WorksWhenPaused() public {
        _configureRoles();
        vm.prank(PAUSER);
        bridge.pause();

        address newPauser = address(0xFEED);
        address newFreezer = address(0xFACE);
        address newUnpauser = address(0xCEED);
        bytes memory payload = _setPauserAddressesPayload(testChainId, newPauser, newFreezer, newUnpauser);
        bridge.submitSetPauserAddresses(_signAndEncodeVM(payload, 7));
        assertEq(bridge.pauser(), newPauser);
        assertEq(bridge.freezer(), newFreezer);
        assertEq(bridge.unpauser(), newUnpauser);
        assertTrue(bridge.paused());
    }

    function testSubmitSetPauserAddresses_Revert_WrongLength() public {
        bytes memory payload = abi.encodePacked(_setPauserAddressesPayload(testChainId, PAUSER, FREEZER, UNPAUSER), hex"ff");
        vm.expectRevert(ITokenBridge.WrongLength.selector);
        bridge.submitSetPauserAddresses(_signAndEncodeVM(payload, 0));
    }

    function testSubmitSetPauserAddresses_Revert_WrongGovernanceChain() public {
        bytes memory payload = _setPauserAddressesPayload(testChainId, PAUSER, FREEZER, UNPAUSER);
        bytes memory vaa = _signAndEncodeVMFrom(payload, 0, uint16(99), bytes32(uint256(0x4)));
        vm.expectRevert(ITokenBridge.WrongGovernanceChain.selector);
        bridge.submitSetPauserAddresses(vaa);
    }

    function testSubmitSetPauserAddresses_Revert_WrongGovernanceContract() public {
        bytes memory payload = _setPauserAddressesPayload(testChainId, PAUSER, FREEZER, UNPAUSER);
        bytes memory vaa = _signAndEncodeVMFrom(payload, 0, uint16(1), bytes32(uint256(0xDEADBEEF)));
        vm.expectRevert(ITokenBridge.WrongGovernanceContract.selector);
        bridge.submitSetPauserAddresses(vaa);
    }

    function testNotPaused_RegisterChain_WorksWhenPaused() public {
        _configureRoles();
        vm.prank(PAUSER);
        bridge.pause();

        uint16 foreignChain = 42;
        bytes32 foreignEmitter = bytes32(uint256(0xDEAD));
        bytes memory payload = abi.encodePacked(
            tokenBridgeModule,
            uint8(1),         // action: RegisterChain
            uint16(0),        // chainId 0 = any
            foreignChain,
            foreignEmitter
        );
        bridge.registerChain(_signAndEncodeVM(payload, 100));
        assertEq(bridge.bridgeContracts(foreignChain), foreignEmitter);
    }

    function testNotPaused_Upgrade_NotBlockedByPause() public {
        _configureRoles();
        vm.prank(PAUSER);
        bridge.pause();

        // Wrong-chain UpgradeContract VAA: the handler must revert at the chainId check (proving it
        // runs past the absent notPaused gate), NOT at `BridgePaused`.
        bytes memory payload = abi.encodePacked(
            tokenBridgeModule,
            uint8(2),                                // action: Upgrade
            uint16(testChainId + 1),                 // wrong chain
            bytes32(uint256(uint160(address(0xBEEF))))
        );
        vm.expectRevert(ITokenBridge.WrongChainId.selector);
        bridge.upgrade(_signAndEncodeVM(payload, 101));
    }

    function testNotPaused_SubmitRecoverChainId_WorksWhenPaused() public {
        _configureRoles();
        vm.prank(PAUSER);
        bridge.pause();

        uint256 forkChainId = testEvmChainId + 1;
        vm.chainId(forkChainId);

        uint16 newWormholeChainId = 999;
        bytes memory payload = abi.encodePacked(
            tokenBridgeModule,
            uint8(3),                  // action: RecoverChainId
            uint256(forkChainId),      // evmChainId — must match block.chainid
            newWormholeChainId
        );
        bridge.submitRecoverChainId(_signAndEncodeVM(payload, 102));
        assertEq(bridge.chainId(), newWormholeChainId);
        assertEq(bridge.evmChainId(), forkChainId);
    }

    // ============================ Idempotency ============================

    function testPause_Idempotent_State() public {
        _configureRoles();
        vm.prank(PAUSER);
        bridge.pause();
        assertTrue(bridge.paused());

        // A second pause() keeps `paused` true (and pushes expiry — see testPause_PushesExpiryForward).
        vm.prank(PAUSER);
        bridge.pause();
        assertTrue(bridge.paused());
    }

    // ============================ Rotation while paused ============================

    function testSubmitSetPauserAddresses_CanRotateWhilePaused() public {
        _configureRoles();
        vm.prank(PAUSER);
        bridge.pause();
        assertTrue(bridge.paused());

        // Rotate to a fresh role set while paused; the new addresses take effect immediately and
        // the bridge stays paused (rotation does not unpause).
        address newPauser = address(0xAAAA);
        address newFreezer = address(0xCCCC);
        address newUnpauser = address(0xBBBB);
        bytes memory payload = _setPauserAddressesPayload(testChainId, newPauser, newFreezer, newUnpauser);
        bridge.submitSetPauserAddresses(_signAndEncodeVM(payload, 200));

        assertEq(bridge.pauser(), newPauser);
        assertEq(bridge.freezer(), newFreezer);
        assertEq(bridge.unpauser(), newUnpauser);
        assertTrue(bridge.paused());

        // Old unpauser can no longer act.
        vm.expectRevert(ITokenBridge.NotUnpauser.selector);
        vm.prank(UNPAUSER);
        bridge.unpause();

        // New unpauser unpauses successfully.
        vm.prank(newUnpauser);
        bridge.unpause();
        assertFalse(bridge.paused());

        // And the new pauser can re-pause.
        vm.prank(newPauser);
        bridge.pause();
        assertTrue(bridge.paused());
    }

    // ============================ Storage layout ============================
    //
    // These tests pin the ERC-7201 namespaced storage decision: pause role addresses + pauseExpiry
    // live in a namespaced slot disjoint from `BridgeStorage.State`, so adding pauser support never
    // shifts slot 13 (the `_status` slot inherited from OpenZeppelin's `ReentrancyGuard`).

    /// @dev Matches `BridgePauserStorage.LAYOUT_SLOT`.
    bytes32 constant PAUSER_NAMESPACE_SLOT =
        0x685f7dd8ace9c4fb94a4997fcd733e0d769273ee87b95731641e14d0cc4a6700;

    function testStorageLayout_NamespacedSlotMatchesERC7201() public {
        bytes32 expected =
            keccak256(abi.encode(uint256(keccak256("wormhole.tokenbridge.pauser.storage")) - 1))
            & ~bytes32(uint256(0xff));
        assertEq(BridgePauserStorage.LAYOUT_SLOT, expected);
        assertEq(BridgePauserStorage.LAYOUT_SLOT, PAUSER_NAMESPACE_SLOT);
    }

    function testStorageLayout_FreshDeploy_RolesAndExpiryAreZero() public {
        assertEq(bridge.pauser(), address(0));
        assertEq(bridge.freezer(), address(0));
        assertEq(bridge.unpauser(), address(0));
        assertEq(bridge.pauseExpiry(), 0);
        assertFalse(bridge.paused());
    }

    function testStorageLayout_RolesLiveAtNamespacedSlots() public {
        bytes memory payload = _setPauserAddressesPayload(testChainId, PAUSER, FREEZER, UNPAUSER);
        bridge.submitSetPauserAddresses(_signAndEncodeVM(payload, 0));

        // Layout (append-only, preserving the deployed pauser/unpauser slots):
        //   slot+0: pauser, slot+1: unpauser, slot+2: freezer (low 20 bytes) | pauseExpiry (next 8).
        bytes32 pauserSlot = vm.load(address(bridge), PAUSER_NAMESPACE_SLOT);
        bytes32 unpauserSlot = vm.load(address(bridge), bytes32(uint256(PAUSER_NAMESPACE_SLOT) + 1));
        bytes32 freezerSlot = vm.load(address(bridge), bytes32(uint256(PAUSER_NAMESPACE_SLOT) + 2));
        assertEq(address(uint160(uint256(pauserSlot))), PAUSER);
        assertEq(address(uint160(uint256(unpauserSlot))), UNPAUSER);
        assertEq(address(uint160(uint256(freezerSlot))), FREEZER);
    }

    function testStorageLayout_PauseExpiryPacksWithFreezer() public {
        _configureRoles();
        vm.prank(PAUSER);
        bridge.pause();
        uint64 expiry = bridge.pauseExpiry();
        assertGt(expiry, 0);

        // pauseExpiry packs into the freezer slot (slot+2) above the 20-byte freezer address: it
        // occupies bytes 20..27 of that slot.
        bytes32 freezerSlot = vm.load(address(bridge), bytes32(uint256(PAUSER_NAMESPACE_SLOT) + 2));
        uint64 packedExpiry = uint64(uint256(freezerSlot) >> 160);
        assertEq(packedExpiry, expiry);
        // Freezer address still intact in the low 20 bytes.
        assertEq(address(uint160(uint256(freezerSlot))), FREEZER);
    }

    function testStorageLayout_StateSlotsUnchangedByPauser() public {
        // Slot 13 is the ReentrancyGuard `_status` slot. The ERC-7201 split keeps the pauser address
        // out of `BridgeStorage.State`, so slot 13 must NOT hold the pauser address.
        bytes memory payload = _setPauserAddressesPayload(testChainId, PAUSER, FREEZER, UNPAUSER);
        bridge.submitSetPauserAddresses(_signAndEncodeVM(payload, 0));

        bytes32 slot13 = vm.load(address(bridge), bytes32(uint256(13)));
        assertTrue(slot13 != bytes32(uint256(uint160(PAUSER))));
        assertLt(uint256(slot13), uint256(3));
    }

    function testStorageLayout_PausedPacksIntoProviderSlot() public {
        // `paused` is packed into the first slot of `BridgeStorage.Provider` after
        // `chainId(2) | governanceChainId(2) | finality(1)` — slot 2 of the contract.
        bytes32 providerSlotBefore = vm.load(address(bridge), bytes32(uint256(2)));
        uint256 pausedByteIndex = 31 - 5;
        assertEq(uint8(providerSlotBefore[pausedByteIndex]), 0);

        _configureRoles();
        vm.prank(PAUSER);
        bridge.pause();

        bytes32 providerSlotAfter = vm.load(address(bridge), bytes32(uint256(2)));
        assertEq(uint8(providerSlotAfter[pausedByteIndex]), 1);
        // chainId, governanceChainId, finality must be untouched by the pause.
        assertEq(
            providerSlotBefore & bytes32(uint256(0x000000000000000000000000000000000000000000000000000000ffffffffff)),
            providerSlotAfter & bytes32(uint256(0x000000000000000000000000000000000000000000000000000000ffffffffff))
        );
    }

    // ============================ Real wire vector ============================
    //
    // Pin compatibility with the length-prefixed action-4 wire format (whitepaper 0003): three
    // length-prefixed addresses in order pauser, freezer, unpauser.

    function testSubmitSetPauserAddresses_RealVector_EvmAllSet() public {
        bytes memory payload =
            hex"000000000000000000000000000000000000000000546f6b656e427269646765"
            hex"04"
            hex"0002"
            hex"14" hex"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
            hex"14" hex"cccccccccccccccccccccccccccccccccccccccc"
            hex"14" hex"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb";

        bridge.submitSetPauserAddresses(_signAndEncodeVM(payload, 0));
        assertEq(bridge.pauser(), address(0xaAaAaAaaAaAaAaaAaAAAAAAAAaaaAaAaAaaAaaAa));
        assertEq(bridge.freezer(), address(0xCcCCccccCCCCcCCCCCCcCcCccCcCCCcCcccccccC));
        assertEq(bridge.unpauser(), address(0xbBbBBBBbbBBBbbbBbbBbbbbBBbBbbbbBbBbbBBbB));
    }

    function testSubmitSetPauserAddresses_RealVector_FreezerUnassigned() public {
        bytes memory payload =
            hex"000000000000000000000000000000000000000000546f6b656e427269646765"
            hex"04"
            hex"0002"
            hex"14" hex"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
            hex"00"
            hex"14" hex"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb";

        bridge.submitSetPauserAddresses(_signAndEncodeVM(payload, 0));
        assertEq(bridge.pauser(), address(0xaAaAaAaaAaAaAaaAaAAAAAAAAaaaAaAaAaaAaaAa));
        assertEq(bridge.freezer(), address(0));
        assertEq(bridge.unpauser(), address(0xbBbBBBBbbBBBbbbBbbBbbbbBBbBbbbbBbBbbBBbB));
    }

    function testSubmitSetPauserAddresses_RealVector_AllUnassigned() public {
        bytes memory payload =
            hex"000000000000000000000000000000000000000000546f6b656e427269646765"
            hex"04"
            hex"0002"
            hex"00"
            hex"00"
            hex"00";

        bridge.submitSetPauserAddresses(_signAndEncodeVM(payload, 0));
        assertEq(bridge.pauser(), address(0));
        assertEq(bridge.freezer(), address(0));
        assertEq(bridge.unpauser(), address(0));
    }

    // ============================ Internal ============================

    function _configureRoles() internal {
        bytes memory payload = _setPauserAddressesPayload(testChainId, PAUSER, FREEZER, UNPAUSER);
        bridge.submitSetPauserAddresses(_signAndEncodeVM(payload, 0));
    }
}
