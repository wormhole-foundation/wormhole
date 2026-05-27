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
    address constant UNPAUSER = address(0xBEEF);

    // Re-declared so vm.expectEmit can match (Solidity 0.8.4 doesn't allow IName.Event references).
    event PauserAddressesSet(address indexed pauser, address indexed unpauser);
    event Paused(address indexed by);
    event Unpaused(address indexed by);

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
    ///      whitepapers/0003_token_bridge.md. Each address is preceded by its length (20 on EVM, or
    ///      0 to leave the role unassigned).
    function _setPauserAddressesPayload(uint16 chain_, address pauser_, address unpauser_)
        internal
        pure
        returns (bytes memory)
    {
        return abi.encodePacked(
            tokenBridgeModule,
            ACTION_SET_PAUSER_ADDRESSES,
            chain_,
            EVM_ADDR_LEN,
            pauser_,
            EVM_ADDR_LEN,
            unpauser_
        );
    }

    /// @dev Encode a SetPauserAddresses payload where each role may independently be marked as
    ///      unassigned (length 0, zero-byte body).
    function _setPauserAddressesPayloadOptional(
        uint16 chain_,
        bool hasPauser,
        address pauser_,
        bool hasUnpauser,
        address unpauser_
    ) internal pure returns (bytes memory) {
        bytes memory pauserField = hasPauser
            ? abi.encodePacked(EVM_ADDR_LEN, pauser_)
            : abi.encodePacked(uint8(0));
        bytes memory unpauserField = hasUnpauser
            ? abi.encodePacked(EVM_ADDR_LEN, unpauser_)
            : abi.encodePacked(uint8(0));
        return abi.encodePacked(
            tokenBridgeModule,
            ACTION_SET_PAUSER_ADDRESSES,
            chain_,
            pauserField,
            unpauserField
        );
    }

    // ============================ submitSetPauserAddresses ============================

    function testSubmitSetPauserAddresses_Success() public {
        bytes memory payload = _setPauserAddressesPayload(testChainId, PAUSER, UNPAUSER);
        bytes memory vaa = _signAndEncodeVM(payload, 0);

        vm.expectEmit(true, true, true, true);
        emit PauserAddressesSet(PAUSER, UNPAUSER);
        bridge.submitSetPauserAddresses(vaa);

        assertEq(bridge.pauser(), PAUSER);
        assertEq(bridge.unpauser(), UNPAUSER);
        assertFalse(bridge.paused());
    }

    function testSubmitSetPauserAddresses_Revert_WrongChain() public {
        bytes memory payload = _setPauserAddressesPayload(uint16(99), PAUSER, UNPAUSER);
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
            uint8(21),
            UNPAUSER,
            uint8(0xff)
        );
        vm.expectRevert(ITokenBridge.InvalidAddressLength.selector);
        bridge.submitSetPauserAddresses(_signAndEncodeVM(payload, 0));
    }

    function testSubmitSetPauserAddresses_BothUnassigned_ZeroLength() public {
        // Length 0 on both roles is the canonical "unassigned" encoding.
        bytes memory payload = _setPauserAddressesPayloadOptional(
            testChainId, false, address(0), false, address(0)
        );
        vm.expectEmit(true, true, true, true);
        emit PauserAddressesSet(address(0), address(0));
        bridge.submitSetPauserAddresses(_signAndEncodeVM(payload, 0));
        assertEq(bridge.pauser(), address(0));
        assertEq(bridge.unpauser(), address(0));
    }

    function testSubmitSetPauserAddresses_PauserUnassigned_OnlyUnpauserSet() public {
        bytes memory payload = _setPauserAddressesPayloadOptional(
            testChainId, false, address(0), true, UNPAUSER
        );
        bridge.submitSetPauserAddresses(_signAndEncodeVM(payload, 0));
        assertEq(bridge.pauser(), address(0));
        assertEq(bridge.unpauser(), UNPAUSER);
    }

    function testSubmitSetPauserAddresses_AllZero20ByteAddress_IsUnassigned() public {
        // An all-zero 20-byte address must be treated as equivalent to a zero-length field —
        // i.e., the role ends up unassigned and the entry point reverts before comparing msg.sender.
        bytes memory payload = _setPauserAddressesPayload(testChainId, address(0), address(0));
        bridge.submitSetPauserAddresses(_signAndEncodeVM(payload, 0));
        assertEq(bridge.pauser(), address(0));
        assertEq(bridge.unpauser(), address(0));

        vm.expectRevert(ITokenBridge.NotPauser.selector);
        vm.prank(address(0));
        bridge.pause();

        vm.expectRevert(ITokenBridge.NotUnpauser.selector);
        vm.prank(address(0));
        bridge.unpause();
    }

    function testPause_Revert_PauserUnassigned() public {
        // No prior governance message → pauser defaults to address(0) → pause() must revert before
        // comparing msg.sender.
        vm.expectRevert(ITokenBridge.NotPauser.selector);
        vm.prank(PAUSER);
        bridge.pause();
    }

    function testUnpause_Revert_UnpauserUnassigned() public {
        // Configure only pauser, then pause; unpause must remain stuck because unpauser is unassigned.
        bytes memory payload = _setPauserAddressesPayloadOptional(
            testChainId, true, PAUSER, false, address(0)
        );
        bridge.submitSetPauserAddresses(_signAndEncodeVM(payload, 0));

        vm.prank(PAUSER);
        bridge.pause();
        assertTrue(bridge.paused());

        vm.expectRevert(ITokenBridge.NotUnpauser.selector);
        vm.prank(UNPAUSER);
        bridge.unpause();

        // Recovery path: governance assigns an unpauser, then unpause succeeds.
        bytes memory recovery = _setPauserAddressesPayload(testChainId, PAUSER, UNPAUSER);
        bridge.submitSetPauserAddresses(_signAndEncodeVM(recovery, 1));
        vm.prank(UNPAUSER);
        bridge.unpause();
        assertFalse(bridge.paused());
    }

    function testSubmitSetPauserAddresses_Revert_AlreadyConsumed() public {
        bytes memory payload = _setPauserAddressesPayload(testChainId, PAUSER, UNPAUSER);
        bytes memory vaa = _signAndEncodeVM(payload, 0);
        bridge.submitSetPauserAddresses(vaa);
        vm.expectRevert(ITokenBridge.GovernanceActionConsumed.selector);
        bridge.submitSetPauserAddresses(vaa);
    }

    function testSubmitSetPauserAddresses_CanRotate() public {
        bytes memory v1 = _signAndEncodeVM(_setPauserAddressesPayload(testChainId, PAUSER, UNPAUSER), 0);
        bridge.submitSetPauserAddresses(v1);

        address newPauser = address(0xAAAA);
        address newUnpauser = address(0xBBBB);
        bytes memory v2 = _signAndEncodeVM(
            _setPauserAddressesPayload(testChainId, newPauser, newUnpauser),
            1
        );
        bridge.submitSetPauserAddresses(v2);
        assertEq(bridge.pauser(), newPauser);
        assertEq(bridge.unpauser(), newUnpauser);
    }

    // ============================ pause / unpause ============================

    function testPause_Success() public {
        _configurePauser();

        vm.expectEmit(true, true, true, true);
        emit Paused(PAUSER);
        vm.prank(PAUSER);
        bridge.pause();
        assertTrue(bridge.paused());
    }

    function testPause_Revert_NotPauser() public {
        _configurePauser();
        vm.expectRevert(ITokenBridge.NotPauser.selector);
        bridge.pause();
    }

    function testUnpause_Success() public {
        _configurePauser();
        vm.prank(PAUSER);
        bridge.pause();

        vm.expectEmit(true, true, true, true);
        emit Unpaused(UNPAUSER);
        vm.prank(UNPAUSER);
        bridge.unpause();
        assertFalse(bridge.paused());
    }

    function testUnpause_Revert_NotUnpauser() public {
        _configurePauser();
        vm.prank(PAUSER);
        bridge.pause();
        vm.expectRevert(ITokenBridge.NotUnpauser.selector);
        bridge.unpause();
    }

    function testUnpause_NotGuardedByNotPaused() public {
        _configurePauser();
        // unpause is callable even when not paused; idempotent.
        vm.prank(UNPAUSER);
        bridge.unpause();
        assertFalse(bridge.paused());
    }

    // ============================ notPaused on entry points ============================

    function testNotPaused_AttestToken_RevertsWhenPaused() public {
        _configurePauser();
        vm.prank(PAUSER);
        bridge.pause();

        vm.expectRevert(ITokenBridge.BridgePaused.selector);
        bridge.attestToken(address(weth), 0);
    }

    function testNotPaused_TransferTokens_RevertsWhenPaused() public {
        _configurePauser();
        vm.prank(PAUSER);
        bridge.pause();

        vm.expectRevert(ITokenBridge.BridgePaused.selector);
        bridge.transferTokens(address(weth), 1, 2, bytes32(0), 0, 0);
    }

    function testNotPaused_TransferTokensWithPayload_RevertsWhenPaused() public {
        _configurePauser();
        vm.prank(PAUSER);
        bridge.pause();
        vm.expectRevert(ITokenBridge.BridgePaused.selector);
        bridge.transferTokensWithPayload(address(weth), 1, 2, bytes32(0), 0, hex"");
    }

    function testNotPaused_WrapAndTransferETH_RevertsWhenPaused() public {
        _configurePauser();
        vm.prank(PAUSER);
        bridge.pause();
        vm.expectRevert(ITokenBridge.BridgePaused.selector);
        bridge.wrapAndTransferETH(2, bytes32(0), 0, 0);
    }

    function testNotPaused_WrapAndTransferETHWithPayload_RevertsWhenPaused() public {
        _configurePauser();
        vm.prank(PAUSER);
        bridge.pause();
        vm.expectRevert(ITokenBridge.BridgePaused.selector);
        bridge.wrapAndTransferETHWithPayload(2, bytes32(0), 0, hex"");
    }

    function testNotPaused_CompleteTransfer_RevertsWhenPaused() public {
        _configurePauser();
        vm.prank(PAUSER);
        bridge.pause();
        vm.expectRevert(ITokenBridge.BridgePaused.selector);
        bridge.completeTransfer(hex"");
    }

    function testNotPaused_CompleteTransferAndUnwrapETH_RevertsWhenPaused() public {
        _configurePauser();
        vm.prank(PAUSER);
        bridge.pause();
        vm.expectRevert(ITokenBridge.BridgePaused.selector);
        bridge.completeTransferAndUnwrapETH(hex"");
    }

    function testNotPaused_CompleteTransferWithPayload_RevertsWhenPaused() public {
        _configurePauser();
        vm.prank(PAUSER);
        bridge.pause();
        vm.expectRevert(ITokenBridge.BridgePaused.selector);
        bridge.completeTransferWithPayload(hex"");
    }

    function testNotPaused_CompleteTransferAndUnwrapETHWithPayload_RevertsWhenPaused() public {
        _configurePauser();
        vm.prank(PAUSER);
        bridge.pause();
        vm.expectRevert(ITokenBridge.BridgePaused.selector);
        bridge.completeTransferAndUnwrapETHWithPayload(hex"");
    }

    function testNotPaused_CreateWrapped_RevertsWhenPaused() public {
        _configurePauser();
        vm.prank(PAUSER);
        bridge.pause();
        vm.expectRevert(ITokenBridge.BridgePaused.selector);
        bridge.createWrapped(hex"");
    }

    function testNotPaused_UpdateWrapped_RevertsWhenPaused() public {
        _configurePauser();
        vm.prank(PAUSER);
        bridge.pause();
        vm.expectRevert(ITokenBridge.BridgePaused.selector);
        bridge.updateWrapped(hex"");
    }

    function testNotPaused_GovernanceStillWorksWhenPaused() public {
        _configurePauser();
        vm.prank(PAUSER);
        bridge.pause();

        // submitSetPauserAddresses is governance and must remain callable when paused. Both
        // fields update atomically — assert each independently.
        address newPauser = address(0xFEED);
        address newUnpauser = address(0xCEED);
        bytes memory payload = _setPauserAddressesPayload(testChainId, newPauser, newUnpauser);
        bridge.submitSetPauserAddresses(_signAndEncodeVM(payload, 7));
        assertEq(bridge.pauser(), newPauser);
        assertEq(bridge.unpauser(), newUnpauser);
        assertTrue(bridge.paused());
    }

    function testSubmitSetPauserAddresses_Revert_WrongLength() public {
        bytes memory payload = abi.encodePacked(_setPauserAddressesPayload(testChainId, PAUSER, UNPAUSER), hex"ff");
        vm.expectRevert(ITokenBridge.WrongLength.selector);
        bridge.submitSetPauserAddresses(_signAndEncodeVM(payload, 0));
    }

    function testSubmitSetPauserAddresses_Revert_WrongGovernanceChain() public {
        bytes memory payload = _setPauserAddressesPayload(testChainId, PAUSER, UNPAUSER);
        bytes memory vaa = _signAndEncodeVMFrom(payload, 0, uint16(99), bytes32(uint256(0x4)));
        vm.expectRevert(ITokenBridge.WrongGovernanceChain.selector);
        bridge.submitSetPauserAddresses(vaa);
    }

    function testSubmitSetPauserAddresses_Revert_WrongGovernanceContract() public {
        bytes memory payload = _setPauserAddressesPayload(testChainId, PAUSER, UNPAUSER);
        bytes memory vaa = _signAndEncodeVMFrom(payload, 0, uint16(1), bytes32(uint256(0xDEADBEEF)));
        vm.expectRevert(ITokenBridge.WrongGovernanceContract.selector);
        bridge.submitSetPauserAddresses(vaa);
    }

    // ============================ Other governance handlers while paused ============================
    //
    // The whitepaper specifies that every governance handler must remain callable while the bridge
    // is paused (only user-facing entry points are gated by `notPaused`). For each handler we
    // either drive a successful execution or assert it reverts on a handler-specific check rather
    // than `BridgePaused` — proving the `notPaused` modifier is not applied.

    function testNotPaused_RegisterChain_WorksWhenPaused() public {
        _configurePauser();
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
        _configurePauser();
        vm.prank(PAUSER);
        bridge.pause();

        // Send an UpgradeContract VAA targeted at the wrong chain so the handler reverts at the
        // chainId check rather than executing an upgrade. The point is to demonstrate the revert
        // is `WrongChainId`, NOT `BridgePaused` — i.e. the handler runs past the (absent)
        // notPaused gate.
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
        _configurePauser();
        vm.prank(PAUSER);
        bridge.pause();

        // Simulate a fork: bump block.chainid so `isFork()` returns true, then send a matching
        // RecoverChainId VAA. The handler must run to completion despite the bridge being paused.
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

    function testPause_Idempotent() public {
        _configurePauser();
        vm.prank(PAUSER);
        bridge.pause();
        assertTrue(bridge.paused());

        // A second pause() must not toggle the state and must not revert. State stays `true`.
        vm.prank(PAUSER);
        bridge.pause();
        assertTrue(bridge.paused());
    }

    function testUnpause_Idempotent() public {
        _configurePauser();
        // Bridge starts unpaused. Calling unpause() should keep it unpaused, not toggle to paused.
        vm.prank(UNPAUSER);
        bridge.unpause();
        assertFalse(bridge.paused());

        // Pause, unpause, then unpause again — the second unpause must not re-pause the bridge.
        vm.prank(PAUSER);
        bridge.pause();
        vm.prank(UNPAUSER);
        bridge.unpause();
        assertFalse(bridge.paused());

        vm.prank(UNPAUSER);
        bridge.unpause();
        assertFalse(bridge.paused());
    }

    // ============================ Rotation while paused ============================

    function testSubmitSetPauserAddresses_CanRotateWhilePaused() public {
        _configurePauser();
        vm.prank(PAUSER);
        bridge.pause();
        assertTrue(bridge.paused());

        // Rotate to a fresh pauser / unpauser pair while paused; the new addresses must take
        // effect immediately and the bridge must stay paused (rotation does not unpause).
        address newPauser = address(0xAAAA);
        address newUnpauser = address(0xBBBB);
        bytes memory payload = _setPauserAddressesPayload(testChainId, newPauser, newUnpauser);
        bridge.submitSetPauserAddresses(_signAndEncodeVM(payload, 200));

        assertEq(bridge.pauser(), newPauser);
        assertEq(bridge.unpauser(), newUnpauser);
        assertTrue(bridge.paused());

        // Old pauser/unpauser can no longer act on the bridge.
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
    // These tests pin the ERC-7201 namespaced storage decision: `pauser` and `unpauser` live in a
    // namespaced slot disjoint from `BridgeStorage.State`, so adding/removing pauser support never
    // shifts slot 13 (the `_status` slot inherited from OpenZeppelin's `ReentrancyGuard`).

    /// @dev Matches `BridgePauserStorage.LAYOUT_SLOT`.
    bytes32 constant PAUSER_NAMESPACE_SLOT =
        0x685f7dd8ace9c4fb94a4997fcd733e0d769273ee87b95731641e14d0cc4a6700;

    function testStorageLayout_NamespacedSlotMatchesERC7201() public {
        // Recompute the slot per ERC-7201 and assert it matches the constant baked into
        // `BridgePauserStorage`. Guards against silent drift: any rename of the namespace string,
        // or any miscopied constant, fails here rather than corrupting storage on a live deploy.
        bytes32 expected =
            keccak256(abi.encode(uint256(keccak256("wormhole.tokenbridge.pauser.storage")) - 1))
            & ~bytes32(uint256(0xff));
        assertEq(BridgePauserStorage.LAYOUT_SLOT, expected);
        assertEq(BridgePauserStorage.LAYOUT_SLOT, PAUSER_NAMESPACE_SLOT);
    }

    function testStorageLayout_FreshDeploy_RolesAreZero() public {
        assertEq(bridge.pauser(), address(0));
        assertEq(bridge.unpauser(), address(0));
        assertFalse(bridge.paused());
    }

    function testStorageLayout_PauserLivesAtNamespacedSlot() public {
        bytes memory payload = _setPauserAddressesPayload(testChainId, PAUSER, UNPAUSER);
        bridge.submitSetPauserAddresses(_signAndEncodeVM(payload, 0));

        // `pauser` is at the namespaced slot, `unpauser` packs into the next slot.
        bytes32 pauserSlotValue = vm.load(address(bridge), PAUSER_NAMESPACE_SLOT);
        bytes32 unpauserSlotValue =
            vm.load(address(bridge), bytes32(uint256(PAUSER_NAMESPACE_SLOT) + 1));
        assertEq(address(uint160(uint256(pauserSlotValue))), PAUSER);
        assertEq(address(uint160(uint256(unpauserSlotValue))), UNPAUSER);
    }

    function testStorageLayout_StateSlotsUnchangedByPauser() public {
        // Slot 13 is the ReentrancyGuard `_status` slot. A previous version of this PR placed
        // `pauser` directly in `BridgeStorage.State`, which would have pushed `_status` to slot 15
        // and made the freshly-upgraded proxy read `pauser` from the old `_status` slot. The
        // ERC-7201 split prevents that: slot 13 must NOT hold the pauser address.
        bytes memory payload = _setPauserAddressesPayload(testChainId, PAUSER, UNPAUSER);
        bridge.submitSetPauserAddresses(_signAndEncodeVM(payload, 0));

        bytes32 slot13 = vm.load(address(bridge), bytes32(uint256(13)));
        assertTrue(slot13 != bytes32(uint256(uint160(PAUSER))));
        // `_status` on a fresh proxy is `0` until the first `nonReentrant` call lazily initializes
        // it; either way it stays a small reentrancy sentinel, never an address.
        assertLt(uint256(slot13), uint256(3));
    }

    function testStorageLayout_PausedPacksIntoProviderSlot() public {
        // `paused` is packed into the first slot of `BridgeStorage.Provider` after
        // `chainId(2) | governanceChainId(2) | finality(1)`. That slot starts at offset 2 within
        // `_state` (after `wormhole` and `tokenImplementation`), i.e. slot 2 of the contract.
        bytes32 providerSlotBefore = vm.load(address(bridge), bytes32(uint256(2)));
        // `paused` byte sits at offset 5 within the slot (after chainId + governanceChainId +
        // finality). Note that storage is right-aligned: byte offset N in the slot corresponds to
        // byte index (31 - N) from the left in the bytes32 representation.
        uint256 pausedByteIndex = 31 - 5;
        assertEq(uint8(providerSlotBefore[pausedByteIndex]), 0);

        _configurePauser();
        vm.prank(PAUSER);
        bridge.pause();

        bytes32 providerSlotAfter = vm.load(address(bridge), bytes32(uint256(2)));
        assertEq(uint8(providerSlotAfter[pausedByteIndex]), 1);
        // The lower bytes (chainId, governanceChainId, finality) must be untouched by the pause.
        assertEq(
            providerSlotBefore & bytes32(uint256(0x000000000000000000000000000000000000000000000000000000ffffffffff)),
            providerSlotAfter & bytes32(uint256(0x000000000000000000000000000000000000000000000000000000ffffffffff))
        );
    }

    // ============================ Real wire vector ============================
    //
    // Pin compatibility with the off-chain encoder in `sdk/vaa/payloads.go`. The hex vector below
    // is copied from the `TestBodyTokenBridgeSetPauserAddressesSerialize` "evm both set" case
    // (PR #4810): module(32) || action(04) || chain(0002) || pauserLen(14) || pauser(20 × 0xaa)
    // || unpauserLen(14) || unpauser(20 × 0xbb). If the guardian encoder format ever diverges
    // from what this contract parses, this test fails.

    function testSubmitSetPauserAddresses_RealVector_EvmBothSet() public {
        bytes memory payload =
            hex"000000000000000000000000000000000000000000546f6b656e427269646765"
            hex"04"
            hex"0002"
            hex"14" hex"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
            hex"14" hex"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb";

        bridge.submitSetPauserAddresses(_signAndEncodeVM(payload, 0));
        assertEq(bridge.pauser(), address(0xaAaAaAaaAaAaAaaAaAAAAAAAAaaaAaAaAaaAaaAa));
        assertEq(bridge.unpauser(), address(0xbBbBBBBbbBBBbbbBbbBbbbbBBbBbbbbBbBbbBBbB));
    }

    function testSubmitSetPauserAddresses_RealVector_PauserUnassigned() public {
        // "pauser unassigned, unpauser set" vector from PR #4810.
        bytes memory payload =
            hex"000000000000000000000000000000000000000000546f6b656e427269646765"
            hex"04"
            hex"0002"
            hex"00"
            hex"14" hex"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb";

        bridge.submitSetPauserAddresses(_signAndEncodeVM(payload, 0));
        assertEq(bridge.pauser(), address(0));
        assertEq(bridge.unpauser(), address(0xbBbBBBBbbBBBbbbBbbBbbbbBBbBbbbbBbBbbBBbB));
    }

    function testSubmitSetPauserAddresses_RealVector_BothUnassigned() public {
        // "both unassigned" vector from PR #4810.
        bytes memory payload =
            hex"000000000000000000000000000000000000000000546f6b656e427269646765"
            hex"04"
            hex"0002"
            hex"00"
            hex"00";

        bridge.submitSetPauserAddresses(_signAndEncodeVM(payload, 0));
        assertEq(bridge.pauser(), address(0));
        assertEq(bridge.unpauser(), address(0));
    }

    // ============================ Internal ============================

    function _configurePauser() internal {
        bytes memory payload = _setPauserAddressesPayload(testChainId, PAUSER, UNPAUSER);
        bridge.submitSetPauserAddresses(_signAndEncodeVM(payload, 0));
    }
}
