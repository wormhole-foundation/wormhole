// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.0;

import "../contracts/bridge/Bridge.sol";
import "../contracts/bridge/BridgeSetup.sol";
import "../contracts/bridge/BridgeImplementation.sol";
import "../contracts/bridge/TokenBridge.sol";
import "../contracts/interfaces/IWormhole.sol";
import "../contracts/bridge/interfaces/ITokenBridge.sol";
import "../contracts/bridge/token/TokenImplementation.sol";
import "../contracts/bridge/mock/MockBridgeImplementation.sol";
import "../contracts/bridge/mock/MockTokenBridgeIntegration.sol";
import "../contracts/bridge/mock/MockFeeToken.sol";
import "forge-std/Test.sol";
import "./Implementation.t.sol";
import "../contracts/bridge/mock/MockWETH9.sol";

// @dev ensure some internal methods are public for testing
contract ExportedBridge is BridgeImplementation {
    function _truncateAddressPub(bytes32 b) public pure returns (address) {
        return super._truncateAddress(b);
    }

    function setChainIdPub(uint16 chainId) public {
        return super.setChainId(chainId);
    }

    function setEvmChainIdPub(uint256 evmChainId) public {
        return super.setEvmChainId(evmChainId);
    }
}

interface ITokenBridgeTest is ITokenBridge {
    function _truncateAddressPub(bytes32 b) external pure returns (address);

    function setChainIdPub(uint16 chainId) external;

    function setEvmChainIdPub(uint256 evmChainId) external;
}

contract TestBridge is Test {
    BridgeSetup bridgeSetup;
    ExportedBridge bridgeImpl;
    ITokenBridgeTest bridge;
    IWormhole wormhole;
    TestImplementation implementationTest;
    TokenImplementation tokenImpl;
    IERC20 weth;
    uint16 testChainId;
    uint256 testEvmChainId;
    uint16 governanceChainId;
    bytes32 governanceContract;
    uint8 constant finality = 15;

    // "TokenBridge" (left padded)
    bytes32 constant tokenBridgeModule =
        0x000000000000000000000000000000000000000000546f6b656e427269646765;
    uint8 actionRegisterChain = 1;
    uint8 actionContractUpgrade = 2;
    uint8 actionRecoverChainId = 3;

    uint16 fakeChainId = 1337;
    uint256 fakeEvmChainId = 10001;

    uint16 testForeignChainId = 1;
    bytes32 testForeignBridgeContract =
        0x0000000000000000000000000000000000000000000000000000000000000004;
    uint16 testBridgedAssetChain = 1;
    bytes32 testBridgedAssetAddress =
        0x000000000000000000000000b7a2211e8165943192ad04f5dd21bedc29ff003e;

    uint256 public constant testGuardian =
        93941733246223705020089879371323733820373732307041878556247502674739205313440;

    function setUp() public {
        // Setup wormhole
        implementationTest = new TestImplementation();
        implementationTest.setUp();

        // Get wormhole from implementation tests
        wormhole = IWormhole(address(implementationTest.proxied()));

        // Deploy setup
        bridgeSetup = new BridgeSetup();
        // Deploy implementation contract
        bridgeImpl = new ExportedBridge();
        // Deploy token implementation
        tokenImpl = new TokenImplementation();
        // Deploy WETH
        weth = IERC20(address(new MockWETH9()));

        testChainId = implementationTest.testChainId();
        testEvmChainId = implementationTest.testEvmChainId();
        vm.chainId(testEvmChainId);
        governanceChainId = implementationTest.governanceChainId();
        governanceContract = implementationTest.governanceContract();

        bytes memory setupAbi = abi.encodeCall(
            BridgeSetup.setup,
            (
                address(bridgeImpl),
                testChainId,
                address(wormhole),
                governanceChainId,
                governanceContract,
                address(tokenImpl),
                address(weth),
                finality,
                testEvmChainId
            )
        );

        // Deploy proxy
        bridge = ITokenBridgeTest(
            address(new TokenBridge(address(bridgeSetup), setupAbi))
        );
    }

    function testTruncate(bytes32 b) public {
        bool invalidAddress = bytes12(b) != 0;
        if (invalidAddress) {
            vm.expectRevert("invalid EVM address");
        }
        bytes32 converted = bytes32(
            uint256(uint160(bytes20(bridge._truncateAddressPub(b))))
        );

        if (!invalidAddress) {
            require(converted == b, "truncate does not roundrip");
        }
    }

    function testSetEvmChainId() public {
        vm.chainId(1);
        bridge.setChainIdPub(1);
        bridge.setEvmChainIdPub(1);
        assertEq(bridge.chainId(), 1);
        assertEq(bridge.evmChainId(), 1);

        // fork occurs, block.chainid changes
        vm.chainId(10001);

        bridge.setEvmChainIdPub(10001);
        assertEq(bridge.chainId(), 1);
        assertEq(bridge.evmChainId(), 10001);

        // evmChainId must equal block.chainid
        vm.expectRevert("invalid evmChainId");
        bridge.setEvmChainIdPub(1337);
    }

    function testShouldBeInitializedWithTheCorrectSignersAndValues() public {
        assertEq(address(bridge.WETH()), address(weth));
        assertEq(bridge.tokenImplementation(), address(tokenImpl));
        // test beacon functionality
        assertEq(bridge.implementation(), address(tokenImpl));
        assertEq(bridge.chainId(), testChainId);
        assertEq(bridge.evmChainId(), testEvmChainId);
        assertEq(bridge.finality(), finality);

        // governance
        uint16 readGovernanceChainId = bridge.governanceChainId();
        bytes32 readGovernanceContract = bridge.governanceContract();
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

    function testShouldRegisterAForeignBridgeImplementationCorrectly() public {
        bytes memory data = abi.encodePacked(
            tokenBridgeModule,
            actionRegisterChain,
            uint16(0),
            testForeignChainId,
            testForeignBridgeContract
        );
        bytes memory vaa = signAndEncodeVM(
            1,
            1,
            governanceChainId,
            governanceContract,
            0,
            data,
            uint256Array(testGuardian),
            0,
            0
        );

        assertEq(
            bridge.bridgeContracts(testForeignChainId),
            bytes32(
                0x0000000000000000000000000000000000000000000000000000000000000000
            )
        );
        bridge.registerChain(vaa);
        assertEq(
            bridge.bridgeContracts(testForeignChainId),
            testForeignBridgeContract
        );
    }

    function testShouldAcceptAValidUpgrade() public {
        MockBridgeImplementation mock = new MockBridgeImplementation();
        bytes memory data = abi.encodePacked(
            tokenBridgeModule,
            actionContractUpgrade,
            testChainId,
            addressToBytes32(address(mock))
        );
        bytes memory vaa = signAndEncodeVM(
            1,
            1,
            governanceChainId,
            governanceContract,
            0,
            data,
            uint256Array(testGuardian),
            0,
            0
        );

        bytes32 IMPLEMENTATION_STORAGE_SLOT = 0x360894a13ba1a3210667c828492db98dca3e2076cc3735a920a3ca505d382bbc;
        assertEq(
            vm.load(address(bridge), IMPLEMENTATION_STORAGE_SLOT),
            addressToBytes32(address(bridgeImpl))
        );
        bridge.upgrade(vaa);
        assertEq(
            vm.load(address(bridge), IMPLEMENTATION_STORAGE_SLOT),
            addressToBytes32(address(mock))
        );
    }

    function testBridgedTokensShouldOnlyBeMintAndBurnableByOwner() public {
        address owner = address(this);
        address notOwner = address(0x1);
        tokenImpl.initialize("TestToken", "TT", 18, 0, owner, 0, 0x0);
        tokenImpl.mint(owner, 10);
        tokenImpl.burn(owner, 5);

        vm.expectRevert("caller is not the owner");
        vm.prank(address(notOwner));
        tokenImpl.mint(owner, 10);

        vm.expectRevert("caller is not the owner");
        vm.prank(address(notOwner));
        tokenImpl.burn(owner, 5);
    }

    event LogMessagePublished(
        address indexed sender,
        uint64 sequence,
        uint32 nonce,
        bytes payload,
        uint8 consistencyLevel
    );

    function testShouldAttestATokenCorrectly() public {
        tokenImpl.initialize("TestToken", "TT", 18, 0, address(this), 0, 0x0);
        bytes memory attestPayload = abi.encodePacked(
            uint8(2),
            addressToBytes32(address(tokenImpl)),
            testChainId,
            // decimals
            uint8(18),
            // symbol (TT)
            bytes32(
                0x5454000000000000000000000000000000000000000000000000000000000000
            ),
            // name (TestToken)
            bytes32(
                0x54657374546f6b656e0000000000000000000000000000000000000000000000
            )
        );
        vm.expectEmit();
        emit LogMessagePublished(
            address(bridge),
            uint64(0),
            uint32(234),
            attestPayload,
            uint8(finality)
        );
        bridge.attestToken(address(tokenImpl), 234);
    }

    function testShouldCorrectlyDeployAWrappedAssetForATokenAttestation()
        public
    {
        testShouldRegisterAForeignBridgeImplementationCorrectly();
        testShouldAttestATokenCorrectly();
        bytes memory data = abi.encodePacked(
            uint8(2),
            testBridgedAssetAddress,
            testBridgedAssetChain,
            uint8(18),
            // symbol (TT)
            bytes32(
                0x5454000000000000000000000000000000000000000000000000000000000000
            ),
            // name (TestToken)
            bytes32(
                0x54657374546f6b656e0000000000000000000000000000000000000000000000
            )
        );
        bytes memory vaa = signAndEncodeVM(
            0,
            0,
            testForeignChainId,
            testForeignBridgeContract,
            0,
            data,
            uint256Array(testGuardian),
            0,
            0
        );

        bridge.createWrapped(vaa);
        address wrappedAddress = bridge.wrappedAsset(
            testBridgedAssetChain,
            testBridgedAssetAddress
        );
        assertTrue(
            bridge.isWrappedAsset(wrappedAddress),
            "Wrapped asset is not a wrapped asset"
        );

        TokenImplementation wrapped = TokenImplementation(wrappedAddress);

        assertEq(wrapped.symbol(), "TT");
        assertEq(wrapped.name(), "TestToken");
        assertEq(wrapped.decimals(), 18);
        assertEq(wrapped.chainId(), testBridgedAssetChain);
        assertEq(wrapped.nativeContract(), testBridgedAssetAddress);
    }

    function testShouldCorrectlyUpdateAWrappedAssetForATokenAttestation()
        public
    {
        testShouldCorrectlyDeployAWrappedAssetForATokenAttestation();
        bytes memory data = abi.encodePacked(
            uint8(2),
            testBridgedAssetAddress,
            testBridgedAssetChain,
            uint8(18),
            // symbol
            bytes32(
                0x5555000000000000000000000000000000000000000000000000000000000000
            ),
            // name
            bytes32(
                0x5472656500000000000000000000000000000000000000000000000000000000
            )
        );

        bytes memory vaa = signAndEncodeVM(
            0,
            0,
            testForeignChainId,
            testForeignBridgeContract,
            0,
            data,
            uint256Array(testGuardian),
            0,
            0
        );

        vm.expectRevert("current metadata is up to date");
        bridge.updateWrapped(vaa);

        vaa = signAndEncodeVM(
            0,
            0,
            testForeignChainId,
            testForeignBridgeContract,
            1,
            data,
            uint256Array(testGuardian),
            0,
            0
        );

        bridge.updateWrapped(vaa);

        address wrappedAddress = bridge.wrappedAsset(
            testBridgedAssetChain,
            testBridgedAssetAddress
        );
        assertTrue(
            bridge.isWrappedAsset(wrappedAddress),
            "Wrapped asset is not a wrapped asset"
        );

        TokenImplementation wrapped = TokenImplementation(wrappedAddress);

        assertEq(wrapped.symbol(), "UU");
        assertEq(wrapped.name(), "Tree");
        assertEq(wrapped.decimals(), 18);
        assertEq(wrapped.chainId(), testBridgedAssetChain);
        assertEq(wrapped.nativeContract(), testBridgedAssetAddress);
    }

    function testShouldDepositAndLogTransfersCorrectly() public {
        testShouldCorrectlyDeployAWrappedAssetForATokenAttestation();
        uint256 amount = 1_000_000_000_000_000_000;
        uint256 fee = 100_000_000_000_000_000;
        tokenImpl.mint(address(this), amount);
        tokenImpl.approve(address(bridge), amount);

        uint256 accountBalanceBefore = tokenImpl.balanceOf(address(this));
        uint256 bridgeBalanceBefore = tokenImpl.balanceOf(address(bridge));

        assertEq(accountBalanceBefore, amount);
        assertEq(bridgeBalanceBefore, 0);

        uint16 toChain = testForeignChainId;
        bytes32 toAddress = testForeignBridgeContract;

        bytes memory transferPayload = abi.encodePacked(
            uint8(1),
            amount / 1e10,
            addressToBytes32(address(tokenImpl)),
            testChainId,
            toAddress,
            toChain,
            fee / 1e10
        );
        vm.expectEmit();
        emit LogMessagePublished(
            address(bridge),
            uint64(1),
            uint32(234),
            transferPayload,
            uint8(finality)
        );
        bridge.transferTokens(
            address(tokenImpl),
            amount,
            toChain,
            toAddress,
            fee,
            234
        );

        uint256 accountBalanceAfter = tokenImpl.balanceOf(address(this));
        uint256 bridgeBalanceAfter = tokenImpl.balanceOf(address(bridge));

        assertEq(accountBalanceAfter, 0);
        assertEq(bridgeBalanceAfter, amount);
    }

    function testShouldDepositAndLogFeeTokenTransfersCorrectly() public {
        testShouldCorrectlyDeployAWrappedAssetForATokenAttestation();

        uint256 mintAmount = 10_000_000_000_000_000_000;
        uint256 amount = 1_000_000_000_000_000_000;
        uint256 fee = 100_000_000_000_000_000;

        uint16 toChain = testForeignChainId;
        bytes32 toAddress = testForeignBridgeContract;

        FeeToken feeToken = new FeeToken();
        feeToken.initialize("Test", "TST", 18, 123, address(this), 0, 0x0);
        feeToken.mint(address(this), mintAmount);
        feeToken.approve(address(bridge), mintAmount);

        uint256 bridgeBalanceBefore = feeToken.balanceOf(address(bridge));

        uint256 feeAmount = (amount * 9) / 10;

        bytes memory transferPayload = abi.encodePacked(
            uint8(1),
            feeAmount / 1e10,
            addressToBytes32(address(feeToken)),
            testChainId,
            toAddress,
            toChain,
            fee / 1e10
        );
        vm.expectEmit();
        emit LogMessagePublished(
            address(bridge),
            uint64(1),
            uint32(234),
            transferPayload,
            uint8(finality)
        );
        bridge.transferTokens(
            address(feeToken),
            amount,
            toChain,
            toAddress,
            fee,
            234
        );

        uint256 bridgeBalanceAfter = feeToken.balanceOf(address(bridge));
        assertEq(bridgeBalanceAfter, feeAmount, "Bridge balance is incorrect");
    }

    event TransferRedeemed(
        uint16 indexed emitterChainId,
        bytes32 indexed emitterAddress,
        uint64 indexed sequence
    );

    function testShouldTransferOutLockedAssetsForAValidTransferVM() public {
        testShouldDepositAndLogTransfersCorrectly();

        uint256 amount = 1_000_000_000_000_000_000;
        uint64 sequence = 1697;

        uint256 accountBalanceBefore = tokenImpl.balanceOf(address(this));
        uint256 bridgeBalanceBefore = tokenImpl.balanceOf(address(bridge));
        assertEq(accountBalanceBefore, 0);
        assertEq(bridgeBalanceBefore, amount);

        bytes memory transferPayload = abi.encodePacked(
            uint8(1),
            amount / 1e10,
            addressToBytes32(address(tokenImpl)),
            testChainId,
            addressToBytes32(address(this)),
            testChainId,
            uint256(0)
        );

        bytes memory vaa = signAndEncodeVM(
            0,
            0,
            testForeignChainId,
            testForeignBridgeContract,
            sequence,
            transferPayload,
            uint256Array(testGuardian),
            0,
            0
        );

        vm.expectEmit();
        emit TransferRedeemed(
            testForeignChainId,
            testForeignBridgeContract,
            sequence
        );
        bridge.completeTransfer(vaa);

        uint256 accountBalanceAfter = tokenImpl.balanceOf(address(this));
        uint256 bridgeBalanceAfter = tokenImpl.balanceOf(address(bridge));
        assertEq(accountBalanceAfter, amount);
        assertEq(bridgeBalanceAfter, 0);
    }

    function testShouldDepositAndLogTransferWithPayloadCorrectly() public {
        testShouldCorrectlyDeployAWrappedAssetForATokenAttestation();
        uint256 amount = 1_000_000_000_000_000_000;
        tokenImpl.mint(address(this), amount);
        tokenImpl.approve(address(bridge), amount);

        uint256 accountBalanceBefore = tokenImpl.balanceOf(address(this));
        uint256 bridgeBalanceBefore = tokenImpl.balanceOf(address(bridge));

        assertEq(accountBalanceBefore, amount);
        assertEq(bridgeBalanceBefore, 0);

        bytes memory additionalPayload = bytes("abc123");

        uint16 toChain = testForeignChainId;
        bytes32 toAddress = testForeignBridgeContract;

        bytes memory transferPayload = abi.encodePacked(
            uint8(3),
            amount / 1e10,
            addressToBytes32(address(tokenImpl)),
            testChainId,
            toAddress,
            toChain,
            addressToBytes32(address(this)),
            additionalPayload
        );
        vm.expectEmit();
        emit LogMessagePublished(
            address(bridge),
            uint64(1),
            uint32(234),
            transferPayload,
            uint8(finality)
        );
        bridge.transferTokensWithPayload(
            address(tokenImpl),
            amount,
            toChain,
            toAddress,
            234,
            additionalPayload
        );

        uint256 accountBalanceAfter = tokenImpl.balanceOf(address(this));
        uint256 bridgeBalanceAfter = tokenImpl.balanceOf(address(bridge));

        assertEq(accountBalanceAfter, 0);
        assertEq(bridgeBalanceAfter, amount);
    }

    function testShouldTransferOutLockedAssetsForAValidTransferWithPayloadVM()
        public
    {
        testShouldDepositAndLogTransfersCorrectly();

        uint256 amount = 1_000_000_000_000_000_000;
        uint64 sequence = 1111;

        uint256 accountBalanceBefore = tokenImpl.balanceOf(address(this));
        uint256 bridgeBalanceBefore = tokenImpl.balanceOf(address(bridge));
        assertEq(accountBalanceBefore, 0);
        assertEq(bridgeBalanceBefore, amount);

        bytes memory transferPayload = abi.encodePacked(
            uint8(3),
            amount / 1e10,
            addressToBytes32(address(tokenImpl)),
            testChainId,
            addressToBytes32(address(this)),
            testChainId,
            uint256(0),
            bytes("abc123")
        );

        bytes memory vaa = signAndEncodeVM(
            0,
            0,
            testForeignChainId,
            testForeignBridgeContract,
            sequence,
            transferPayload,
            uint256Array(testGuardian),
            0,
            0
        );

        vm.expectEmit();
        emit TransferRedeemed(
            testForeignChainId,
            testForeignBridgeContract,
            sequence
        );
        bridge.completeTransferWithPayload(vaa);

        uint256 accountBalanceAfter = tokenImpl.balanceOf(address(this));
        uint256 bridgeBalanceAfter = tokenImpl.balanceOf(address(bridge));
        assertEq(accountBalanceAfter, amount);
        assertEq(bridgeBalanceAfter, 0);
    }

    function testShouldMintBridgedAssetWrappersOnTransferFromAnotherChainAndHandleFeesCorrectly()
        public
    {
        testShouldTransferOutLockedAssetsForAValidTransferWithPayloadVM();

        uint256 amount = 1_000_000_000_000_000_000;
        uint256 fee = 100_000_000_000_000_000;

        address wrappedAddress = bridge.wrappedAsset(
            testBridgedAssetChain,
            testBridgedAssetAddress
        );
        TokenImplementation wrapped = TokenImplementation(wrappedAddress);

        assertEq(wrapped.totalSupply(), 0, "Wrong total supply");

        bytes memory data = abi.encodePacked(
            uint8(1),
            amount / 1e10,
            testBridgedAssetAddress,
            testBridgedAssetChain,
            addressToBytes32(address(this)),
            testChainId,
            fee / 1e10
        );

        bytes memory vaa = signAndEncodeVM(
            0,
            0,
            testForeignChainId,
            testForeignBridgeContract,
            0,
            data,
            uint256Array(testGuardian),
            0,
            0
        );

        address sender = address(0x1);
        vm.prank(sender);
        bridge.completeTransfer(vaa);

        uint256 accountBalanceAfter = wrapped.balanceOf(address(this));
        uint256 senderBalanceAfter = wrapped.balanceOf(address(sender));
        assertEq(accountBalanceAfter, amount - fee);
        assertEq(senderBalanceAfter, fee);
        assertEq(wrapped.totalSupply(), amount);

        vm.prank(sender);
        wrapped.transfer(address(this), fee);
    }

    function testShouldNotAllowARedemptionFromMsgSenderOtherThanToOnTokenBridgeTransferWithPayload()
        public
    {
        testShouldTransferOutLockedAssetsForAValidTransferWithPayloadVM();

        uint256 amount = 1_000_000_000_000_000_000;

        address wrappedAddress = bridge.wrappedAsset(
            testBridgedAssetChain,
            testBridgedAssetAddress
        );
        TokenImplementation wrapped = TokenImplementation(wrappedAddress);

        assertEq(wrapped.totalSupply(), 0, "Wrong total supply");

        address fromAddress = address(0x2);
        bytes memory additionalPayload = bytes("abc123");

        bytes memory data = abi.encodePacked(
            uint8(3),
            amount / 1e10,
            testBridgedAssetAddress,
            testBridgedAssetChain,
            addressToBytes32(address(this)),
            testChainId,
            addressToBytes32(fromAddress),
            additionalPayload
        );

        bytes memory vaa = signAndEncodeVM(
            0,
            0,
            testForeignChainId,
            testForeignBridgeContract,
            0,
            data,
            uint256Array(testGuardian),
            0,
            0
        );

        address sender = address(0x1);

        vm.expectRevert("invalid sender");
        vm.prank(sender);
        bridge.completeTransferWithPayload(vaa);
    }

    function testShouldAllowARedemptionFromMsgSenderIsToOnTokenBridgeTransferWithPayloadAndCheckThatSenderReceivesFees()
        public
    {
        testShouldTransferOutLockedAssetsForAValidTransferWithPayloadVM();

        uint256 amount = 1_000_000_000_000_000_000;

        address wrappedAddress = bridge.wrappedAsset(
            testBridgedAssetChain,
            testBridgedAssetAddress
        );
        TokenImplementation wrapped = TokenImplementation(wrappedAddress);

        assertEq(wrapped.totalSupply(), 0, "Wrong total supply");

        address fromAddress = address(0x2);
        bytes memory additionalPayload = bytes("abc123");

        bytes memory data = abi.encodePacked(
            uint8(3),
            amount / 1e10,
            testBridgedAssetAddress,
            testBridgedAssetChain,
            addressToBytes32(address(this)),
            testChainId,
            addressToBytes32(fromAddress),
            additionalPayload
        );

        bytes memory vaa = signAndEncodeVM(
            0,
            0,
            testForeignChainId,
            testForeignBridgeContract,
            0,
            data,
            uint256Array(testGuardian),
            0,
            0
        );

        bridge.completeTransferWithPayload(vaa);

        uint256 accountBalanceAfter = wrapped.balanceOf(address(this));
        assertEq(accountBalanceAfter, amount);
        assertEq(wrapped.totalSupply(), amount);
    }

    function testShouldBurnBridgedAssetsWrappersOnTransferToAnotherChain()
        public
    {
        testShouldMintBridgedAssetWrappersOnTransferFromAnotherChainAndHandleFeesCorrectly();

        uint256 amount = 1_000_000_000_000_000_000;

        address wrappedAddress = bridge.wrappedAsset(
            testBridgedAssetChain,
            testBridgedAssetAddress
        );
        TokenImplementation wrapped = TokenImplementation(wrappedAddress);
        wrapped.approve(address(bridge), amount);

        assertEq(wrapped.balanceOf(address(this)), amount);
        assertEq(wrapped.totalSupply(), amount);

        bridge.transferTokens(
            wrappedAddress,
            amount,
            11,
            testForeignBridgeContract,
            0,
            234
        );

        assertEq(wrapped.balanceOf(address(this)), 0);
        assertEq(wrapped.balanceOf(address(bridge)), 0);
        assertEq(wrapped.totalSupply(), 0);
    }

    function testShouldHandleETHDepositsCorrectly() public {
        testShouldRegisterAForeignBridgeImplementationCorrectly();
        uint256 amount = 1_000_000_000_000_000_000;
        uint256 fee = 100_000_000_000_000_000;

        vm.deal(address(this), amount);

        uint256 totalWETHsupply = weth.totalSupply();
        uint256 bridgeBalanceBefore = weth.balanceOf(address(bridge));

        assertEq(totalWETHsupply, 0);
        assertEq(bridgeBalanceBefore, 0);

        uint16 toChain = testForeignChainId;
        bytes32 toAddress = testForeignBridgeContract;

        bytes memory transferPayload = abi.encodePacked(
            uint8(1),
            amount / 1e10,
            addressToBytes32(address(weth)),
            testChainId,
            toAddress,
            toChain,
            fee / 1e10
        );
        vm.expectEmit();
        emit LogMessagePublished(
            address(bridge),
            uint64(0),
            uint32(234),
            transferPayload,
            uint8(finality)
        );
        bridge.wrapAndTransferETH{value: amount}(toChain, toAddress, fee, 234);

        uint256 totalWETHSupplyAfter = weth.totalSupply();
        uint256 bridgeBalanceAfter = weth.balanceOf(address(bridge));

        assertEq(totalWETHSupplyAfter, amount);
        assertEq(bridgeBalanceAfter, amount);
    }

    function testShouldHandleETHWithdrawalsAndFeesCorrectly() public {
        testShouldHandleETHDepositsCorrectly();
        uint256 amount = 1_000_000_000_000_000_000;
        uint256 fee = 500_000_000_000_000_000;
        uint64 sequence = 235;
        address feeRecipient = address(
            0x1234123412341234123412341234123412341234
        );

        uint256 accountBalanceBefore = weth.balanceOf(address(this));
        uint256 feeRecipientBalanceBefore = weth.balanceOf(feeRecipient);
        uint256 bridgeBalanceBefore = weth.balanceOf(address(bridge));
        assertEq(accountBalanceBefore, 0);
        assertEq(feeRecipientBalanceBefore, 0);
        assertEq(bridgeBalanceBefore, amount);

        bytes memory transferPayload = abi.encodePacked(
            uint8(1),
            amount / 1e10,
            addressToBytes32(address(weth)),
            testChainId,
            addressToBytes32(address(this)),
            testChainId,
            fee / 1e10
        );

        bytes memory vaa = signAndEncodeVM(
            0,
            0,
            testForeignChainId,
            testForeignBridgeContract,
            sequence,
            transferPayload,
            uint256Array(testGuardian),
            0,
            0
        );

        vm.expectEmit();
        emit TransferRedeemed(
            testForeignChainId,
            testForeignBridgeContract,
            sequence
        );
        vm.prank(feeRecipient);
        bridge.completeTransferAndUnwrapETH(vaa);

        uint256 totalSupplyAfter = weth.totalSupply();
        assertEq(totalSupplyAfter, 0);

        uint256 bridgeBalanceAfter = weth.balanceOf(address(bridge));
        uint256 accountBalanceAfter = address(this).balance;
        uint256 feeRecipientBalanceAfter = feeRecipient.balance;
        assertEq(accountBalanceAfter, amount - fee);
        assertEq(bridgeBalanceAfter, 0);
        assertEq(feeRecipientBalanceAfter, fee);
    }

    function testShouldHandleETHDepositsWithPayloadCorrectly() public {
        testShouldRegisterAForeignBridgeImplementationCorrectly();
        uint256 amount = 1_000_000_000_000_000_000;

        vm.deal(address(this), amount);

        uint256 totalWETHsupply = weth.totalSupply();
        uint256 bridgeBalanceBefore = weth.balanceOf(address(bridge));

        assertEq(totalWETHsupply, 0);
        assertEq(bridgeBalanceBefore, 0);

        uint16 toChain = testForeignChainId;
        bytes32 toAddress = testForeignBridgeContract;

        bytes memory additionalPayload = bytes("abc123");

        bytes memory transferPayload = abi.encodePacked(
            uint8(3),
            amount / 1e10,
            addressToBytes32(address(weth)),
            testChainId,
            toAddress,
            toChain,
            addressToBytes32(address(this)),
            additionalPayload
        );
        vm.expectEmit();
        emit LogMessagePublished(
            address(bridge),
            uint64(0),
            uint32(234),
            transferPayload,
            uint8(finality)
        );
        bridge.wrapAndTransferETHWithPayload{value: amount}(
            toChain,
            toAddress,
            234,
            additionalPayload
        );

        uint256 totalWETHSupplyAfter = weth.totalSupply();
        uint256 bridgeBalanceAfter = weth.balanceOf(address(bridge));

        assertEq(totalWETHSupplyAfter, amount);
        assertEq(bridgeBalanceAfter, amount);
    }

    function testShouldHandleETHWithdrawalsWithPayloadCorrectly() public {
        testShouldHandleETHDepositsWithPayloadCorrectly();
        uint256 amount = 1_000_000_000_000_000_000;
        uint64 sequence = 235;

        uint256 accountBalanceBefore = weth.balanceOf(address(this));
        uint256 bridgeBalanceBefore = weth.balanceOf(address(bridge));
        assertEq(accountBalanceBefore, 0);
        assertEq(bridgeBalanceBefore, amount);
        assertEq(weth.totalSupply(), amount);

        address receiver = address(0x2);

        bytes memory additionalPayload = bytes("abc123");

        bytes memory transferPayload = abi.encodePacked(
            uint8(3),
            amount / 1e10,
            addressToBytes32(address(weth)),
            testChainId,
            addressToBytes32(receiver),
            testChainId,
            uint256(0),
            additionalPayload
        );

        bytes memory vaa = signAndEncodeVM(
            0,
            0,
            testForeignChainId,
            testForeignBridgeContract,
            sequence,
            transferPayload,
            uint256Array(testGuardian),
            0,
            0
        );

        vm.expectEmit();
        emit TransferRedeemed(
            testForeignChainId,
            testForeignBridgeContract,
            sequence
        );
        vm.prank(receiver);
        bridge.completeTransferAndUnwrapETHWithPayload(vaa);

        uint256 totalSupplyAfter = weth.totalSupply();
        assertEq(totalSupplyAfter, 0);

        uint256 bridgeBalanceAfter = weth.balanceOf(address(bridge));
        uint256 receiverBalanceAfter = address(receiver).balance;
        assertEq(receiverBalanceAfter, amount);
        assertEq(bridgeBalanceAfter, 0);
    }

    function testShouldRevertOnTransferOutOfATotalOfMaxUint64Tokens() public {
        uint256 amount = 184467440737095516160000000000;
        uint256 firstTransfer = 1000000000000;

        testShouldCorrectlyDeployAWrappedAssetForATokenAttestation();
        tokenImpl.mint(address(this), amount);
        tokenImpl.approve(address(bridge), amount);

        uint16 toChain = testForeignChainId;
        bytes32 toAddress = testForeignBridgeContract;

        bridge.transferTokens(
            address(tokenImpl),
            firstTransfer,
            toChain,
            toAddress,
            0,
            234
        );

        vm.expectRevert(
            "transfer exceeds max outstanding bridged token amount"
        );
        bridge.transferTokens(
            address(tokenImpl),
            amount - firstTransfer,
            toChain,
            toAddress,
            0,
            234
        );
    }

    function addressToBytes32(address input) internal returns (bytes32 output) {
        return bytes32(uint256(uint160(input)));
    }

    function uint256Array(
        uint256 member
    ) internal returns (uint256[] memory arr) {
        arr = new uint256[](1);
        arr[0] = member;
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
    ) public returns (bytes memory signedMessage) {
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

    function testShouldRejectSmartContractUpgradesOnForks() public {
        uint32 timestamp = 1000;
        uint32 nonce = 1001;

        // Perform a successful upgrade
        MockBridgeImplementation mock = new MockBridgeImplementation();

        bytes memory data = abi.encodePacked(
            tokenBridgeModule,
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
        bytes32 before = vm.load(address(bridge), IMPLEMENTATION_STORAGE_SLOT);

        bridge.upgrade(vaa);

        bytes32 afterUpgrade = vm.load(
            address(bridge),
            IMPLEMENTATION_STORAGE_SLOT
        );
        assertEq(afterUpgrade, addressToBytes32(address(mock)));
        assertEq(
            MockBridgeImplementation(payable(address(bridge)))
                .testNewImplementationActive(),
            true,
            "New implementation not active"
        );

        // Overwrite EVM Chain ID
        MockBridgeImplementation(payable(address(bridge)))
            .testOverwriteEVMChainId(fakeChainId, fakeEvmChainId);
        assertEq(
            bridge.chainId(),
            fakeChainId,
            "Overwrite didn't work for chain ID"
        );
        assertEq(
            bridge.evmChainId(),
            fakeEvmChainId,
            "Overwrite didn't work for evm chain ID"
        );

        data = abi.encodePacked(
            tokenBridgeModule,
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
        bridge.upgrade(vaa);
    }

    function testShouldAllowRecoverChainIDGovernancePacketsForks() public {
        uint32 timestamp = 1000;
        uint32 nonce = 1001;

        // Perform a successful upgrade
        MockBridgeImplementation mock = new MockBridgeImplementation();

        bytes memory data = abi.encodePacked(
            tokenBridgeModule,
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
        bytes32 before = vm.load(address(bridge), IMPLEMENTATION_STORAGE_SLOT);

        bridge.upgrade(vaa);

        bytes32 afterUpgrade = vm.load(
            address(bridge),
            IMPLEMENTATION_STORAGE_SLOT
        );
        assertEq(afterUpgrade, bytes32(uint256(uint160(address(mock)))));
        assertEq(
            MockBridgeImplementation(payable(address(bridge)))
                .testNewImplementationActive(),
            true,
            "New implementation not active"
        );

        // Overwrite EVM Chain ID
        MockBridgeImplementation(payable(address(bridge)))
            .testOverwriteEVMChainId(fakeChainId, fakeEvmChainId);
        assertEq(
            bridge.chainId(),
            fakeChainId,
            "Overwrite didn't work for chain ID"
        );
        assertEq(
            bridge.evmChainId(),
            fakeEvmChainId,
            "Overwrite didn't work for evm chain ID"
        );

        // recover chain ID
        data = abi.encodePacked(
            tokenBridgeModule,
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

        bridge.submitRecoverChainId(vaa);

        assertEq(
            bridge.chainId(),
            testChainId,
            "Recover didn't work for chain ID"
        );
        assertEq(
            bridge.evmChainId(),
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
        MockBridgeImplementation mock = new MockBridgeImplementation();

        bytes memory data = abi.encodePacked(
            tokenBridgeModule,
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
        bytes32 before = vm.load(address(bridge), IMPLEMENTATION_STORAGE_SLOT);

        bridge.upgrade(vaa);

        bytes32 afterUpgrade = vm.load(
            address(bridge),
            IMPLEMENTATION_STORAGE_SLOT
        );
        assertEq(afterUpgrade, bytes32(uint256(uint160(address(mock)))));
        assertEq(
            MockBridgeImplementation(payable(address(bridge)))
                .testNewImplementationActive(),
            true,
            "New implementation not active"
        );

        // Overwrite EVM Chain ID
        MockBridgeImplementation(payable(address(bridge)))
            .testOverwriteEVMChainId(fakeChainId, fakeEvmChainId);
        assertEq(
            bridge.chainId(),
            fakeChainId,
            "Overwrite didn't work for chain ID"
        );
        assertEq(
            bridge.evmChainId(),
            fakeEvmChainId,
            "Overwrite didn't work for evm chain ID"
        );

        // recover chain ID
        data = abi.encodePacked(
            tokenBridgeModule,
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

        bridge.submitRecoverChainId(vaa);

        assertEq(
            bridge.chainId(),
            testChainId,
            "Recover didn't work for chain ID"
        );
        assertEq(
            bridge.evmChainId(),
            testEvmChainId,
            "Recover didn't work for evm chain ID"
        );

        // Perform a successful upgrade
        mock = new MockBridgeImplementation();

        data = abi.encodePacked(
            tokenBridgeModule,
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

        before = vm.load(address(bridge), IMPLEMENTATION_STORAGE_SLOT);

        bridge.upgrade(vaa);

        afterUpgrade = vm.load(address(bridge), IMPLEMENTATION_STORAGE_SLOT);
        assertEq(afterUpgrade, bytes32(uint256(uint160(address(mock)))));
        assertEq(
            MockBridgeImplementation(payable(address(bridge)))
                .testNewImplementationActive(),
            true,
            "New implementation not active"
        );
    }

    fallback() external payable {}
}
