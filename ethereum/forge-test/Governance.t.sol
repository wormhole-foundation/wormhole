// test/Messages.sol
// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.0;

import "../contracts/Implementation.sol";
import "../contracts/Setup.sol";
import "../contracts/Wormhole.sol";
import "forge-std/Test.sol";
import "forge-test/rv-helpers/TestUtils.sol";
import "forge-test/rv-helpers/MyImplementation.sol";
import "forge-test/rv-helpers/IMyWormhole.sol";

contract TestGovernance is TestUtils {

    uint16  constant CHAINID = 2;
    uint256 constant EVMCHAINID = 1;
    bytes32 constant MODULE = 0x00000000000000000000000000000000000000000000000000000000436f7265;
    bytes32 constant governanceContract = 0x0000000000000000000000000000000000000000000000000000000000000004;

    bytes32 constant CHAINID_SLOT = bytes32(uint256(0));
    bytes32 constant GUARDIANSETS_SLOT = bytes32(uint256(2));
    bytes32 constant GUARDIANSETINDEX_SLOT = bytes32(uint256(3));
    bytes32 constant IMPLEMENTATION_SLOT = 0x360894a13ba1a3210667c828492db98dca3e2076cc3735a920a3ca505d382bbc;
    bytes32 constant CONSUMED_ACTIONS_SLOT = bytes32(uint256(5));
    bytes32 constant INIT_IMPLEMENTATION_SLOT = bytes32(uint256(6));
    bytes32 constant MESSAGEFEE_SLOT = bytes32(uint256(7));
    bytes32 constant EVMCHAINID_SLOT = bytes32(uint256(8));

    Wormhole proxy;
    Implementation impl;
    Setup setup;
    Setup proxiedSetup;
    IMyWormhole proxied;

    uint256 constant testGuardian = 93941733246223705020089879371323733820373732307041878556247502674739205313440;

    event ContractUpgraded(address indexed oldContract, address indexed newContract);
    
    function setUp() public {
        // Deploy setup
        setup = new Setup();
        // Deploy implementation contract
        impl = new Implementation();
        // Deploy proxy
        proxy = new Wormhole(address(setup), bytes(""));

        address[] memory keys = new address[](1);
        keys[0] = 0xbeFA429d57cD18b7F8A4d91A2da9AB4AF05d0FBe; // vm.addr(testGuardian)

        //proxied setup
        proxiedSetup = Setup(address(proxy));

        vm.chainId(1);
        proxiedSetup.setup({
            implementation: address(impl),
            initialGuardians: keys,
            chainId: CHAINID,
            governanceChainId: 1,
            governanceContract: governanceContract,
            evmChainId: EVMCHAINID
        });

        proxied = IMyWormhole(address(proxy));
    }

    function testSubmitContractUpgrade(
        bytes32 storageSlot,
        uint32 timestamp,
        uint32 nonce,
        uint64 sequence,
        uint8 consistencyLevel)
        public
        unchangedStorage(address(proxied), storageSlot)
    {
        MyImplementation newImpl = new MyImplementation(EVMCHAINID, CHAINID);

        vm.assume(storageSlot != IMPLEMENTATION_SLOT);
        vm.assume(storageSlot != hashedLocation(address(newImpl), INIT_IMPLEMENTATION_SLOT));

        bytes memory payload = payloadSubmitContract(MODULE, CHAINID, address(newImpl));
        (bytes memory _vm, bytes32 hash) = validVm(
            0, timestamp, nonce, 1, governanceContract, sequence, consistencyLevel, payload, testGuardian);

        vm.assume(storageSlot != hashedLocation(hash, CONSUMED_ACTIONS_SLOT));

        proxied.submitContractUpgrade(_vm);

        assertEq(address(newImpl), address(proxied.getImplementation()));
        assertEq(true, proxied.isInitialized(address(newImpl)));
        assertEq(true, proxied.governanceActionIsConsumed(hash));
    }

    function testSubmitContractUpgrade_Emit(
        bytes32 storageSlot,
        uint32 timestamp,
        uint32 nonce,
        uint64 sequence,
        uint8 consistencyLevel)
        public
        unchangedStorage(address(proxied), storageSlot)
    {
        MyImplementation newImpl = new MyImplementation(EVMCHAINID, CHAINID);

        vm.assume(storageSlot != IMPLEMENTATION_SLOT);
        vm.assume(storageSlot != hashedLocation(address(newImpl), INIT_IMPLEMENTATION_SLOT));

        vm.chainId(EVMCHAINID);

        bytes memory payload = payloadSubmitContract(MODULE, CHAINID, address(newImpl));
        (bytes memory _vm, bytes32 hash) = validVm(
            0, timestamp, nonce, 1, governanceContract, sequence, consistencyLevel, payload, testGuardian);

        vm.assume(storageSlot != hashedLocation(hash, CONSUMED_ACTIONS_SLOT));

        vm.expectEmit(true,true,true,true);
        emit ContractUpgraded(address(impl), address(newImpl));

        proxied.submitContractUpgrade(_vm);
    }

    function testInitialize_after_upgrade_revert(bytes32 storageSlot, address alice)
        public
        unchangedStorage(address(proxied), storageSlot)
    {
        Implementation newImpl = new Implementation(); 

        vm.assume(storageSlot != IMPLEMENTATION_SLOT);
        vm.assume(storageSlot != hashedLocation(address(newImpl), INIT_IMPLEMENTATION_SLOT));
        
        vm.chainId(EVMCHAINID);

        bytes memory payload = payloadSubmitContract(MODULE, 2, address(newImpl));
        (bytes memory _vm, bytes32 hash) = validVm(0, 0, 0, 1, governanceContract, 0, 0, payload, testGuardian);

        vm.assume(storageSlot != hashedLocation(hash, CONSUMED_ACTIONS_SLOT));

        proxied.submitContractUpgrade(_vm);

        vm.prank(alice);
        vm.expectRevert("already initialized");
        proxied.initialize();
    }

    function testSubmitContractUpgrade_Revert_InvalidFork(
        bytes32 storageSlot,
        uint64 evmChainId,
        uint32 timestamp,
        uint32 nonce,
        uint64 sequence,
        uint8 consistencyLevel)
        public
        unchangedStorage(address(proxied), storageSlot)
    {
        vm.assume(evmChainId != EVMCHAINID);
        vm.chainId(evmChainId);

        MyImplementation newImpl = new MyImplementation(evmChainId, CHAINID);
        bytes memory payload = payloadSubmitContract(MODULE, CHAINID, address(newImpl));
        (bytes memory _vm, ) = validVm(
            0, timestamp, nonce, 1, governanceContract, sequence, consistencyLevel, payload, testGuardian);

        vm.expectRevert("invalid fork");
        proxied.submitContractUpgrade(_vm);
    }

    function testSubmitContractUpgrade_Revert_InvalidModule(
        bytes32 storageSlot,
        uint32 timestamp,
        uint32 nonce,
        uint64 sequence,
        uint8 consistencyLevel,
        bytes32 module)
        public
        unchangedStorage(address(proxied), storageSlot)
    {
        vm.assume(module != MODULE);
        vm.chainId(EVMCHAINID);

        MyImplementation newImpl = new MyImplementation(EVMCHAINID, CHAINID);
        bytes memory payload = payloadSubmitContract(module, CHAINID, address(newImpl));
        (bytes memory _vm, ) = validVm(
            0, timestamp, nonce, 1, governanceContract, sequence, consistencyLevel, payload, testGuardian);

        vm.expectRevert("Invalid Module");
        proxied.submitContractUpgrade(_vm);
    }

    function testSubmitContractUpgrade_Revert_InvalidChain(
        bytes32 storageSlot,
        uint32 timestamp,
        uint32 nonce,
        uint64 sequence,
        uint8 consistencyLevel,
        uint16 chainId)
        public
        unchangedStorage(address(proxied), storageSlot)
    {
        vm.assume(chainId != CHAINID);
        vm.chainId(EVMCHAINID);

        MyImplementation newImpl = new MyImplementation(EVMCHAINID, CHAINID);
        bytes memory payload = payloadSubmitContract(MODULE, chainId, address(newImpl));
        (bytes memory _vm, ) = validVm(
            0, timestamp, nonce, 1, governanceContract, sequence, consistencyLevel, payload, testGuardian);
        
        vm.expectRevert("Invalid Chain");
        proxied.submitContractUpgrade(_vm);
    }

    function testSubmitContractUpgrade_Revert_InvalidGuardianSetIndex(
        bytes32 storageSlot,
        uint32 guardianSetIndex,
        uint32 timestamp,
        uint32 nonce,
        uint64 sequence,
        uint8 consistencyLevel)
        public
        unchangedStorage(address(proxied), storageSlot)
    {
        vm.assume(guardianSetIndex != 0);
        vm.chainId(EVMCHAINID);

        MyImplementation newImpl = new MyImplementation(EVMCHAINID, CHAINID);
        bytes memory payload = payloadSubmitContract(MODULE, CHAINID, address(newImpl));
        (bytes memory _vm, ) = validVm(
            guardianSetIndex, timestamp, nonce, 1, governanceContract, sequence, consistencyLevel, payload, testGuardian);

        // Since the current version of the test uses only one guardian set,
        // in practice only the 'else' branch will be taken
        if (guardianSetIndex < proxied.getCurrentGuardianSetIndex()) {
            vm.expectRevert("not signed by current guardian set");
        } else {
            vm.expectRevert("invalid guardian set");
        }

        proxied.submitContractUpgrade(_vm);
    }

    function testSubmitContractUpgrade_Revert_WrongGovernanceChain(
        bytes32 storageSlot,
        uint32 timestamp,
        uint32 nonce,
        uint16 emitterChainId,
        uint64 sequence,
        uint8 consistencyLevel)
        public
        unchangedStorage(address(proxied), storageSlot)
    {
        vm.assume(emitterChainId != 1);
        vm.chainId(EVMCHAINID);

        MyImplementation newImpl = new MyImplementation(EVMCHAINID, CHAINID);
        bytes memory payload = payloadSubmitContract(MODULE, CHAINID, address(newImpl));
        (bytes memory _vm, ) = validVm(
            0, timestamp, nonce, emitterChainId, governanceContract, sequence, consistencyLevel, payload, testGuardian);

        vm.expectRevert("wrong governance chain");
        proxied.submitContractUpgrade(_vm);
    }

    function testSubmitContractUpgrade_Revert_WrongGovernanceContract(
        bytes32 storageSlot,
        uint32 timestamp,
        uint32 nonce,
        bytes32 emitterAddress,
        uint64 sequence,
        uint8 consistencyLevel)
        public
        unchangedStorage(address(proxied), storageSlot)
    {
        vm.assume(emitterAddress != governanceContract);
        vm.chainId(EVMCHAINID);

        MyImplementation newImpl = new MyImplementation(EVMCHAINID, CHAINID);
        bytes memory payload = payloadSubmitContract(MODULE, CHAINID, address(newImpl));
        (bytes memory _vm, ) = validVm(
            0, timestamp, nonce, 1, emitterAddress, sequence, consistencyLevel, payload, testGuardian);

        vm.expectRevert("wrong governance contract");
        proxied.submitContractUpgrade(_vm);
    }

    function testSubmitContractUpgrade_Revert_ReplayAttack(
        bytes32 storageSlot,
        uint32 timestamp,
        uint32 nonce,
        uint64 sequence,
        uint8 consistencyLevel)
        public
        unchangedStorage(address(proxied), storageSlot)
    {
        MyImplementation newImpl = new MyImplementation(EVMCHAINID, CHAINID);

        vm.assume(storageSlot != IMPLEMENTATION_SLOT);
        vm.assume(storageSlot != hashedLocation(address(newImpl), INIT_IMPLEMENTATION_SLOT));

        vm.chainId(EVMCHAINID);

        bytes memory payload = payloadSubmitContract(MODULE, CHAINID, address(newImpl));
        (bytes memory _vm, bytes32 hash) = validVm(
            0, timestamp, nonce, 1, governanceContract, sequence, consistencyLevel, payload, testGuardian);

        vm.assume(storageSlot != hashedLocation(hash, CONSUMED_ACTIONS_SLOT));

        proxied.submitContractUpgrade(_vm);

        vm.expectRevert("governance action already consumed");
        proxied.submitContractUpgrade(_vm);
    }

    function testSubmitSetMessageFee(
        bytes32 storageSlot,
        uint32 timestamp,
        uint32 nonce,
        uint64 sequence,
        uint8 consistencyLevel,
        uint256 newMessageFee)
        public
        unchangedStorage(address(proxied), storageSlot)
    {
        vm.assume(storageSlot != MESSAGEFEE_SLOT);
        vm.chainId(EVMCHAINID);

        bytes memory payload = payloadSubmitMessageFee(MODULE, CHAINID, newMessageFee);
        (bytes memory _vm, bytes32 hash) = validVm(
            0, timestamp, nonce, 1, governanceContract, sequence, consistencyLevel, payload, testGuardian);

        vm.assume(storageSlot != hashedLocation(hash, CONSUMED_ACTIONS_SLOT));

        proxied.submitSetMessageFee(_vm);

        assertEq(newMessageFee, proxied.messageFee());
        assertEq(true, proxied.governanceActionIsConsumed(hash));
    }

    function testSubmitSetMessageFee_Revert_InvalidModule(
        bytes32 storageSlot,
        uint32 timestamp,
        uint32 nonce,
        uint64 sequence,
        uint8 consistencyLevel,
        bytes32 module,
        uint256 newMessageFee)
        public
        unchangedStorage(address(proxied), storageSlot)
    {
        vm.assume(module != MODULE);
        vm.chainId(EVMCHAINID);

        bytes memory payload = payloadSubmitMessageFee(module, CHAINID, newMessageFee);
        (bytes memory _vm, ) = validVm(
            0, timestamp, nonce, 1, governanceContract, sequence, consistencyLevel, payload, testGuardian);

        vm.expectRevert("Invalid Module");
        proxied.submitSetMessageFee(_vm);
    }

    function testSubmitSetMessageFee_Revert_InvalidChain(
        bytes32 storageSlot,
        uint32 timestamp,
        uint32 nonce,
        uint64 sequence,
        uint8 consistencyLevel,
        uint16 chainId,
        uint256 newMessageFee)
        public
        unchangedStorage(address(proxied), storageSlot)
    {
        vm.assume(chainId != CHAINID);
        vm.chainId(EVMCHAINID);

        bytes memory payload = payloadSubmitMessageFee(MODULE, chainId, newMessageFee);
        (bytes memory _vm, ) = validVm(
            0, timestamp, nonce, 1, governanceContract, sequence, consistencyLevel, payload, testGuardian);

        vm.expectRevert("Invalid Chain");
        proxied.submitSetMessageFee(_vm);
    }

    function testSubmitSetMessageFee_Revert_InvalidEvmChain(
        bytes32 storageSlot,
        uint64 evmChainId,
        uint32 timestamp,
        uint32 nonce,
        uint64 sequence,
        uint8 consistencyLevel,
        uint256 newMessageFee)
        public
        unchangedStorage(address(proxied), storageSlot)
    {
        vm.assume(evmChainId != EVMCHAINID);
        vm.chainId(evmChainId);

        bytes memory payload = payloadSubmitMessageFee(MODULE, CHAINID, newMessageFee);
        (bytes memory _vm, ) = validVm(
            0, timestamp, nonce, 1, governanceContract, sequence, consistencyLevel, payload, testGuardian);

        vm.expectRevert("Invalid Chain");
        proxied.submitSetMessageFee(_vm);
    }

    function testSubmitSetMessageFee_Revert_InvalidGuardianSetIndex(
        bytes32 storageSlot,
        uint32 guardianSetIndex,
        uint32 timestamp,
        uint32 nonce,
        uint64 sequence,
        uint8 consistencyLevel,
        uint256 newMessageFee)
        public
        unchangedStorage(address(proxied), storageSlot)
    {
        vm.assume(guardianSetIndex != 0);
        vm.chainId(EVMCHAINID);

        bytes memory payload = payloadSubmitMessageFee(MODULE, CHAINID, newMessageFee);
        (bytes memory _vm, ) = validVm(
            guardianSetIndex, timestamp, nonce, 1, governanceContract, sequence, consistencyLevel, payload, testGuardian);

        // Since the current version of the test uses only one guardian set,
        // in practice only the 'else' branch will be taken
        if (guardianSetIndex < proxied.getCurrentGuardianSetIndex()) {
            vm.expectRevert("not signed by current guardian set");
        } else {
            vm.expectRevert("invalid guardian set");
        }

        proxied.submitSetMessageFee(_vm);
    }

    function testSubmitSetMessageFee_Revert_WrongGovernanceChain(
        bytes32 storageSlot,
        uint32 timestamp,
        uint32 nonce,
        uint16 emitterChainId,
        uint64 sequence,
        uint8 consistencyLevel,
        uint256 newMessageFee)
        public
        unchangedStorage(address(proxied), storageSlot)
    {
        vm.assume(emitterChainId != 1);
        vm.chainId(EVMCHAINID);

        bytes memory payload = payloadSubmitMessageFee(MODULE, CHAINID, newMessageFee);
        (bytes memory _vm, ) = validVm(
            0, timestamp, nonce, emitterChainId, governanceContract, sequence, consistencyLevel, payload, testGuardian);

        vm.expectRevert("wrong governance chain");
        proxied.submitSetMessageFee(_vm);
    }

    function testSubmitSetMessageFee_Revert_WrongGovernanceContract(
        bytes32 storageSlot,
        uint32 timestamp,
        uint32 nonce,
        bytes32 emitterAddress,
        uint64 sequence,
        uint8 consistencyLevel,
        uint256 newMessageFee)
        public
        unchangedStorage(address(proxied), storageSlot)
    {
        vm.assume(emitterAddress != governanceContract);
        vm.chainId(EVMCHAINID);

        bytes memory payload = payloadSubmitMessageFee(MODULE, CHAINID, newMessageFee);
        (bytes memory _vm, ) = validVm(
            0, timestamp, nonce, 1, emitterAddress, sequence, consistencyLevel, payload, testGuardian);

        vm.expectRevert("wrong governance contract");
        proxied.submitSetMessageFee(_vm);
    }

    function testSubmitSetMessageFee_Revert_ReplayAttack(
        bytes32 storageSlot,
        uint32 timestamp,
        uint32 nonce,
        uint64 sequence,
        uint8 consistencyLevel,
        uint256 newMessageFee)
        public
        unchangedStorage(address(proxied), storageSlot)
    {
        vm.assume(storageSlot != MESSAGEFEE_SLOT);
        vm.chainId(EVMCHAINID);

        bytes memory payload = payloadSubmitMessageFee(MODULE, CHAINID, newMessageFee);
        (bytes memory _vm, bytes32 hash) = validVm(
            0, timestamp, nonce, 1, governanceContract, sequence, consistencyLevel, payload, testGuardian);

        vm.assume(storageSlot != hashedLocation(hash, CONSUMED_ACTIONS_SLOT));

        proxied.submitSetMessageFee(_vm);

        vm.expectRevert("governance action already consumed");
        proxied.submitSetMessageFee(_vm);
    }

    //Make a similar test but with chainId = 0
    function testSubmitNewGuardianSet(
        bytes32 storageSlot,
        uint32 timestamp,
        uint32 nonce,
        uint64 sequence,
        uint8 consistencyLevel,
        address[] memory newGuardianSet)
        public
        unchangedStorage(address(proxied), storageSlot)
    {
        vm.assume(storageSlot != hashedLocationOffset(0, GUARDIANSETS_SLOT, 1));

        // New GuardianSet array length should be initialized from zero to non-zero
        vm.assume(storageSlot != hashedLocationOffset(1, GUARDIANSETS_SLOT, 0));

        vm.assume(storageSlot != GUARDIANSETINDEX_SLOT);
        vm.assume(0 < newGuardianSet.length);
        vm.assume(newGuardianSet.length < 20);

        for(uint8 i = 0; i < newGuardianSet.length; i++) {
            vm.assume(newGuardianSet[i] != address(0));
            // New GuardianSet key array elements should be initialized from zero to non-zero
            vm.assume(storageSlot != arrayElementLocation(hashedLocationOffset(1, GUARDIANSETS_SLOT, 0), i));
        }

        vm.chainId(EVMCHAINID);

        bytes memory payload = payloadSubmitNewGuardianSet(MODULE, CHAINID, 1, newGuardianSet);
        (bytes memory _vm, bytes32 hash) = validVm(
            0, timestamp, nonce, 1, governanceContract, sequence, consistencyLevel, payload, testGuardian);

        vm.assume(storageSlot != hashedLocation(hash, CONSUMED_ACTIONS_SLOT));

        proxied.submitNewGuardianSet(_vm);

        assertEq(true, proxied.governanceActionIsConsumed(hash));
        assertEq(uint32(block.timestamp) + 86400, proxied.getGuardianSet(0).expirationTime);
        assertEq(newGuardianSet, proxied.getGuardianSet(1).keys);
        assertEq(1, proxied.getCurrentGuardianSetIndex());
    }

    function testSubmitNewGuardianSet_Revert_InvalidModule(
        bytes32 storageSlot,
        uint64 evmChainId,
        uint32 timestamp,
        uint32 nonce,
        uint64 sequence,
        uint8 consistencyLevel,
        bytes32 module,
        uint16 chainId,
        address[] memory newGuardianSet)
        public
        unchangedStorage(address(proxied), storageSlot)
    {
        vm.assume(module != MODULE);
        vm.assume(chainId == 0 || (chainId == CHAINID && evmChainId == EVMCHAINID));
        vm.assume(newGuardianSet.length < 20);
        vm.chainId(evmChainId);

        bytes memory payload = payloadSubmitNewGuardianSet(module,chainId, 1, newGuardianSet);
        (bytes memory _vm, ) = validVm(
            0, timestamp, nonce, 1, governanceContract, sequence, consistencyLevel, payload, testGuardian);

        vm.expectRevert("invalid Module");
        proxied.submitNewGuardianSet(_vm);
    }

    function testSubmitNewGuardianSet_Revert_InvalidChain(
        bytes32 storageSlot,
        uint32 timestamp,
        uint32 nonce,
        uint64 sequence,
        uint8 consistencyLevel,
        uint16 chainId,
        address[] memory newGuardianSet)
        public
        unchangedStorage(address(proxied), storageSlot)
    {
        vm.assume(chainId != CHAINID && chainId != 0);
        vm.assume(newGuardianSet.length < 20);
        vm.chainId(EVMCHAINID);

        bytes memory payload = payloadSubmitNewGuardianSet(MODULE, chainId, 1, newGuardianSet);
        (bytes memory _vm, ) = validVm(
            0, timestamp, nonce, 1, governanceContract, sequence, consistencyLevel, payload, testGuardian);

        vm.expectRevert("invalid Chain");
        proxied.submitNewGuardianSet(_vm);
    }

    function testSubmitNewGuardianSet_Revert_InvalidEvmChain(
        bytes32 storageSlot,
        uint64 evmChainId,
        uint32 timestamp,
        uint32 nonce,
        uint64 sequence,
        uint8 consistencyLevel,
        address[] memory newGuardianSet)
        public
        unchangedStorage(address(proxied), storageSlot)
    {
        vm.assume(evmChainId != EVMCHAINID);
        vm.assume(newGuardianSet.length < 20);
        vm.chainId(evmChainId);

        bytes memory payload = payloadSubmitNewGuardianSet(MODULE, CHAINID, 1, newGuardianSet);
        (bytes memory _vm, ) = validVm(
            0, timestamp, nonce, 1, governanceContract, sequence, consistencyLevel, payload, testGuardian);

        vm.expectRevert("invalid Chain");
        proxied.submitNewGuardianSet(_vm);
    }

    function testSubmitNewGuardianSet_Revert_GuardianSetEmpty(
        bytes32 storageSlot,
        uint64 evmChainId,
        uint32 timestamp,
        uint32 nonce,
        uint64 sequence,
        uint8 consistencyLevel,
        uint16 chainId)
        public
        unchangedStorage(address(proxied), storageSlot)
    {
        vm.assume(chainId == 0 || (chainId == CHAINID && evmChainId == EVMCHAINID));
        vm.chainId(evmChainId);

        address[] memory newGuardianSet = new address[](0); // Empty guardian set

        bytes memory payload = payloadSubmitNewGuardianSet(MODULE, chainId, 1, newGuardianSet);
        (bytes memory _vm, ) = validVm(
            0, timestamp, nonce, 1, governanceContract, sequence, consistencyLevel, payload, testGuardian);

        vm.expectRevert("new guardian set is empty");
        proxied.submitNewGuardianSet(_vm);
    }

    function testSubmitNewGuardianSet_Revert_WrongIndex(
        bytes32 storageSlot,
        uint64 evmChainId,
        uint32 timestamp,
        uint32 nonce,
        uint64 sequence,
        uint8 consistencyLevel,
        uint16 chainId,
        uint32 newGuardianSetIndex,
        address[] memory newGuardianSet)
        public
        unchangedStorage(address(proxied), storageSlot)
    {
        vm.assume(newGuardianSetIndex != 1);
        vm.assume(chainId == 0 || (chainId == CHAINID && evmChainId == EVMCHAINID));
        vm.assume(0 < newGuardianSet.length);
        vm.assume(newGuardianSet.length < 20);
        vm.chainId(evmChainId);

        bytes memory payload = payloadSubmitNewGuardianSet(MODULE, chainId, newGuardianSetIndex, newGuardianSet);
        (bytes memory _vm, ) = validVm(
            0, timestamp, nonce, 1, governanceContract, sequence, consistencyLevel, payload, testGuardian);

        vm.expectRevert("index must increase in steps of 1");
        proxied.submitNewGuardianSet(_vm);
    }

    function testSubmitNewGuardianSet_Revert_InvalidGuardianSetIndex(
        bytes32 storageSlot,
        uint32 guardianSetIndex,
        uint64 evmChainId,
        uint32 timestamp,
        uint32 nonce,
        uint64 sequence,
        uint8 consistencyLevel,
        uint16 chainId,
        address[] memory newGuardianSet)
        public
        unchangedStorage(address(proxied), storageSlot)
    {
        vm.assume(guardianSetIndex != 0);
        vm.assume(chainId == 0 || (chainId == CHAINID && evmChainId == EVMCHAINID));
        vm.assume(0 < newGuardianSet.length);
        vm.assume(newGuardianSet.length < 20);
        vm.chainId(evmChainId);

        bytes memory payload = payloadSubmitNewGuardianSet(MODULE, chainId, 1, newGuardianSet);
        (bytes memory _vm, ) = validVm(
            guardianSetIndex, timestamp, nonce, 1, governanceContract, sequence, consistencyLevel, payload, testGuardian);

        // Since the current version of the test uses only one guardian set,
        // in practice only the 'else' branch will be taken
        if (guardianSetIndex < proxied.getCurrentGuardianSetIndex()) {
            vm.expectRevert("not signed by current guardian set");
        } else {
            vm.expectRevert("invalid guardian set");
        }

        proxied.submitNewGuardianSet(_vm);
    }

    function testSubmitNewGuardianSet_Revert_WrongGovernanceChain(
        bytes32 storageSlot,
        uint64 evmChainId,
        uint32 timestamp,
        uint32 nonce,
        uint16 emitterChainId,
        uint64 sequence,
        uint8 consistencyLevel,
        uint16 chainId,
        address[] memory newGuardianSet)
        public
        unchangedStorage(address(proxied), storageSlot)
    {
        vm.assume(emitterChainId != 1);
        vm.assume(chainId == 0 || (chainId == CHAINID && evmChainId == EVMCHAINID));
        vm.assume(0 < newGuardianSet.length);
        vm.assume(newGuardianSet.length < 20);
        vm.chainId(evmChainId);

        bytes memory payload = payloadSubmitNewGuardianSet(MODULE, chainId, 1, newGuardianSet);
        (bytes memory _vm, ) = validVm(
            0, timestamp, nonce, emitterChainId, governanceContract, sequence, consistencyLevel, payload, testGuardian);

        vm.expectRevert("wrong governance chain");
        proxied.submitNewGuardianSet(_vm);
    }

    function testSubmitNewGuardianSet_Revert_WrongGovernanceContract(
        bytes32 storageSlot,
        uint64 evmChainId,
        uint32 timestamp,
        uint32 nonce,
        bytes32 emitterAddress,
        uint64 sequence,
        uint8 consistencyLevel,
        uint16 chainId,
        address[] memory newGuardianSet)
        public
        unchangedStorage(address(proxied), storageSlot)
    {
        vm.assume(emitterAddress != governanceContract);
        vm.assume(chainId == 0 || (chainId == CHAINID && evmChainId == EVMCHAINID));
        vm.assume(0 < newGuardianSet.length);
        vm.assume(newGuardianSet.length < 20);
        vm.chainId(evmChainId);

        bytes memory payload = payloadSubmitNewGuardianSet(MODULE, chainId, 1, newGuardianSet);
        (bytes memory _vm, ) = validVm(
            0, timestamp, nonce, 1, emitterAddress, sequence, consistencyLevel, payload, testGuardian);

        vm.expectRevert("wrong governance contract");
        proxied.submitNewGuardianSet(_vm);
    }

    function testSubmitNewGuardianSet_Revert_ReplayAttack(
        bytes32 storageSlot,
        uint32 timestamp,
        uint32 nonce,
        uint64 sequence,
        uint8 consistencyLevel,
        address[] memory newGuardianSet)
        public
        unchangedStorage(address(proxied), storageSlot)
    {
        vm.assume(storageSlot != hashedLocationOffset(0, GUARDIANSETS_SLOT, 1));

        // New GuardianSet array length should be initialized from zero to non-zero
        vm.assume(storageSlot != hashedLocationOffset(1, GUARDIANSETS_SLOT, 0));

        vm.assume(storageSlot != GUARDIANSETINDEX_SLOT);
        vm.assume(0 < newGuardianSet.length);
        vm.assume(newGuardianSet.length < 20);

        for(uint8 i = 0; i < newGuardianSet.length; i++) {
            vm.assume(newGuardianSet[i] != address(0));
            // New GuardianSet key array elements should be initialized from zero to non-zero
            vm.assume(storageSlot != arrayElementLocation(hashedLocationOffset(1, GUARDIANSETS_SLOT, 0), i));
        }

        vm.chainId(EVMCHAINID);

        bytes memory payload = payloadSubmitNewGuardianSet(MODULE, CHAINID, 1, newGuardianSet);
        (bytes memory _vm, bytes32 hash) = validVm(
            0, timestamp, nonce, 1, governanceContract, sequence, consistencyLevel, payload, testGuardian);

        vm.assume(storageSlot != hashedLocation(hash, CONSUMED_ACTIONS_SLOT));

        proxied.submitNewGuardianSet(_vm);

        // The error message is not "governance action already consumed" because the guardian set index is updated,
        // and the check for the current guardian set index comes first than the check for action already consumed
        vm.expectRevert("not signed by current guardian set");
        proxied.submitNewGuardianSet(_vm);
    }

    function isReservedAddress(address addr) internal view returns (bool) {
        return
            // Avoid precompiled contracts
            addr <= address(0x9) ||
            // Wormhole implementation contract does not accept assets
            addr == address(impl) ||
            // Wormhole proxy contract does not accept assets
            addr == address(proxied) ||
			// Setup contract
			addr == address(setup) ||
			// Test contract
			addr == address(this) ||
            // Cheatcode contract
            addr == address(vm) ||
            // Create2Deployer address
            addr == address(0x4e59b44847b379578588920cA78FbF26c0B4956C);
    }

    function testSubmitTransferFees(
        bytes32 storageSlot,
        uint32 timestamp,
        uint32 nonce,
        uint64 sequence,
        uint8 consistencyLevel,
        uint256 amount,
        bytes32 recipient)
        public
        unchangedStorage(address(proxied), storageSlot)
    {
        // Avoid reserved addresses (which will cause the transfer to revert)
        vm.assume(!isReservedAddress(address(uint160(uint256(recipient)))));

        vm.chainId(EVMCHAINID);
        vm.deal(address(proxied), amount);

        bytes memory payload = payloadSubmitTransferFees(MODULE, CHAINID, amount, recipient);
        (bytes memory _vm, bytes32 hash) = validVm(
            0, timestamp, nonce, 1, governanceContract, sequence, consistencyLevel, payload, testGuardian);

        vm.assume(storageSlot != hashedLocation(hash, CONSUMED_ACTIONS_SLOT));

        address payable receiver = payable(address(uint160(uint256(recipient))));
        uint256 previousBalance = receiver.balance;

        proxied.submitTransferFees(_vm);

        assertEq(receiver.balance, previousBalance + amount);
        assertEq(address(proxied).balance, 0);
        assertEq(true, proxied.governanceActionIsConsumed(hash));
    }

    function testSubmitTransferFees_Revert_InvalidModule(
        bytes32 storageSlot,
        uint64 evmChainId,
        uint32 timestamp,
        uint32 nonce,
        uint64 sequence,
        uint8 consistencyLevel,
        bytes32 module,
        uint16 chainId,
        uint256 amount,
        bytes32 recipient)
        public
        unchangedStorage(address(proxied), storageSlot)
    {
        vm.assume(module != MODULE);
        vm.assume(chainId == 0 || (chainId == CHAINID && evmChainId == EVMCHAINID));
        vm.chainId(evmChainId);
        vm.deal(address(proxied), amount);

        bytes memory payload = payloadSubmitTransferFees(module, CHAINID, amount, recipient);
        bytes memory body = abi.encodePacked(
                timestamp, nonce, uint16(1), governanceContract, sequence, consistencyLevel, payload);
        
        bytes32 hash = keccak256(abi.encodePacked(keccak256(body)));

        bytes memory _vm = bytes.concat(validVmHeader(0), validSignature(testGuardian, hash), body);

        vm.expectRevert("invalid Module");
        proxied.submitTransferFees(_vm);
    }

    function testSubmitTransferFees_Revert_InvalidChain(
        bytes32 storageSlot,
        uint32 timestamp,
        uint32 nonce,
        uint64 sequence,
        uint8 consistencyLevel,
        uint16 chainId,
        uint256 amount,
        bytes32 recipient)
        public
        unchangedStorage(address(proxied), storageSlot)
    {
        vm.assume(chainId != CHAINID && chainId != 0);
        vm.chainId(EVMCHAINID);
        vm.deal(address(proxied), amount);

        bytes memory payload = payloadSubmitTransferFees(MODULE, chainId, amount, recipient);
        (bytes memory _vm, ) = validVm(
            0, timestamp, nonce, 1, governanceContract, sequence, consistencyLevel, payload, testGuardian);

        vm.expectRevert("invalid Chain");
        proxied.submitTransferFees(_vm);
    }

    function testSubmitTransferFees_Revert_InvalidEvmChain(
        bytes32 storageSlot,
        uint64 evmchainId,
        uint32 timestamp,
        uint32 nonce,
        uint64 sequence,
        uint8 consistencyLevel,
        uint256 amount,
        bytes32 recipient)
        public
        unchangedStorage(address(proxied), storageSlot)
    {
        vm.assume(evmchainId != EVMCHAINID);
        vm.chainId(evmchainId);
        vm.deal(address(proxied), amount);

        bytes memory payload = payloadSubmitTransferFees(MODULE, CHAINID, amount, recipient);
        (bytes memory _vm, ) = validVm(
            0, timestamp, nonce, 1, governanceContract, sequence, consistencyLevel, payload, testGuardian);

        vm.expectRevert("invalid Chain");
        proxied.submitTransferFees(_vm);
    }


    function testSubmitTransferFees_Revert_InvalidGuardianSet(
        bytes32 storageSlot,
        uint64 evmChainId,
        uint32 guardianSetIndex,
        uint32 timestamp,
        uint32 nonce,
        uint64 sequence,
        uint8 consistencyLevel,
        uint16 chainId,
        uint256 amount,
        bytes32 recipient)
        public
        unchangedStorage(address(proxied), storageSlot)
    {
        vm.assume(guardianSetIndex != 0);
        vm.assume(chainId == 0 || (chainId == CHAINID && evmChainId == EVMCHAINID));
        vm.chainId(evmChainId);
        vm.deal(address(proxied), amount);

        bytes memory payload = payloadSubmitTransferFees(MODULE, chainId, amount, recipient);
        (bytes memory _vm, ) = validVm(
            guardianSetIndex, timestamp, nonce, 1, governanceContract, sequence, consistencyLevel, payload, testGuardian);

        // Since the current version of the test uses only one guardian set,
        // in practice only the 'else' branch will be taken
        if (guardianSetIndex < proxied.getCurrentGuardianSetIndex()) {
            vm.expectRevert("not signed by current guardian set");
        } else {
            vm.expectRevert("invalid guardian set");
        }

        proxied.submitTransferFees(_vm);
    }

    function testSubmitTransferFees_Revert_WrongGovernanceChain(
        bytes32 storageSlot,
        uint64 evmChainId,
        uint32 timestamp,
        uint32 nonce,
        uint16 emitterChainId,
        uint64 sequence,
        uint8 consistencyLevel,
        uint16 chainId,
        uint256 amount,
        bytes32 recipient)
        public
        unchangedStorage(address(proxied), storageSlot)
    {
        vm.assume(emitterChainId != 1);
        vm.assume(chainId == 0 || (chainId == CHAINID && evmChainId == EVMCHAINID));
        vm.chainId(evmChainId);
        vm.deal(address(proxied), amount);

        bytes memory payload = payloadSubmitTransferFees(MODULE, chainId, amount, recipient);
        (bytes memory _vm, ) = validVm(
            0, timestamp, nonce, emitterChainId, governanceContract, sequence, consistencyLevel, payload, testGuardian);

        vm.expectRevert("wrong governance chain");
        proxied.submitTransferFees(_vm);
    }

    function testSubmitTransferFees_Revert_WrongGovernanceContract(
        bytes32 storageSlot,
        uint64 evmChainId,
        uint32 timestamp,
        uint32 nonce,
        bytes32 emitterAddress,
        uint64 sequence,
        uint8 consistencyLevel,
        uint16 chainId,
        uint256 amount,
        bytes32 recipient)
        public
        unchangedStorage(address(proxied), storageSlot)
    {
        vm.assume(emitterAddress != governanceContract);
        vm.assume(chainId == 0 || (chainId == CHAINID && evmChainId == EVMCHAINID));
        vm.chainId(evmChainId);
        vm.deal(address(proxied), amount);

        bytes memory payload = payloadSubmitTransferFees(MODULE, chainId, amount, recipient);
        (bytes memory _vm, ) = validVm(
            0, timestamp, nonce, 1, emitterAddress, sequence, consistencyLevel, payload, testGuardian);

        vm.expectRevert("wrong governance contract");
        proxied.submitTransferFees(_vm);
    }

    function testSubmitTransferFees_Revert_ReplayAttack(
        bytes32 storageSlot,
        uint32 timestamp,
        uint32 nonce,
        uint64 sequence,
        uint8 consistencyLevel,
        uint256 amount,
        bytes32 recipient)
        public
        unchangedStorage(address(proxied), storageSlot)
    {
        // Avoid reserved addresses (which will cause the transfer to revert)
        vm.assume(!isReservedAddress(address(uint160(uint256(recipient)))));

        vm.chainId(EVMCHAINID);
        vm.deal(address(proxied), amount);

        bytes memory payload = payloadSubmitTransferFees(MODULE, CHAINID, amount, recipient);
        (bytes memory _vm, bytes32 hash) = validVm(
            0, timestamp, nonce, 1, governanceContract, sequence, consistencyLevel, payload, testGuardian);

        vm.assume(storageSlot != hashedLocation(hash, CONSUMED_ACTIONS_SLOT));

        proxied.submitTransferFees(_vm);

        vm.expectRevert("governance action already consumed");
        proxied.submitTransferFees(_vm);
    }

    function testSubmitRecoverChainId(
        bytes32 storageSlot,
        uint32 timestamp,
        uint32 nonce,
        uint64 sequence,
        uint8 consistencyLevel,
        uint256 evmChainId,
        uint16 newChainId)
        public
        unchangedStorage(address(proxied), storageSlot)
    {
        vm.assume(storageSlot != CHAINID_SLOT);
        vm.assume(storageSlot != EVMCHAINID_SLOT);
        vm.assume(evmChainId != EVMCHAINID);
        vm.assume(evmChainId < 2 ** 64);
        vm.chainId(evmChainId);

        bytes memory payload = payloadSubmitRecoverChainId(MODULE, evmChainId, newChainId);
        (bytes memory _vm, bytes32 hash) = validVm(
            0, timestamp, nonce, 1, governanceContract, sequence, consistencyLevel, payload, testGuardian);

        vm.assume(storageSlot != hashedLocation(hash, CONSUMED_ACTIONS_SLOT));

        proxied.submitRecoverChainId(_vm);

        assertEq(true, proxied.governanceActionIsConsumed(hash));
        assertEq(evmChainId, proxied.evmChainId());
        assertEq(newChainId, proxied.chainId());
    }

    function testSubmitRecoverChainId_Revert_NotAFork(
        bytes32 storageSlot,
        uint32 timestamp,
        uint32 nonce,
        uint64 sequence,
        uint8 consistencyLevel,
        uint256 evmChainId,
        uint16 newChainId)
        public
        unchangedStorage(address(proxied), storageSlot)
    {
        vm.assume(evmChainId != EVMCHAINID);
        vm.chainId(EVMCHAINID);

        bytes memory payload = payloadSubmitRecoverChainId(MODULE, evmChainId, newChainId);
        (bytes memory _vm, ) = validVm(
            0, timestamp, nonce, 1, governanceContract, sequence, consistencyLevel, payload, testGuardian);

        vm.expectRevert("not a fork");
        proxied.submitRecoverChainId(_vm);
    }

    function testSubmitRecoverChainId_Revert_InvalidModule(
        bytes32 storageSlot,
        uint32 timestamp,
        uint32 nonce,
        uint64 sequence,
        uint8 consistencyLevel,
        bytes32 module,
        uint256 evmChainId,
        uint16 newChainId)
        public
        unchangedStorage(address(proxied), storageSlot)
    {
        vm.assume(module != MODULE);
        vm.assume(evmChainId != EVMCHAINID);
        vm.assume(evmChainId < 2 ** 64);
        vm.chainId(evmChainId);

        vm.assume(module != MODULE);
        bytes memory payload = payloadSubmitRecoverChainId(module, evmChainId, newChainId);
        (bytes memory _vm, ) = validVm(
            0, timestamp, nonce, 1, governanceContract, sequence, consistencyLevel, payload, testGuardian);

        vm.expectRevert("invalid Module");
        proxied.submitRecoverChainId(_vm);
    }

    function testSubmitRecoverChainId_Revert_InvalidEVMChain(
        bytes32 storageSlot,
        uint64 blockChainId,
        uint32 timestamp,
        uint32 nonce,
        uint64 sequence,
        uint8 consistencyLevel,
        uint256 evmChainId,
        uint16 newChainId)
        public
        unchangedStorage(address(proxied), storageSlot)
    {
        vm.assume(evmChainId != EVMCHAINID);
        vm.assume(evmChainId < 2 ** 64);
        vm.assume(blockChainId != evmChainId && blockChainId != EVMCHAINID);
        vm.chainId(blockChainId);

        bytes memory payload = payloadSubmitRecoverChainId(MODULE, evmChainId, newChainId);
        (bytes memory _vm, ) = validVm(
            0, timestamp, nonce, 1, governanceContract, sequence, consistencyLevel, payload, testGuardian);

        vm.expectRevert("invalid EVM Chain");
        proxied.submitRecoverChainId(_vm);
    }


    function testSubmitRecoverChainId_Revert_InvalidGuardianSetIndex(
        bytes32 storageSlot,
        uint32 guardianSetIndex,
        uint32 timestamp,
        uint32 nonce,
        uint64 sequence,
        uint8 consistencyLevel,
        uint256 evmChainId,
        uint16 newChainId)
        public
        unchangedStorage(address(proxied), storageSlot)
    {
        vm.assume(guardianSetIndex != 0);
        vm.assume(evmChainId != EVMCHAINID);
        vm.assume(evmChainId < 2 ** 64);
        vm.chainId(evmChainId);

        bytes memory payload = payloadSubmitRecoverChainId(MODULE, evmChainId, newChainId);
        (bytes memory _vm, ) = validVm(
            guardianSetIndex, timestamp, nonce, 1, governanceContract, sequence, consistencyLevel, payload, testGuardian);

        // Since the current version of the test uses only one guardian set,
        // in practice only the 'else' branch will be taken
        if (guardianSetIndex < proxied.getCurrentGuardianSetIndex()) {
            vm.expectRevert("not signed by current guardian set");
        } else {
            vm.expectRevert("invalid guardian set");
        }

        proxied.submitRecoverChainId(_vm);
    }

    function testSubmitRecoverChainId_Revert_WrongGovernanceChain(
        bytes32 storageSlot,
        uint32 timestamp,
        uint32 nonce,
        uint16 emitterChainId,
        uint64 sequence,
        uint8 consistencyLevel,
        uint256 evmChainId,
        uint16 newChainId)
        public
        unchangedStorage(address(proxied), storageSlot)
    {
        vm.assume(emitterChainId != 1);
        vm.assume(evmChainId != EVMCHAINID);
        vm.assume(evmChainId < 2 ** 64);
        vm.chainId(evmChainId);

        bytes memory payload = payloadSubmitRecoverChainId(MODULE, evmChainId, newChainId);
        (bytes memory _vm, ) = validVm(
            0, timestamp, nonce, emitterChainId, governanceContract, sequence, consistencyLevel, payload, testGuardian);

        vm.expectRevert("wrong governance chain");
        proxied.submitRecoverChainId(_vm);
    }

    function testSubmitRecoverChainId_Revert_WrongGovernanceContract(
        bytes32 storageSlot,
        uint32 timestamp,
        uint32 nonce,
        bytes32 emitterAddress,
        uint64 sequence,
        uint8 consistencyLevel,
        uint256 evmChainId,
        uint16 newChainId)
        public
        unchangedStorage(address(proxied), storageSlot)
    {
        vm.assume(emitterAddress != governanceContract);
        vm.assume(evmChainId != EVMCHAINID);
        vm.assume(evmChainId < 2 ** 64);
        vm.chainId(evmChainId);

        bytes memory payload = payloadSubmitRecoverChainId(MODULE, evmChainId, newChainId);
        (bytes memory _vm, ) = validVm(
            0, timestamp, nonce, 1, emitterAddress, sequence, consistencyLevel, payload, testGuardian);

        vm.expectRevert("wrong governance contract");
        proxied.submitRecoverChainId(_vm);
    }

    function testSubmitRecoverChainId_Revert_ReplayAttack(
        bytes32 storageSlot,
        uint32 timestamp,
        uint32 nonce,
        uint64 sequence,
        uint8 consistencyLevel,
        uint256 evmChainId,
        uint16 newChainId)
        public
        unchangedStorage(address(proxied), storageSlot)
    {
        vm.assume(storageSlot != CHAINID_SLOT);
        vm.assume(storageSlot != EVMCHAINID_SLOT);
        vm.assume(evmChainId != EVMCHAINID);
        vm.assume(evmChainId < 2 ** 64);
        vm.chainId(evmChainId);

        bytes memory payload = payloadSubmitRecoverChainId(MODULE, evmChainId, newChainId);
        (bytes memory _vm, bytes32 hash) = validVm(
            0, timestamp, nonce, 1, governanceContract, sequence, consistencyLevel, payload, testGuardian);

        vm.assume(storageSlot != hashedLocation(hash, CONSUMED_ACTIONS_SLOT));

        proxied.submitRecoverChainId(_vm);

        // The error message is not "governance action already consumed" because the evmChainId is updated,
        // and the check for isFork() comes first than the check for action already consumed
        vm.expectRevert("not a fork");
        proxied.submitRecoverChainId(_vm);
    }
}
