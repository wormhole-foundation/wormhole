// SPDX-License-Identifier: Apache 2
pragma solidity ^0.8.0;

import "../contracts/bridge/BridgeSetup.sol";
import "../contracts/bridge/BridgeImplementation.sol";
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
        bytes memory body = abi.encodePacked(
            uint32(0), uint32(0), uint16(1), bytes32(uint256(0x4)), sequence, uint8(0), data
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

        // submitSetPauserAddresses is governance and must remain callable when paused.
        bytes memory payload = _setPauserAddressesPayload(testChainId, address(0xFEED), UNPAUSER);
        bridge.submitSetPauserAddresses(_signAndEncodeVM(payload, 7));
        assertEq(bridge.pauser(), address(0xFEED));
    }

    function testSubmitSetPauserAddresses_Revert_WrongLength() public {
        bytes memory payload = abi.encodePacked(_setPauserAddressesPayload(testChainId, PAUSER, UNPAUSER), hex"ff");
        vm.expectRevert(ITokenBridge.WrongLength.selector);
        bridge.submitSetPauserAddresses(_signAndEncodeVM(payload, 0));
    }

    // ============================ Internal ============================

    function _configurePauser() internal {
        bytes memory payload = _setPauserAddressesPayload(testChainId, PAUSER, UNPAUSER);
        bridge.submitSetPauserAddresses(_signAndEncodeVM(payload, 0));
    }
}
