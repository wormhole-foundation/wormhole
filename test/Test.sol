// SPDX-License-Identifier: MIT

pragma solidity ^0.8.0;

import {Test} from "forge-std/Test.sol";
import {console} from "forge-std/console.sol";

import {
	VerificationV2,
	ShardInfo,
	OP_APPEND_THRESHOLD_KEY,
	OP_PULL_GUARDIAN_SETS,
	OP_REGISTER_GUARDIAN,
	OP_VERIFY_AND_DECODE_VAA,
	OP_VERIFY_VAA,
	OP_GUARDIAN_SET_GET_CURRENT,
	OP_GUARDIAN_SET_GET,
	OP_GUARDIAN_SHARDS_GET,
	OP_THRESHOLD_GET_CURRENT,
	OP_THRESHOLD_GET,
	GOVERNANCE_ADDRESS
} from "../src/evm/VerificationV2.sol";

import {MODULE_VERIFICATION_V2, ACTION_APPEND_THRESHOLD_KEY} from "../src/evm/ThresholdVerification.sol";
import {REGISTER_TYPE_HASH} from "../src/evm/EIP712Encoding.sol";

import {BytesParsing} from "wormhole-solidity-sdk/libraries/BytesParsing.sol";
import {ICoreBridge, CoreBridgeVM, GuardianSet} from "wormhole-sdk/interfaces/ICoreBridge.sol";
import {CHAIN_ID_ETHEREUM, CHAIN_ID_SOLANA} from "wormhole-sdk/constants/Chains.sol";

contract WormholeMock is ICoreBridge {
	function messageFee() external pure returns (uint256) {
		revert("Not implemented");
	}

	function publishMessage(uint32, bytes memory, uint8) external payable returns (uint64) {
		revert("Not implemented");
	}

	function parseAndVerifyVM(bytes calldata) external pure returns (CoreBridgeVM memory, bool, string memory) {
		revert("Not implemented");
	}

	function nextSequence(address) external pure returns (uint64) {
		revert("Not implemented");
	}

	function chainId() external pure returns (uint16) {
		return CHAIN_ID_ETHEREUM;
	}

	GuardianSet[] private _guardianSets;

	function getGuardianSet(uint32 index) external view returns (GuardianSet memory) {
		require(index < _guardianSets.length, "Guardian set index out of bounds");
		return _guardianSets[index];
	}

	function getCurrentGuardianSetIndex() external view returns (uint32) {
		require(_guardianSets.length > 0, "No guardian sets");
		return uint32(_guardianSets.length - 1);
	}

	function appendGuardianSet(GuardianSet memory guardianSet) external {
		_guardianSets.push(guardianSet);
	}
}

library VerificationHelper {
	using BytesParsing for bytes;

	bytes4 constant RAW_DISPATCHER_EXEC = bytes4(0x00000eb6);
	bytes4 constant RAW_DISPATCHER_GET = bytes4(0x0008a112);

	function exec(VerificationV2 verification, bytes memory data) public returns (bool success) {
		(success,) = address(verification).call(abi.encodePacked(RAW_DISPATCHER_EXEC, data));
	}

	function appendThresholdKey(bytes calldata encodedVaa) public pure returns (bytes memory) {
		return abi.encodePacked(
			OP_APPEND_THRESHOLD_KEY,
			uint16(encodedVaa.length),
			bytes(encodedVaa)
		);
	}

	function pullGuardianSets(uint32 limit) public pure returns (bytes memory) {
		return abi.encodePacked(
			OP_PULL_GUARDIAN_SETS,
			uint32(limit)
		);
	}

	function registerGuardian(
		uint32 thresholdKeyIndex,
		uint32 expirationTime,
		bytes32 guardianId,
		uint8 guardianIndex,
		bytes32 r,
		bytes32 s,
		uint8 v
	) public pure returns (bytes memory) {
		return abi.encodePacked(
			OP_REGISTER_GUARDIAN,
			uint32(thresholdKeyIndex),
			uint32(expirationTime),
			bytes32(guardianId),
			uint8(guardianIndex),
			bytes32(r),
			bytes32(s),
			uint8(v)
		);
	}
	
	function get(VerificationV2 verification, bytes memory data) public returns (bool success, bytes memory result) {
		(success, result) = address(verification).call(abi.encodePacked(RAW_DISPATCHER_GET, data));
		result = abi.decode(result, (bytes));
	}

	function verifyAndDecodeVaa(bytes calldata encodedVaa) public pure returns (bytes memory) {
		return abi.encodePacked(
			OP_VERIFY_AND_DECODE_VAA,
			uint16(encodedVaa.length),
			bytes(encodedVaa)
		);
	}

	function decodeVerifyAndDecodeVaa(bytes memory result, uint256 offset) public pure returns (
    uint32 timestamp,
    uint32 nonce,
    uint16 emitterChainId,
    bytes32 emitterAddress,
    uint64 sequence,
    uint8 consistencyLevel,
    bytes memory payload
  ) {
		(timestamp, nonce, emitterChainId, emitterAddress, sequence, consistencyLevel, payload) = abi.decode(result, (uint32, uint32, uint16, bytes32, uint64, uint8, bytes));
		// TODO: What should the offset be?
	}

	function verifyVaa(bytes calldata encodedVaa) public pure returns (bytes memory) {
		return abi.encodePacked(
			OP_VERIFY_VAA,
			uint16(encodedVaa.length),
			bytes(encodedVaa)
		);
	}

	function getCurrentThresholdKey() public pure returns (bytes memory) {
		return abi.encodePacked(
			OP_THRESHOLD_GET_CURRENT
		);
	}

	function decodeGetCurrentThresholdKey(bytes memory result, uint256 offset) public pure returns (uint256 thresholdKeyPubkey, uint32 thresholdKeyIndex, uint256 newOffset) {
		(thresholdKeyPubkey, thresholdKeyIndex) = abi.decode(result, (uint256, uint32));
		// TODO: What should the offset be?
	}

	function getThresholdKey(uint32 index) public pure returns (bytes memory) {
		return abi.encodePacked(
			OP_THRESHOLD_GET,
			uint32(index)
		);
	}

	function decodeGetThresholdKey(bytes memory result, uint256 offset) public pure returns (uint256 thresholdKeyPubkey, uint32 expirationTime, uint256 newOffset) {
		(thresholdKeyPubkey, expirationTime) = abi.decode(result, (uint256, uint32));
		// TODO: What should the offset be?
	}

	function getCurrentGuardianSet() public pure returns (bytes memory) {
		return abi.encodePacked(
			OP_GUARDIAN_SET_GET_CURRENT
		);
	}

	function decodeGetCurrentGuardianSet(bytes memory result, uint256 offset) public pure returns (address[] memory guardianSetAddrs, uint32 guardianSetIndex, uint256 newOffset) {
		(guardianSetAddrs, guardianSetIndex) = abi.decode(result, (address[], uint32));
		// TODO: What should the offset be?
	}

	function getGuardianSet(uint32 index) public pure returns (bytes memory) {
		return abi.encodePacked(
			OP_GUARDIAN_SET_GET,
			uint32(index)
		);
	}
	
	function decodeGetGuardianSet(bytes memory result, uint256 offset) public pure returns (address[] memory guardianSetAddrs, uint32 expirationTime, uint256 newOffset) {
		(guardianSetAddrs, expirationTime) = abi.decode(result, (address[], uint32));
		// TODO: What should the offset be?
	}
	
	function getShardInfo(uint32 guardianSet) public pure returns (bytes memory) {
		return abi.encodePacked(
			OP_GUARDIAN_SHARDS_GET,
			uint32(guardianSet)
		);
	}

	function decodeGetShardInfo(bytes memory result, uint256 offset) public pure returns (ShardInfo[] memory shards, uint256 newOffset) {
		(shards) = abi.decode(result, (ShardInfo[]));
		// TODO: What should the offset be?
	}
}

contract VerificationV2Test is Test {
	// Setup
	WormholeMock private _wormholeMock;
	VerificationV2 private _verificationV2;

	// V1 test data
	uint256 private constant guardianPrivateKey1 = 0x0123456701234567012345670123456701234567012345670123456701234567;
	uint256[] private guardianPrivateKeys1 = [guardianPrivateKey1];
	address private guardianPublicKey1;
	address[] private guardianKeys1;

	uint256 private constant guardianPrivateKey2 = 0x0123456701234567012345670123456701234567012345670123456701234568;
	uint256[] private guardianPrivateKeys2 = [guardianPrivateKey2];
	address private guardianPublicKey2;
	address[] private guardianKeys2;

	bytes private registerThresholdKeyVaa;
	bytes private invalidVaaV1 = hex"01234567";

	// V2 test data
	uint256 private constant thresholdKey1 = 0x1cafae803bf91a2e5494162625d34fda2f69db7c1f3589938647bc2abd4a0a0f << 1;

	ShardInfo[] private thresholdShards1 = [
		ShardInfo({
			shard: bytes32(0x0000000000000000000000000000000000000000000000000000000000001234),
			id: bytes32(0x0000000000000000000000000000000000000000000000000000000000005678)
		})
	];

	function setUp() public {
		// Create a couple guardian set
		guardianPublicKey1 = vm.addr(guardianPrivateKey1);
		guardianKeys1 = [guardianPublicKey1];

		guardianPublicKey2 = vm.addr(guardianPrivateKey2);
		guardianKeys2 = [guardianPublicKey2];

		// Create a VAA to register the threshold key
		bytes memory payload = createThresholdKeyUpdatePayload(0, thresholdKey1, 0, thresholdShards1);
		bytes memory envelope = createVaaEnvelope(uint32(block.timestamp), 0, CHAIN_ID_SOLANA, GOVERNANCE_ADDRESS, 0, 0, payload);
		registerThresholdKeyVaa = createVaaV1(0, guardianPrivateKeys1, envelope);

		// Create a wormhole mock
		_wormholeMock = new WormholeMock();

		_wormholeMock.appendGuardianSet(GuardianSet({
			keys: guardianKeys1,
			expirationTime: 0
		}));

		// Create the verification contract
		_verificationV2 = new VerificationV2(_wormholeMock, 0, 1);
	}

	// Message helper functions
	function createVaaV1(uint32 guardianSetIndex, uint256[] memory guardianPrivateKeys, bytes memory envelope) public pure returns (bytes memory) {
		uint256 guardianCount = guardianPrivateKeys.length;
		bytes memory signatures = new bytes(guardianCount * 66); // 66 bytes per signature (1 byte index, 32 bytes r, 32 bytes s, 1 byte v)
		bytes32 preMessage = keccak256(envelope);
		bytes32 message = keccak256(abi.encodePacked(preMessage));

		for (uint256 i = 0; i < guardianCount; i++) {
			(uint8 v, bytes32 r, bytes32 s) = vm.sign(guardianPrivateKeys[i], message);

			assembly ("memory-safe") {
				let offset := add(add(signatures, 32), mul(i, 66))
				mstore8(offset, i)
				mstore(add(offset, 1), r)
				mstore(add(offset, 33), s)
				mstore8(add(offset, 65), eq(v, 28))
			}
		}

		return abi.encodePacked(
			// Header
			uint8(1), // version
			uint32(guardianSetIndex), // guardian set index
			uint8(guardianCount), // signature count
			signatures, // signatures
			envelope // envelope
		);
	}

	function createVaaV2(
		uint32 guardianSetIndex,
		address r,
		uint256 s,
		bytes memory envelope
	) public pure returns (bytes memory) {
		return abi.encodePacked(
			// Header
			uint8(2), // version
			guardianSetIndex, // guardian set index
			r,
			s,
			envelope
		);
	}
	function createVaaEnvelope(
		uint32 timestamp,
		uint32 nonce,
		uint16 emitterChainId,
		bytes32 emitterAddress,
		uint64 sequence,
		uint8 consistencyLevel,
		bytes memory payload
	) public pure returns (bytes memory) {
		return abi.encodePacked(
			timestamp,
			nonce,
			emitterChainId,
			emitterAddress,
			sequence,
			consistencyLevel,
			payload
		);
	}

	function createThresholdKeyUpdatePayload(
		uint32 newTSSIndex,
		uint256 newThresholdPubkey,
		uint32 expirationDelaySeconds,
		ShardInfo[] memory shards
	) public pure returns (bytes memory) {
		bytes32[] memory shardsData = new bytes32[](shards.length * 2);
		for (uint256 i = 0; i < shards.length; i++) {
			shardsData[i * 2] = shards[i].shard;
			shardsData[i * 2 + 1] = shards[i].id;
		}

		return abi.encodePacked(
			MODULE_VERIFICATION_V2,
			ACTION_APPEND_THRESHOLD_KEY,
			newTSSIndex,
			newThresholdPubkey,
			expirationDelaySeconds,
			shardsData
		);
	}

	// V1 codepaths

	function test_pullGuardianSets() public {
		_wormholeMock.appendGuardianSet(GuardianSet({
			keys: guardianKeys2,
			expirationTime: 0
		}));

		assertEq(VerificationHelper.exec(_verificationV2, VerificationHelper.pullGuardianSets(1)), true);
	}

	function test_getCurrentGuardianSet() public {
		(bool success, bytes memory result) = VerificationHelper.get(_verificationV2, VerificationHelper.getCurrentGuardianSet());
		assertEq(success, true);

		(address[] memory guardianSetAddrs, uint32 expirationTime,) = VerificationHelper.decodeGetCurrentGuardianSet(result, 0);
		assertEq(guardianSetAddrs.length, 1);
		assertEq(guardianSetAddrs[0], guardianKeys1[0]);
		assertEq(expirationTime, 0);
	}

	function test_getGuardianSet() public {
		// Add a new guardian set
		_wormholeMock.appendGuardianSet(GuardianSet({
			keys: guardianKeys2,
			expirationTime: 1
		}));

		assertEq(VerificationHelper.exec(_verificationV2, VerificationHelper.pullGuardianSets(1)), true);
		
		// Get the old guardian set
		(bool success2, bytes memory result) = VerificationHelper.get(_verificationV2, VerificationHelper.getGuardianSet(0));
		assertEq(success2, true);

		// Decode the guardian set
		(address[] memory guardianSetAddrs, uint32 expirationTime,) = VerificationHelper.decodeGetGuardianSet(result, 0);
		assertEq(guardianSetAddrs.length, 1);
		assertEq(guardianSetAddrs[0], guardianKeys1[0]);
		assertEq(expirationTime, 0);

		// Get the new guardian set
		(bool success3, bytes memory result2) = VerificationHelper.get(_verificationV2, VerificationHelper.getGuardianSet(1));
		assertEq(success3, true);

		// Decode the guardian set
		(address[] memory guardianSetAddrs2, uint32 expirationTime2,) = VerificationHelper.decodeGetGuardianSet(result2, 0);
		assertEq(guardianSetAddrs2.length, 1);
		assertEq(guardianSetAddrs2[0], guardianKeys2[0]);
		assertEq(expirationTime2, 1);
	}

	function test_verifyVaaV1() public {
		(bool success, bytes memory result) = VerificationHelper.get(_verificationV2, VerificationHelper.verifyVaa(registerThresholdKeyVaa));
		assertEq(success, true);
		assertEq(result.length, 0);
	}

	function testRevert_verifyVaaV1() public {
		bytes memory command = VerificationHelper.verifyVaa(invalidVaaV1);
		vm.expectRevert();
		(bool success, bytes memory result) = VerificationHelper.get(_verificationV2, command);
		assertEq(success, false);
		assertEq(result.length, 0);
	}

	function test_veifyAndDecodeVaaV1() public {
		(bool success, bytes memory result) = VerificationHelper.get(_verificationV2, VerificationHelper.verifyAndDecodeVaa(registerThresholdKeyVaa));
		assertEq(success, true);
	}

	function testRevert_veifyAndDecodeVaaV1() public {
		bytes memory command = VerificationHelper.verifyAndDecodeVaa(invalidVaaV1);
		vm.expectRevert();
		(bool success, bytes memory result) = VerificationHelper.get(_verificationV2, command);
		assertEq(success, false);
		assertEq(result.length, 0);
	}

	// V2 codepaths

	function test_appendThresholdKey() public {
		assertEq(VerificationHelper.exec(_verificationV2, VerificationHelper.appendThresholdKey(registerThresholdKeyVaa)), true);
	}

	function test_appendMultipleThresholdKey() public {
		uint32 tssIndex = 8;
		bytes memory payload = createThresholdKeyUpdatePayload(tssIndex, thresholdKey1, 0, thresholdShards1);
		bytes memory envelope = createVaaEnvelope(uint32(block.timestamp), 0, CHAIN_ID_SOLANA, GOVERNANCE_ADDRESS, 0, 0, payload);
		bytes memory thresholdKeyVaa = createVaaV1(0, guardianPrivateKeys1, envelope);
    
		uint32 tssIndex2 = 15;
		bytes memory payload2 = createThresholdKeyUpdatePayload(tssIndex2, thresholdKey1, 0, thresholdShards1);
		bytes memory envelope2 = createVaaEnvelope(uint32(block.timestamp), 0, CHAIN_ID_SOLANA, GOVERNANCE_ADDRESS, 0, 0, payload2);
		bytes memory thresholdKeyVaa2 = createVaaV1(0, guardianPrivateKeys1, envelope2);

		assertEq(VerificationHelper.exec(_verificationV2, VerificationHelper.appendThresholdKey(registerThresholdKeyVaa)), true);
		assertEq(VerificationHelper.exec(_verificationV2, VerificationHelper.appendThresholdKey(thresholdKeyVaa)), true);
		assertEq(VerificationHelper.exec(_verificationV2, VerificationHelper.appendThresholdKey(thresholdKeyVaa2)), true);
	}

	function testRevert_appendThresholdKey() public {
		bytes memory command = VerificationHelper.appendThresholdKey(invalidVaaV1);
		assertEq(VerificationHelper.exec(_verificationV2, command), false);
	}

	function testRevert_appendThresholdKey_duplicatedKey() public {
		assertEq(VerificationHelper.exec(_verificationV2, VerificationHelper.appendThresholdKey(registerThresholdKeyVaa)), true);
		assertEq(VerificationHelper.exec(_verificationV2, VerificationHelper.appendThresholdKey(registerThresholdKeyVaa)), false);
	}

	function testRevert_appendOldThresholdKey() public {
		uint32 tssIndex = 8;
		bytes memory payload = createThresholdKeyUpdatePayload(tssIndex, thresholdKey1, 0, thresholdShards1);
		bytes memory envelope = createVaaEnvelope(uint32(block.timestamp), 0, CHAIN_ID_SOLANA, GOVERNANCE_ADDRESS, 0, 0, payload);
		bytes memory thresholdKeyVaa = createVaaV1(0, guardianPrivateKeys1, envelope);
    
		uint32 tssIndex2 = 2;
		bytes memory payload2 = createThresholdKeyUpdatePayload(tssIndex2, thresholdKey1, 0, thresholdShards1);
		bytes memory envelope2 = createVaaEnvelope(uint32(block.timestamp), 0, CHAIN_ID_SOLANA, GOVERNANCE_ADDRESS, 0, 0, payload2);
		bytes memory thresholdKeyVaa2 = createVaaV1(0, guardianPrivateKeys1, envelope2);
	
		assertEq(VerificationHelper.exec(_verificationV2, VerificationHelper.appendThresholdKey(thresholdKeyVaa)), true);
		assertEq(VerificationHelper.exec(_verificationV2, VerificationHelper.appendThresholdKey(thresholdKeyVaa2)), false);
	}

	function testRevert_appendMaxThresholdKey() public {
		uint32 tssIndex = type(uint32).max;
		bytes memory payload = createThresholdKeyUpdatePayload(tssIndex, thresholdKey1, 0, thresholdShards1);
		bytes memory envelope = createVaaEnvelope(uint32(block.timestamp), 0, CHAIN_ID_SOLANA, GOVERNANCE_ADDRESS, 0, 0, payload);
		bytes memory thresholdKeyVaa = createVaaV1(0, guardianPrivateKeys1, envelope);

		assertEq(VerificationHelper.exec(_verificationV2, VerificationHelper.appendThresholdKey(thresholdKeyVaa)), false);
	}

	function test_getCurrentThresholdKey() public {
		assertEq(VerificationHelper.exec(_verificationV2, VerificationHelper.appendThresholdKey(registerThresholdKeyVaa)), true);

		(bool success, bytes memory result) = VerificationHelper.get(_verificationV2, VerificationHelper.getCurrentThresholdKey());
		assertEq(success, true);

		(uint256 thresholdKeyPubkey, uint32 thresholdKeyIndex,) = VerificationHelper.decodeGetCurrentThresholdKey(result, 0);
		assertEq(thresholdKeyIndex, 0);
		assertEq(thresholdKeyPubkey, thresholdKey1);
	}

	function test_getThresholdKey() public {
		assertEq(VerificationHelper.exec(_verificationV2, VerificationHelper.appendThresholdKey(registerThresholdKeyVaa)), true);

		(bool success, bytes memory result) = VerificationHelper.get(_verificationV2, VerificationHelper.getThresholdKey(0));
		assertEq(success, true);

		(uint256 thresholdKeyPubkey, uint32 expirationTime,) = VerificationHelper.decodeGetThresholdKey(result, 0);
		assertEq(expirationTime, 0);
		assertEq(thresholdKeyPubkey, thresholdKey1);
	}

	function test_verifyVaaV2() public {
		assertEq(VerificationHelper.exec(_verificationV2, VerificationHelper.appendThresholdKey(registerThresholdKeyVaa)), true);

		address r = address(0xE46Df5BEa4597CEF7D3c6EfF36356A3F0bA33a56);
		uint256 s = 0x1c2d1ca6fd3830e653d2abfc57956f3700059a661d8cabae684ea1bc62294e4c;
		bytes memory payload = new bytes(49);
		bytes memory envelope = createVaaEnvelope(0, 0, 0, 0, 0, 0, payload);
		bytes memory vaa = createVaaV2(0, r, s, envelope);
		bytes memory command = VerificationHelper.verifyVaa(vaa);

		(bool success, bytes memory result) = VerificationHelper.get(_verificationV2, command);
		assertEq(success, true);
		assertEq(result.length, 0);
	}

	function testRevert_verifyVaaV2() public {
		assertEq(VerificationHelper.exec(_verificationV2, VerificationHelper.appendThresholdKey(registerThresholdKeyVaa)), true);

		address r = address(0xE46Df5BEa4597CEF7D3c6EfF36356A3F0bA33a56);
		uint256 s = 0x1c2d1ca6fd3830e653d2abfc57956f3700059a661d8cabae684ea1bc62294e4c;
		bytes memory payload = new bytes(50);
		bytes memory envelope = createVaaEnvelope(0, 0, 0, 0, 0, 0, payload);
		bytes memory vaa = createVaaV2(0, r, s, envelope);
		bytes memory command = VerificationHelper.verifyVaa(vaa);

		vm.expectRevert();
		(bool success, bytes memory result) = VerificationHelper.get(_verificationV2, command);
		assertEq(success, false);
		assertEq(result.length, 0);
	}

	function test_veifyAndDecodeVaaV2() public {
		assertEq(VerificationHelper.exec(_verificationV2, VerificationHelper.appendThresholdKey(registerThresholdKeyVaa)), true);

		address r = address(0xE46Df5BEa4597CEF7D3c6EfF36356A3F0bA33a56);
		uint256 s = 0x1c2d1ca6fd3830e653d2abfc57956f3700059a661d8cabae684ea1bc62294e4c;
		bytes memory payload = new bytes(49);
		bytes memory envelope = createVaaEnvelope(0, 0, 0, 0, 0, 0, payload);
		bytes memory vaa = createVaaV2(0, r, s, envelope);
		bytes memory command = VerificationHelper.verifyAndDecodeVaa(vaa);

		(bool success, bytes memory result) = VerificationHelper.get(_verificationV2, command);
		assertEq(success, true);

		(
			uint32 timestamp,
			uint32 nonce,
			uint16 emitterChainId,
			bytes32 emitterAddress,
			uint64 sequence,
			uint8 consistencyLevel,
			bytes memory decodedPayload
		) = VerificationHelper.decodeVerifyAndDecodeVaa(result, 0);

		assertEq(timestamp, 0);
		assertEq(nonce, 0);
		assertEq(emitterChainId, 0);
		assertEq(emitterAddress, 0);
		assertEq(sequence, 0);
		assertEq(consistencyLevel, 0);
		assertEq(decodedPayload.length, 49);
	}

	function testRevert_veifyAndDecodeVaaV2() public {
		assertEq(VerificationHelper.exec(_verificationV2, VerificationHelper.appendThresholdKey(registerThresholdKeyVaa)), true);

		address r = address(0xE46Df5BEa4597CEF7D3c6EfF36356A3F0bA33a56);
		uint256 s = 0x1c2d1ca6fd3830e653d2abfc57956f3700059a661d8cabae684ea1bc62294e4c;
		bytes memory payload = new bytes(50);
		bytes memory envelope = createVaaEnvelope(0, 0, 0, 0, 0, 0, payload);
		bytes memory vaa = createVaaV2(0, r, s, envelope);
		bytes memory command = VerificationHelper.verifyAndDecodeVaa(vaa);

		vm.expectRevert();
		(bool success, bytes memory result) = VerificationHelper.get(_verificationV2, command);
		assertEq(success, false);
		assertEq(result.length, 0);
	}

	// Shard update codepaths

	function test_getShardInfo() public {
		assertEq(VerificationHelper.exec(_verificationV2, VerificationHelper.appendThresholdKey(registerThresholdKeyVaa)), true);

		(bool success, bytes memory result) = VerificationHelper.get(_verificationV2, VerificationHelper.getShardInfo(0));
		assertEq(success, true);

		(ShardInfo[] memory shards,) = VerificationHelper.decodeGetShardInfo(result, 0);
		assertEq(shards.length, 1);
		assertEq(shards[0].shard, thresholdShards1[0].shard);
		assertEq(shards[0].id, thresholdShards1[0].id);
	}

	function test_setShardID() public {
		assertEq(VerificationHelper.exec(_verificationV2, VerificationHelper.appendThresholdKey(registerThresholdKeyVaa)), true);

		bytes32 testID = bytes32(0x0000000000000000000000000000000000000000000000000000123412341234);
		bytes32 registerGuardianMessageHash = _verificationV2.getRegisterGuardianDigest(0, uint32(block.timestamp + 1000), testID);

		(uint8 v, bytes32 r, bytes32 s) = vm.sign(guardianPrivateKey1, registerGuardianMessageHash);
		v = v == 27 ? 0 : 1;
		
		assertEq(VerificationHelper.exec(_verificationV2, VerificationHelper.registerGuardian(
			0,
			uint32(block.timestamp + 1000),
			testID,
			0,
			r, s, v
		)), true);

		(bool success, bytes memory result) = VerificationHelper.get(_verificationV2, VerificationHelper.getShardInfo(0));
		assertEq(success, true);

		(ShardInfo[] memory shards,) = VerificationHelper.decodeGetShardInfo(result, 0);
		assertEq(shards.length, 1);
		assertEq(shards[0].shard, thresholdShards1[0].shard);
		assertEq(shards[0].id, testID);
	}

	function testRevert_setShardID() public {
		assertEq(VerificationHelper.exec(_verificationV2, VerificationHelper.appendThresholdKey(registerThresholdKeyVaa)), true);

		// Create an expired message
		bytes32 testID = bytes32(0x0000000000000000000000000000000000000000000000000000123412341234);
		bytes32 registerGuardianMessageHash = _verificationV2.getRegisterGuardianDigest(0, uint32(block.timestamp - 100), testID);

		(uint8 v, bytes32 r, bytes32 s) = vm.sign(guardianPrivateKey1, registerGuardianMessageHash);
		v = v == 27 ? 0 : 1;
		
		assertEq(VerificationHelper.exec(_verificationV2, VerificationHelper.registerGuardian(
			0,
			uint32(block.timestamp + 1000),
			testID,
			0,
			r, s, v
		)), false);
	}
}
