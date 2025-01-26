// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.0;

import "../contracts/nft/NFTBridge.sol";
import "../contracts/nft/NFTBridgeSetup.sol";
import "../contracts/nft/NFTBridgeImplementation.sol";
import "../contracts/nft/NFTBridgeEntrypoint.sol";
import "../contracts/interfaces/IWormhole.sol";
import "../contracts/nft/interfaces/INFTBridge.sol";

import "../contracts/nft/interfaces/INFTBridge.sol";
import "../contracts/nft/token/NFTImplementation.sol";
import "../contracts/nft/mock/MockNFTImplementation.sol";
import "../contracts/nft/mock/MockNFTBridgeImplementation.sol";
import "forge-std/Test.sol";
import "./Implementation.t.sol";
import "../contracts/bridge/mock/MockWETH9.sol";

contract TestNFTBridge is Test {
    NFTBridgeSetup bridgeSetup;
    NFTBridgeImplementation bridgeImpl;
    NFTImplementation tokenImpl;
    INFTBridge bridge;
    IWormhole wormhole;

    TestImplementation implementationTest;

    uint16 testChainId;
    uint256 testEvmChainId;
    uint16 governanceChainId;
    bytes32 governanceContract;
    uint8 constant finality = 15;

    // "NFTBridge" (left padded)
    bytes32 constant NFTBridgeModule =
        0x00000000000000000000000000000000000000000000004e4654427269646765;
    uint8 actionRegisterChain = 1;
    uint8 actionContractUpgrade = 2;
    uint8 actionRecoverChainId = 3;

    uint16 fakeChainId = 1337;
    uint256 fakeEvmChainId = 10001;

    uint16 testForeignChainId = 1;
    bytes32 testForeignBridgeContract =
        0x000000000000000000000000000000000000000000000000000000000000ffff;
    uint16 testBridgedAssetChain = 3;
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
        bridgeSetup = new NFTBridgeSetup();
        // Deploy implementation contract
        bridgeImpl = new NFTBridgeImplementation();
        // Deploy token implementation
        tokenImpl = new NFTImplementation();

        testChainId = implementationTest.testChainId();
        testEvmChainId = implementationTest.testEvmChainId();
        vm.chainId(testEvmChainId);
        governanceChainId = implementationTest.governanceChainId();
        governanceContract = implementationTest.governanceContract();

        bytes memory setupAbi = abi.encodeWithSelector(
            NFTBridgeSetup.setup.selector,
            address(bridgeImpl),
            testChainId,
            address(wormhole),
            governanceChainId,
            governanceContract,
            address(tokenImpl),
            finality,
            testEvmChainId
        );

        // Deploy proxy
        bridge = INFTBridge(
            address(new NFTBridgeEntrypoint(address(bridgeSetup), setupAbi))
        );
    }

    function testShouldBeInitializedWithTheCorrectSignersAndValues() public {
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
            NFTBridgeModule,
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
        MockNFTBridgeImplementation mock = new MockNFTBridgeImplementation();
        bytes memory data = abi.encodePacked(
            NFTBridgeModule,
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

        assertTrue(
            MockNFTBridgeImplementation(address(bridge))
                .testNewImplementationActive(),
            "implementation not active"
        );
    }

    function testBridgedTokensShouldOnlyBeMintAndBurnableByOwner() public {
        address owner = address(this);
        address notOwner = address(0x1);
        tokenImpl.initialize("TestToken", "TT", owner, 0, 0x0);
        tokenImpl.mint(owner, 10, "");

        vm.expectRevert("caller is not the owner");
        vm.prank(address(notOwner));
        tokenImpl.mint(owner, 11, "");

        vm.expectRevert("caller is not the owner");
        vm.prank(address(notOwner));
        tokenImpl.burn(10);

        tokenImpl.burn(10);
    }

    event LogMessagePublished(
        address indexed sender,
        uint64 sequence,
        uint32 nonce,
        bytes payload,
        uint8 consistencyLevel
    );

    function testShouldDepositAndLogTransfersCorrectly() public {
        testShouldRegisterAForeignBridgeImplementationCorrectly();
        testBridgedTokensShouldOnlyBeMintAndBurnableByOwner();

        uint256 tokenId = 1000000000000000000;

        tokenImpl.mint(address(this), tokenId, "abcd");
        tokenImpl.approve(address(bridge), tokenId);

        address ownerBefore = tokenImpl.ownerOf(tokenId);
        assertEq(ownerBefore, address(this));

        uint16 toChain = testForeignChainId;
        bytes32 toAddress = testForeignBridgeContract;

        bytes memory transferPayload = abi.encodePacked(
            uint8(1),
            addressToBytes32(address(tokenImpl)),
            testChainId,
            bytes32("TT"),
            bytes32("TestToken"),
            tokenId,
            uint8(4),
            hex"61626364",
            toAddress,
            toChain
        );
        vm.expectEmit();
        emit LogMessagePublished(
            address(bridge),
            uint64(0),
            uint32(234),
            transferPayload,
            uint8(finality)
        );
        bridge.transferNFT(
            address(tokenImpl),
            tokenId,
            toChain,
            toAddress,
            234
        );

        address ownerAfter = tokenImpl.ownerOf(tokenId);
        assertEq(ownerAfter, address(bridge));
    }

    function testShouldTransferOutLockedAssetsForAValidTransferVM() public {
        testShouldDepositAndLogTransfersCorrectly();

        uint256 tokenId = 1000000000000000000;

        uint16 toChain = testChainId;
        bytes32 toAddress = addressToBytes32(address(this));

        address ownerBefore = tokenImpl.ownerOf(tokenId);
        assertEq(ownerBefore, address(bridge));

        bytes memory transferPayload = abi.encodePacked(
            uint8(1),
            addressToBytes32(address(tokenImpl)),
            testChainId,
            bytes32(0x0),
            bytes32(0x0),
            tokenId,
            uint8(0),
            hex"",
            toAddress,
            toChain
        );

        bytes memory vaa = signAndEncodeVM(
            0,
            0,
            testForeignChainId,
            testForeignBridgeContract,
            294, // sequence
            transferPayload,
            uint256Array(testGuardian),
            0,
            0
        );

        bridge.completeTransfer(vaa);

        address ownerAfter = tokenImpl.ownerOf(tokenId);
        assertEq(ownerAfter, address(this));
    }

    function testShouldMintBridgedAssetWrappersOnTransferFromAnotherChainAndHandleFeesCorrectly()
        public
    {
        testShouldTransferOutLockedAssetsForAValidTransferVM();

        uint256 tokenId = 1000000000000000001;

        bytes memory transferPayload = abi.encodePacked(
            uint8(1),
            testBridgedAssetAddress,
            testBridgedAssetChain,
            // symbol
            bytes32(
                0x464f520000000000000000000000000000000000000000000000000000000000
            ),
            // name
            bytes32(
                0x466f726569676e20436861696e204e4654000000000000000000000000000000
            ),
            tokenId,
            // no URL
            uint8(0),
            hex"",
            addressToBytes32(address(this)),
            testChainId
        );

        bytes memory vaa = signAndEncodeVM(
            0,
            0,
            testForeignChainId,
            testForeignBridgeContract,
            0,
            transferPayload,
            uint256Array(testGuardian),
            0,
            0
        );

        address sender = address(0x1);
        vm.prank(sender);
        bridge.completeTransfer(vaa);

        address wrappedAddress = bridge.wrappedAsset(
            testBridgedAssetChain,
            testBridgedAssetAddress
        );
        NFTImplementation wrapped = NFTImplementation(wrappedAddress);

        assertTrue(
            bridge.isWrappedAsset(address(wrapped)),
            "not wrapped asset"
        );

        address ownerAfter = wrapped.ownerOf(tokenId);
        assertEq(ownerAfter, address(this));

        assertEq(wrapped.symbol(), "FOR");
        assertEq(wrapped.name(), "Foreign Chain NFT");
        assertEq(wrapped.chainId(), testBridgedAssetChain);
        assertEq(wrapped.nativeContract(), testBridgedAssetAddress);

        tokenId = 1000000000000000002;
        transferPayload = abi.encodePacked(
            uint8(1),
            testBridgedAssetAddress,
            testBridgedAssetChain,
            // symbol
            bytes32(
                0x464f520000000000000000000000000000000000000000000000000000000000
            ),
            // name
            bytes32(
                0x466f726569676e20436861696e204e4654000000000000000000000000000000
            ),
            tokenId,
            // no URL
            uint8(0),
            hex"",
            addressToBytes32(address(this)),
            testChainId
        );

        vaa = signAndEncodeVM(
            0,
            0,
            testForeignChainId,
            testForeignBridgeContract,
            0,
            transferPayload,
            uint256Array(testGuardian),
            0,
            0
        );

        sender = address(0x1);
        vm.prank(sender);
        bridge.completeTransfer(vaa);

        ownerAfter = wrapped.ownerOf(tokenId);
        assertEq(ownerAfter, address(this));
    }

    function testShouldMintBridgedAssetsFromSolanaUnderUnifiedNameCachingTheOriginal()
        public
    {
        testShouldTransferOutLockedAssetsForAValidTransferVM();

        uint256 tokenId = 1000000000000000001;

        bytes memory transferPayload = abi.encodePacked(
            uint8(1),
            testBridgedAssetAddress,
            uint16(1), // solana
            // symbol
            bytes32(
                0x464f520000000000000000000000000000000000000000000000000000000000
            ),
            // name
            bytes32(
                0x466f726569676e20436861696e204e4654000000000000000000000000000000
            ),
            tokenId,
            // no URL
            uint8(0),
            hex"",
            addressToBytes32(address(this)),
            testChainId
        );

        bytes memory vaa = signAndEncodeVM(
            0,
            0,
            testForeignChainId,
            testForeignBridgeContract,
            0,
            transferPayload,
            uint256Array(testGuardian),
            0,
            0
        );

        address sender = address(0x1);
        vm.prank(sender);
        bridge.completeTransfer(vaa);

        address wrappedAddress = bridge.wrappedAsset(
            1,
            testBridgedAssetAddress
        );
        NFTImplementation wrapped = NFTImplementation(wrappedAddress);

        INFTBridge.SPLCache memory cache = bridge.splCache(tokenId);
        assertEq(
            cache.symbol,
            bytes32(
                0x464f520000000000000000000000000000000000000000000000000000000000
            )
        );
        assertEq(
            cache.name,
            bytes32(
                0x466f726569676e20436861696e204e4654000000000000000000000000000000
            )
        );

        address ownerAfter = wrapped.ownerOf(tokenId);
        assertEq(ownerAfter, address(this));

        assertEq(wrapped.symbol(), "WORMSPLNFT");
        assertEq(wrapped.name(), "Wormhole Bridged Solana-NFT");
        assertEq(wrapped.chainId(), 1);
        assertEq(wrapped.nativeContract(), testBridgedAssetAddress);
    }

    function testCachedSPLNamesAreLoadedWhenTransferringOutCacheIsCleared()
        public
    {
        testShouldMintBridgedAssetsFromSolanaUnderUnifiedNameCachingTheOriginal();
        address wrappedAddress = bridge.wrappedAsset(
            1,
            testBridgedAssetAddress
        );
        NFTImplementation wrapped = NFTImplementation(wrappedAddress);

        uint256 tokenId = 1000000000000000001;

        bytes memory transferPayload = abi.encodePacked(
            uint8(1),
            testBridgedAssetAddress,
            uint16(1),
            bytes32(
                0x464f520000000000000000000000000000000000000000000000000000000000
            ),
            bytes32(
                0x466f726569676e20436861696e204e4654000000000000000000000000000000
            ),
            tokenId,
            uint8(0),
            hex"",
            testBridgedAssetAddress, // to address
            testBridgedAssetChain // to chain
        );

        wrapped.approve(address(bridge), tokenId);

        vm.expectEmit();
        emit LogMessagePublished(
            address(bridge),
            uint64(1),
            uint32(2345),
            transferPayload,
            uint8(finality)
        );
        bridge.transferNFT(
            wrappedAddress,
            tokenId,
            testBridgedAssetChain, // to chain
            testBridgedAssetAddress, // to address
            2345
        );

        INFTBridge.SPLCache memory cache = bridge.splCache(tokenId);
        assertEq(
            cache.symbol,
            bytes32(
                0x0000000000000000000000000000000000000000000000000000000000000000
            )
        );
        assertEq(
            cache.name,
            bytes32(
                0x0000000000000000000000000000000000000000000000000000000000000000
            )
        );
    }

    function testShouldFailDepositUnapprovedNFTs() public {
        testShouldRegisterAForeignBridgeImplementationCorrectly();
        testBridgedTokensShouldOnlyBeMintAndBurnableByOwner();

        uint256 tokenId = 1000000000000000000;

        tokenImpl.mint(address(this), tokenId, "abcd");
        // tokenImpl.approve(address(bridge), tokenId);

        address ownerBefore = tokenImpl.ownerOf(tokenId);
        assertEq(ownerBefore, address(this));

        uint16 toChain = testForeignChainId;
        bytes32 toAddress = testForeignBridgeContract;

        vm.expectRevert("ERC721: transfer caller is not owner nor approved");
        bridge.transferNFT(
            address(tokenImpl),
            tokenId,
            toChain,
            toAddress,
            234
        );
    }

    function testShouldRefuseToBurnWrappersNotHeldByMsgSender() public {
        testShouldMintBridgedAssetWrappersOnTransferFromAnotherChainAndHandleFeesCorrectly();
        address wrappedAddress = bridge.wrappedAsset(
            testBridgedAssetChain,
            testBridgedAssetAddress
        );
        NFTImplementation wrapped = NFTImplementation(wrappedAddress);

        uint256 tokenId = 1000000000000000001;

        wrapped.approve(address(bridge), tokenId);

        vm.expectRevert("ERC721: transfer of token that is not own");
        vm.prank(address(0x1));
        bridge.transferNFT(
            wrappedAddress,
            tokenId,
            testBridgedAssetChain, // to chain
            testBridgedAssetAddress, // to address
            2345
        );
    }

    function testShouldDepositAndBurnApprovedBridgedAssetWrapperOnTransferToAnotherChain()
        public
    {
        testShouldMintBridgedAssetWrappersOnTransferFromAnotherChainAndHandleFeesCorrectly();
        address wrappedAddress = bridge.wrappedAsset(
            testBridgedAssetChain,
            testBridgedAssetAddress
        );
        NFTImplementation wrapped = NFTImplementation(wrappedAddress);

        uint256 tokenId = 1000000000000000001;

        wrapped.approve(address(bridge), tokenId);

        bridge.transferNFT(
            wrappedAddress,
            tokenId,
            testBridgedAssetChain, // to chain
            testBridgedAssetAddress, // to address
            2345
        );

        vm.expectRevert("ERC721: owner query for nonexistent token");
        wrapped.ownerOf(tokenId);
    }

    function addressToBytes32(address input) internal pure returns (bytes32 output) {
        return bytes32(uint256(uint160(input)));
    }

    function uint256Array(
        uint256 member
    ) internal pure returns (uint256[] memory arr) {
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

    function testShouldRejectSmartContractUpgradesOnForks() public {
        uint32 timestamp = 1000;
        uint32 nonce = 1001;

        // Perform a successful upgrade
        MockNFTBridgeImplementation mock = new MockNFTBridgeImplementation();

        bytes memory data = abi.encodePacked(
            NFTBridgeModule,
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

        bridge.upgrade(vaa);

        bytes32 afterUpgrade = vm.load(
            address(bridge),
            IMPLEMENTATION_STORAGE_SLOT
        );
        assertEq(afterUpgrade, addressToBytes32(address(mock)));
        assertEq(
            MockNFTBridgeImplementation(payable(address(bridge)))
                .testNewImplementationActive(),
            true,
            "New implementation not active"
        );

        // Overwrite EVM Chain ID
        MockNFTBridgeImplementation(payable(address(bridge)))
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
            NFTBridgeModule,
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
        MockNFTBridgeImplementation mock = new MockNFTBridgeImplementation();

        bytes memory data = abi.encodePacked(
            NFTBridgeModule,
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

        bridge.upgrade(vaa);

        bytes32 afterUpgrade = vm.load(
            address(bridge),
            IMPLEMENTATION_STORAGE_SLOT
        );
        assertEq(afterUpgrade, bytes32(uint256(uint160(address(mock)))));
        assertEq(
            MockNFTBridgeImplementation(payable(address(bridge)))
                .testNewImplementationActive(),
            true,
            "New implementation not active"
        );

        // Overwrite EVM Chain ID
        MockNFTBridgeImplementation(payable(address(bridge)))
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
            NFTBridgeModule,
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
        MockNFTBridgeImplementation mock = new MockNFTBridgeImplementation();

        bytes memory data = abi.encodePacked(
            NFTBridgeModule,
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
            MockNFTBridgeImplementation(payable(address(bridge)))
                .testNewImplementationActive(),
            true,
            "New implementation not active"
        );

        // Overwrite EVM Chain ID
        MockNFTBridgeImplementation(payable(address(bridge)))
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
            NFTBridgeModule,
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
        mock = new MockNFTBridgeImplementation();

        data = abi.encodePacked(
            NFTBridgeModule,
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
            MockNFTBridgeImplementation(payable(address(bridge)))
                .testNewImplementationActive(),
            true,
            "New implementation not active"
        );
    }

    function onERC721Received(
        address,
        address,
        uint256,
        bytes calldata
    ) external pure returns (bytes4) {
        return 0x150b7a02;
    }

    fallback() external payable {}
    receive() external payable {}
}
