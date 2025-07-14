// SPDX-License-Identifier: MIT

pragma solidity ^0.8.0;

import {Test} from "forge-std/Test.sol";
import {StdAssertions} from "forge-std/StdAssertions.sol";
import {Vm} from "forge-std/Vm.sol";
import {console} from "forge-std/console.sol";

import {
	VerificationV2,
	ShardInfo,
	OP_APPEND_THRESHOLD_KEY,
	OP_PULL_GUARDIAN_SETS,
	OP_REGISTER_GUARDIAN,
	OP_VERIFY_AND_DECODE_VAA,
	OP_VERIFY_VAA,
	OP_GET_GUARDIAN_SET_CURRENT,
	OP_GET_GUARDIAN_SET,
	OP_GET_SHARDS,
	OP_GET_THRESHOLD_CURRENT,
	OP_GET_THRESHOLD,
	GOVERNANCE_ADDRESS
} from "../src/evm/VerificationV2.sol";

import {MultisigVerification} from "../src/evm/MultisigVerification.sol";
import {SSTORE2} from "../src/evm/ExtStore.sol";
import {
	ThresholdVerification,
	MODULE_VERIFICATION_V2,
	ACTION_APPEND_THRESHOLD_KEY,
	Q
} from "../src/evm/ThresholdVerification.sol";
import {ThresholdVerificationState} from "../src/evm/ThresholdVerificationState.sol";
import {REGISTER_TYPE_HASH} from "../src/evm/EIP712Encoding.sol";

import {RawDispatcher} from "wormhole-solidity-sdk/RawDispatcher.sol";
import {keccak256SliceUnchecked} from "wormhole-solidity-sdk/utils/Keccak.sol";
import {reRevert} from "wormhole-solidity-sdk/utils/Revert.sol";
import {BytesParsing} from "wormhole-solidity-sdk/libraries/BytesParsing.sol";
import {ICoreBridge, CoreBridgeVM, GuardianSet} from "wormhole-sdk/interfaces/ICoreBridge.sol";
import {CHAIN_ID_ETHEREUM, CHAIN_ID_SOLANA} from "wormhole-sdk/constants/Chains.sol";

// See https://docs.soliditylang.org/en/v0.8.29/control-structures.html#panic-via-assert-and-error-via-require
// For some reason, `Panic` is a reserved identifier, but the error itself is not defined builtin so we end up needing to hack around.
// We define it as a function within VerificationHelper instead.
// error Panic(uint256);
uint256 constant outOfBounds = 0x32;

contract WormholeMock is ICoreBridge {
	GuardianSet[] private _guardianSets;

	constructor(address[] memory guardianSet) {
		if (guardianSet.length != 0) {
			_guardianSets = new GuardianSet[](1);
			_guardianSets[0] = GuardianSet({
				expirationTime: 0,
				keys: guardianSet
			});
		}
	}

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

/// @dev Cheat code address.
/// Calculated as `address(uint160(uint256(keccak256("hevm cheat code"))))`.
address constant VM_ADDRESS = 0x7109709ECfa91a80626fF3989D68f67F5b1DD12D;
Vm constant vm = Vm(VM_ADDRESS);

// Message helper functions
function createVaaV1(uint32 guardianSetIndex, uint256[] memory guardianPrivateKeys, bytes memory envelope) pure returns (bytes memory) {
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
	uint32 tssIndex,
	address r,
	uint256 s,
	bytes memory envelope
) pure returns (bytes memory) {
	return abi.encodePacked(
		// Header
		uint8(2), // version
		tssIndex, // TSS index
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
) pure returns (bytes memory) {
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
) pure returns (bytes memory) {
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

function execAppendThresholdKey(VerificationV2 verification, bytes memory encodedVaa) {
	(bool success, bytes memory response) = address(verification).call(abi.encodePacked(
		bytes4(RawDispatcher.exec768.selector),
		OP_APPEND_THRESHOLD_KEY,
		uint16(encodedVaa.length),
		bytes(encodedVaa)
	));

	if (!success) reRevert(response);
}

contract VerificationHelper is StdAssertions {
	using BytesParsing for bytes;

	error ExecutionFailed();
	error UnexpectedSuccess(bytes response);

	bytes4 constant RAW_DISPATCHER_EXEC = bytes4(0x00000eb6);
	bytes4 constant RAW_DISPATCHER_GET = bytes4(0x0008a112);

	function Panic(uint256) public {}

	function expectRevertAppendThresholdKey(VerificationV2 verification, bytes memory encodedVaa) public returns (bytes memory) {
		(bool success, bytes memory response) = address(verification).call(abi.encodePacked(
			bytes4(RawDispatcher.exec768.selector),
			OP_APPEND_THRESHOLD_KEY,
			uint16(encodedVaa.length),
			bytes(encodedVaa)
		));

		if (success) revert UnexpectedSuccess(response);

		return response;
	}

	function execPullGuardianSets(VerificationV2 verification, uint32 limit) public {
		(bool success, bytes memory response) = address(verification).call(abi.encodePacked(
			bytes4(RawDispatcher.exec768.selector),
			OP_PULL_GUARDIAN_SETS,
			uint32(limit)
		));

		if (!success) reRevert(response);
	}

	function execRegisterGuardian(
		VerificationV2 verification,
		uint32 thresholdKeyIndex,
		uint256 nonce,
		bytes32 guardianId,
		uint8 guardianIndex,
		bytes32 r,
		bytes32 s,
		uint8 v
	) public {
		(bool success, bytes memory response) = address(verification).call(abi.encodePacked(
			bytes4(RawDispatcher.exec768.selector),
			OP_REGISTER_GUARDIAN,
			uint32(thresholdKeyIndex),
			uint256(nonce),
			bytes32(guardianId),
			uint8(guardianIndex),
			bytes32(r),
			bytes32(s),
			uint8(v)
		));

		if (!success) reRevert(response);
	}

	function expectRevertRegisterGuardian(
		VerificationV2 verification,
		uint32 thresholdKeyIndex,
		uint256 nonce,
		bytes32 guardianId,
		uint8 guardianIndex,
		bytes32 r,
		bytes32 s,
		uint8 v
	) public returns (bytes memory) {
		(bool success, bytes memory response) = address(verification).call(abi.encodePacked(
			bytes4(RawDispatcher.exec768.selector),
			OP_REGISTER_GUARDIAN,
			uint32(thresholdKeyIndex),
			uint256(nonce),
			bytes32(guardianId),
			uint8(guardianIndex),
			bytes32(r),
			bytes32(s),
			uint8(v)
		));

		if (success) revert UnexpectedSuccess(response);
		return response;
	}

	function verifyAndDecodeVaa(VerificationV2 verification, bytes memory encodedVaa) public view returns (
    uint32 timestamp,
    uint32 nonce,
    uint16 emitterChainId,
    bytes32 emitterAddress,
    uint64 sequence,
    uint8 consistencyLevel,
    bytes memory payload
  ) {
		(bool success, bytes memory result) = address(verification).staticcall(abi.encodePacked(
			bytes4(RawDispatcher.get1959.selector),
			OP_VERIFY_AND_DECODE_VAA,
			uint16(encodedVaa.length),
			bytes(encodedVaa)
		));

		if (!success) reRevert(result);

		uint256 offset = 32;
		uint256 length;
		(length, offset) = result.asUint256MemUnchecked(offset);
		// Must be at least minimum body size
		assertGe(length, 51);

		(timestamp, offset) = result.asUint32MemUnchecked(offset);
		(nonce, offset) = result.asUint32MemUnchecked(offset);
		(emitterChainId, offset) = result.asUint16MemUnchecked(offset);
		(emitterAddress, offset) = result.asBytes32MemUnchecked(offset);
		(sequence, offset) = result.asUint64MemUnchecked(offset);
		(consistencyLevel, offset) = result.asUint8MemUnchecked(offset);
		(payload, offset) = result.sliceUint16PrefixedMemUnchecked(offset);
		assertEq(length + 32 * 2, offset);
		assertLe(offset, result.length);
	}

	function expectFailureVerifyAndDecodeVaa(VerificationV2 verification, bytes memory encodedVaa) public view returns (bytes memory) {
		(bool success, bytes memory response) = address(verification).staticcall(abi.encodePacked(
			bytes4(RawDispatcher.get1959.selector),
			OP_VERIFY_AND_DECODE_VAA,
			uint16(encodedVaa.length),
			bytes(encodedVaa)
		));

		if (success) revert UnexpectedSuccess(response);
		return response;
	}

	function verifyVaa(VerificationV2 verification, bytes memory encodedVaa) public view {
		(bool success, bytes memory response) = address(verification).staticcall(abi.encodePacked(
			bytes4(RawDispatcher.get1959.selector),
			OP_VERIFY_VAA,
			uint16(encodedVaa.length),
			bytes(encodedVaa)
		));

		if (!success) reRevert(response);
	}

	function expectFailureVerifyVaa(VerificationV2 verification, bytes memory encodedVaa) public view returns (bytes memory) {
		(bool success, bytes memory response) = address(verification).staticcall(abi.encodePacked(
			bytes4(RawDispatcher.get1959.selector),
			OP_VERIFY_VAA,
			uint16(encodedVaa.length),
			bytes(encodedVaa)
		));

		if (success) revert UnexpectedSuccess(response);
		return response;
	}

	function getCurrentThresholdKey() public pure returns (bytes memory) {
		return abi.encodePacked(
			OP_GET_THRESHOLD_CURRENT
		);
	}

	function decodeGetCurrentThresholdKey(bytes memory result, uint256 offset) public pure returns (uint256 thresholdKeyPubkey, uint32 thresholdKeyIndex, uint256 newOffset) {
		(thresholdKeyPubkey, newOffset) = result.asUint256MemUnchecked(offset);
		(thresholdKeyIndex, newOffset) = result.asUint32MemUnchecked(newOffset);
	}

	function getThresholdKey(uint32 index) public pure returns (bytes memory) {
		return abi.encodePacked(
			OP_GET_THRESHOLD,
			uint32(index)
		);
	}

	function decodeGetThresholdKey(bytes memory result, uint256 offset) public pure returns (uint256 thresholdKeyPubkey, uint32 expirationTime, uint256 newOffset) {
		(thresholdKeyPubkey, newOffset) = result.asUint256MemUnchecked(offset);
		(expirationTime, newOffset) = result.asUint32MemUnchecked(newOffset);
	}

	function getCurrentGuardianSet() public pure returns (bytes memory) {
		return abi.encodePacked(
			OP_GET_GUARDIAN_SET_CURRENT
		);
	}

	function decodeGetCurrentGuardianSet(bytes memory result, uint256 offset) public pure returns (address[] memory guardianSetAddrs, uint32 guardianSetIndex, uint256 newOffset) {
		uint8 guardianCount;
		(guardianCount, newOffset) = result.asUint8MemUnchecked(offset);

		guardianSetAddrs = new address[](guardianCount);
		for (uint256 i = 0; i < guardianCount; i++) {
			(guardianSetAddrs[i], newOffset) = result.asAddressMemUnchecked(newOffset + 12);
		}

		(guardianSetIndex, newOffset) = result.asUint32MemUnchecked(newOffset);
	}

	function getGuardianSet(uint32 index) public pure returns (bytes memory) {
		return abi.encodePacked(
			OP_GET_GUARDIAN_SET,
			uint32(index)
		);
	}
	
	function decodeGetGuardianSet(bytes memory result, uint256 offset) public pure returns (address[] memory guardianSetAddrs, uint32 expirationTime, uint256 newOffset) {
		uint8 guardianCount;
		(guardianCount, newOffset) = result.asUint8MemUnchecked(offset);

		guardianSetAddrs = new address[](guardianCount);
		for (uint256 i = 0; i < guardianCount; i++) {
			(guardianSetAddrs[i], newOffset) = result.asAddressMemUnchecked(newOffset + 12);
		}

		(expirationTime, newOffset) = result.asUint32MemUnchecked(newOffset);
	}
	
	function getShards(uint32 guardianSet) public pure returns (bytes memory) {
		return abi.encodePacked(
			OP_GET_SHARDS,
			uint32(guardianSet)
		);
	}

	function decodeGetShards(bytes memory result, uint256 offset) public pure returns (ShardInfo[] memory shards, uint256 newOffset) {
		uint8 shardCount;
		(shardCount, newOffset) = result.asUint8MemUnchecked(offset);

		shards = new ShardInfo[](shardCount);
		for (uint256 i = 0; i < shardCount; i++) {
			(shards[i].shard, newOffset) = result.asBytes32MemUnchecked(newOffset);
			(shards[i].id, newOffset) = result.asBytes32MemUnchecked(newOffset);
		}
	}

	function get(VerificationV2 verification, bytes memory queries) public view returns (bytes memory) {
		(bool success, bytes memory response) = address(verification).staticcall(abi.encodePacked(
			bytes4(RawDispatcher.get1959.selector),
			queries
		));

		if (!success) reRevert(response);

		(uint length,) = response.asUint256MemUnchecked(32);
		(bytes memory data, uint256 offset) = response.sliceMemUnchecked(64, length);
		assertEq(length + 64, offset);
		assertGe(response.length, offset);
		if (response.length > offset) {
			(bytes memory trailingZeroes,) = response.sliceMemUnchecked(offset, response.length - offset);
			for (uint i = 0; i < trailingZeroes.length; ++i) {
				assertEq(trailingZeroes[i], 0);
			}
		}
		return data;
	}
}

contract VerificationV2Test is Test, VerificationHelper {
	using BytesParsing for bytes;

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
	bytes private invalidVaaV1 = createVaaV1(0, guardianPrivateKeys2, createVaaEnvelope(uint32(block.timestamp), 0, CHAIN_ID_SOLANA, 0, 0, 0, new bytes(0)));

	// V2 test data
	uint256 private constant thresholdKey1 = 0x79380e24c7cbb0f88706dd035135020063aab3e7f403398ff7f995af0b8a770c << 1;

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
		_wormholeMock = new WormholeMock(guardianKeys1);

		// Create the verification contract
		_verificationV2 = new VerificationV2(_wormholeMock, 0, 1);
	}

	// V1 codepaths

	function test_pullGuardianSets() public {
		VerificationHelper.execPullGuardianSets(_verificationV2, 1);
	}


	function test_getCurrentGuardianSet() public {
		bytes memory result = VerificationHelper.get(_verificationV2, VerificationHelper.getCurrentGuardianSet());

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

		VerificationHelper.execPullGuardianSets(_verificationV2, 1);

		// Get the old guardian set
		bytes memory result = VerificationHelper.get(_verificationV2, VerificationHelper.getGuardianSet(0));

		// Decode the guardian set
		(address[] memory guardianSetAddrs, uint32 expirationTime,) = VerificationHelper.decodeGetGuardianSet(result, 0);
		assertEq(guardianSetAddrs.length, 1);
		assertEq(guardianSetAddrs[0], guardianKeys1[0]);
		assertEq(expirationTime, 0);

		// Get the new guardian set
		bytes memory result2 = VerificationHelper.get(_verificationV2, VerificationHelper.getGuardianSet(1));

		// Decode the guardian set
		(address[] memory guardianSetAddrs2, uint32 expirationTime2,) = VerificationHelper.decodeGetGuardianSet(result2, 0);
		assertEq(guardianSetAddrs2.length, 1);
		assertEq(guardianSetAddrs2[0], guardianKeys2[0]);
		assertEq(expirationTime2, 1);
	}

	function test_verifyVaaV1() public view {
		this.verifyVaa(_verificationV2, registerThresholdKeyVaa);
	}

	function testRevert_verifyVaaV1() public {
		bytes memory result = VerificationHelper.expectFailureVerifyVaa(_verificationV2, invalidVaaV1);
		(bytes4 error,) = result.asBytes4MemUnchecked(0);
		assertEq(error, MultisigVerification.MultisigSignatureVerificationFailed.selector);
		assertEq(result.length, 4);
	}

	function testRevert_verifyVaaV1_notRegisteredGuardianSet() public {
		uint32 fakeGuardianSetIndex = 5;
		bytes memory payload = createThresholdKeyUpdatePayload(0, thresholdKey1, 0, thresholdShards1);
		bytes memory envelope = createVaaEnvelope(uint32(block.timestamp), 0, CHAIN_ID_SOLANA, GOVERNANCE_ADDRESS, 0, 0, payload);
		bytes memory registerThresholdKeyVaa2  = createVaaV1(fakeGuardianSetIndex, guardianPrivateKeys1, envelope);

		bytes memory result = VerificationHelper.expectFailureVerifyVaa(_verificationV2, registerThresholdKeyVaa2);
		// Since we're accessing an internal array beyond what is stored on it, we get an out of bounds panic
		(bytes4 error,) = result.asBytes4MemUnchecked(0);
		assertEq(error, VerificationHelper.Panic.selector);
		(uint256 errorCode,) = result.asUint256MemUnchecked(4);
		assertEq(errorCode, outOfBounds);
		assertEq(result.length, 36);
	}

	function testRevert_verifyVaaV1_skippedGuardianSet() public {
		WormholeMock wormholeMock = new WormholeMock(guardianKeys1);

		wormholeMock.appendGuardianSet(GuardianSet({
			keys: guardianKeys1,
			expirationTime: 0
		}));

		wormholeMock.appendGuardianSet(GuardianSet({
			keys: guardianKeys1,
			expirationTime: 0
		}));

		wormholeMock.appendGuardianSet(GuardianSet({
			keys: guardianKeys1,
			expirationTime: 0
		}));

    uint256 initGuardianSetIndex = 2;
		VerificationV2 verificationV2 = new VerificationV2(wormholeMock, initGuardianSetIndex, 1);

    uint32 fakeGuardianSetIndex = 1;
		bytes memory payload = createThresholdKeyUpdatePayload(0, thresholdKey1, 0, thresholdShards1);
		bytes memory envelope = createVaaEnvelope(uint32(block.timestamp), 0, CHAIN_ID_SOLANA, GOVERNANCE_ADDRESS, 0, 0, payload);
		bytes memory registerThresholdKeyVaa2  = createVaaV1(fakeGuardianSetIndex, guardianPrivateKeys1, envelope);

		bytes memory result = VerificationHelper.expectFailureVerifyVaa(verificationV2, registerThresholdKeyVaa2);
		(bytes4 error,) = result.asBytes4MemUnchecked(0);
		// We get an invalid pointer error because
		// we're attempting to read from an uninitialized account.
		assertEq(error, SSTORE2.InvalidPointer.selector);
		assertEq(result.length, 4);
	}

	function test_verifyAndDecodeVaaV1() public {
		(
			uint32 timestamp,
			uint32 nonce,
			uint16 emitterChainId,
			bytes32 emitterAddress,
			uint64 sequence,
			uint8 consistencyLevel,
			bytes memory payload
		) = VerificationHelper.verifyAndDecodeVaa(_verificationV2, registerThresholdKeyVaa);

		assertEq(timestamp, block.timestamp);
		assertEq(nonce, 0);
		assertEq(emitterChainId, CHAIN_ID_SOLANA);
		assertEq(emitterAddress, GOVERNANCE_ADDRESS);
		assertEq(sequence, 0);
		assertEq(consistencyLevel, 0);
		assertEq(payload.length, 137);
	}

	function testRevert_verifyAndDecodeVaaV1() public {
		bytes memory result = VerificationHelper.expectFailureVerifyAndDecodeVaa(_verificationV2, invalidVaaV1);
		(bytes4 error,) = result.asBytes4MemUnchecked(0);
		assertEq(error, MultisigVerification.MultisigSignatureVerificationFailed.selector);
		assertEq(result.length, 4);
	}

	// V2 codepaths

	function test_appendThresholdKey() public {
		execAppendThresholdKey(_verificationV2, registerThresholdKeyVaa);
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

		execAppendThresholdKey(_verificationV2, registerThresholdKeyVaa);
		execAppendThresholdKey(_verificationV2, thresholdKeyVaa);
		execAppendThresholdKey(_verificationV2, thresholdKeyVaa2);
	}

	function testRevert_appendThresholdKey() public {
		bytes memory result = VerificationHelper.expectRevertAppendThresholdKey(_verificationV2, invalidVaaV1);
		(bytes4 error,) = result.asBytes4MemUnchecked(0);
		assertEq(error, MultisigVerification.MultisigSignatureVerificationFailed.selector);
		assertEq(result.length, 4);
	}

	function testRevert_appendThresholdKey_duplicatedKey() public {
		execAppendThresholdKey(_verificationV2, registerThresholdKeyVaa);
		bytes memory result = VerificationHelper.expectRevertAppendThresholdKey(_verificationV2, registerThresholdKeyVaa);
		(bytes4 error,) = result.asBytes4MemUnchecked(0);
		assertEq(error, ThresholdVerificationState.InvalidThresholdKeyIndex.selector);
		assertEq(result.length, 4);
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
	
		execAppendThresholdKey(_verificationV2, thresholdKeyVaa);
		bytes memory result = VerificationHelper.expectRevertAppendThresholdKey(_verificationV2, thresholdKeyVaa2);
		(bytes4 error,) = result.asBytes4MemUnchecked(0);
		assertEq(error, ThresholdVerificationState.InvalidThresholdKeyIndex.selector);
		assertEq(result.length, 4);
	}

	function testRevert_appendMaxThresholdKey() public {
		uint32 tssIndex = type(uint32).max;
		bytes memory payload = createThresholdKeyUpdatePayload(tssIndex, thresholdKey1, 0, thresholdShards1);
		bytes memory envelope = createVaaEnvelope(uint32(block.timestamp), 0, CHAIN_ID_SOLANA, GOVERNANCE_ADDRESS, 0, 0, payload);
		bytes memory thresholdKeyVaa = createVaaV1(0, guardianPrivateKeys1, envelope);

		bytes memory result = VerificationHelper.expectRevertAppendThresholdKey(_verificationV2, thresholdKeyVaa);
		(bytes4 error,) = result.asBytes4MemUnchecked(0);
		assertEq(error, ThresholdVerificationState.InvalidThresholdKeyIndex.selector);
		assertEq(result.length, 4);
	}

	function test_getCurrentThresholdKey() public {
		execAppendThresholdKey(_verificationV2, registerThresholdKeyVaa);

		bytes memory result = VerificationHelper.get(_verificationV2, VerificationHelper.getCurrentThresholdKey());

		(uint256 thresholdKeyPubkey, uint32 thresholdKeyIndex,) = VerificationHelper.decodeGetCurrentThresholdKey(result, 0);
		assertEq(thresholdKeyIndex, 0);
		assertEq(thresholdKeyPubkey, thresholdKey1);
	}

	function test_getThresholdKey() public {
		execAppendThresholdKey(_verificationV2, registerThresholdKeyVaa);

		bytes memory result = VerificationHelper.get(_verificationV2, VerificationHelper.getThresholdKey(0));

		(uint256 thresholdKeyPubkey, uint32 expirationTime,) = VerificationHelper.decodeGetThresholdKey(result, 0);
		assertEq(expirationTime, 0);
		assertEq(thresholdKeyPubkey, thresholdKey1);
	}

	function test_verifyVaaV2() public {
		execAppendThresholdKey(_verificationV2, registerThresholdKeyVaa);

		address r = address(0x636a8688ef4B82E5A121F7C74D821A5b07d695f3);
		uint256 s = 0xaa6d485b7d7b536442ea7777127d35af43ac539a491c0d85ee0f635eb7745b29;
		bytes memory payload = new bytes(49);
		bytes memory envelope = createVaaEnvelope(0, 0, 0, 0, 0, 0, payload);
		bytes memory vaa = createVaaV2(0, r, s, envelope);

		this.verifyVaa(_verificationV2, vaa);
	}

	function test_verifyVaaV2_direct() public {
		execAppendThresholdKey(_verificationV2, registerThresholdKeyVaa);

		address r = address(0x636a8688ef4B82E5A121F7C74D821A5b07d695f3);
		uint256 s = 0xaa6d485b7d7b536442ea7777127d35af43ac539a491c0d85ee0f635eb7745b29;
		bytes memory payload = new bytes(49);
		bytes memory envelope = createVaaEnvelope(0, 0, 0, 0, 0, 0, payload);
		bytes memory vaa = createVaaV2(0, r, s, envelope);

		_verificationV2.verifyVaa(vaa);
	}

	function testRevert_verifyVaaV2() public {
		execAppendThresholdKey(_verificationV2, registerThresholdKeyVaa);

		address r = address(0xE46Df5BEa4597CEF7D3c6EfF36356A3F0bA33a56);
		uint256 s = 0x1c2d1ca6fd3830e653d2abfc57956f3700059a661d8cabae684ea1bc62294e4c;
		bytes memory payload = new bytes(50);
		bytes memory envelope = createVaaEnvelope(0, 0, 0, 0, 0, 0, payload);
		bytes memory vaa = createVaaV2(0, r, s, envelope);

		bytes memory result = VerificationHelper.expectFailureVerifyVaa(_verificationV2, vaa);
		(bytes4 error,) = result.asBytes4MemUnchecked(0);
		assertEq(error, ThresholdVerification.ThresholdSignatureVerificationFailed.selector);
		assertEq(result.length, 4);
	}

	function testRevert_verifyVaaV2_unregisteredKey() public {
    execAppendThresholdKey(_verificationV2, registerThresholdKeyVaa);

		address r = address(0xE46Df5BEa4597CEF7D3c6EfF36356A3F0bA33a56);
		uint256 s = 0x1c2d1ca6fd3830e653d2abfc57956f3700059a661d8cabae684ea1bc62294e4c;
		bytes memory payload = new bytes(49);
		bytes memory envelope = createVaaEnvelope(0, 0, 0, 0, 0, 0, payload);
    uint32 notRegisteredKeyTssIndex = 3;
		bytes memory vaa = createVaaV2(notRegisteredKeyTssIndex, r, s, envelope);

		bytes memory result = VerificationHelper.expectFailureVerifyVaa(_verificationV2, vaa);
		(bytes4 error,) = result.asBytes4MemUnchecked(0);
		assertEq(error, ThresholdVerification.ThresholdSignatureVerificationFailed.selector);
		assertEq(result.length, 4);
	} 

	function test_verifyVaaV2_skippedKey() public {
    uint32 tssIndex = 5;
    bytes memory payload = createThresholdKeyUpdatePayload(tssIndex, thresholdKey1, 0, thresholdShards1);
		bytes memory envelope = createVaaEnvelope(uint32(block.timestamp), 0, CHAIN_ID_SOLANA, GOVERNANCE_ADDRESS, 0, 0, payload);
		bytes memory registerThresholdKeyVaa2 = createVaaV1(0, guardianPrivateKeys1, envelope);

		execAppendThresholdKey(_verificationV2, registerThresholdKeyVaa);
    execAppendThresholdKey(_verificationV2, registerThresholdKeyVaa2);

		address r = address(0xE46Df5BEa4597CEF7D3c6EfF36356A3F0bA33a56);
		uint256 s = 0x1c2d1ca6fd3830e653d2abfc57956f3700059a661d8cabae684ea1bc62294e4c;
		payload = new bytes(49);
		envelope = createVaaEnvelope(0, 0, 0, 0, 0, 0, payload);
    uint32 skippedTssIndex = 3;
		bytes memory vaa = createVaaV2(skippedTssIndex, r, s, envelope);

		bytes memory result = VerificationHelper.expectFailureVerifyVaa(_verificationV2, vaa);
		(bytes4 error,) = result.asBytes4MemUnchecked(0);
		assertEq(error, ThresholdVerification.ThresholdSignatureVerificationFailed.selector);
		assertEq(result.length, 4);
	}

	function test_verifyAndDecodeVaaV2() public {
		execAppendThresholdKey(_verificationV2, registerThresholdKeyVaa);

		address r = address(0x636a8688ef4B82E5A121F7C74D821A5b07d695f3);
		uint256 s = 0xaa6d485b7d7b536442ea7777127d35af43ac539a491c0d85ee0f635eb7745b29;
		bytes memory payload = new bytes(49);
		bytes memory envelope = createVaaEnvelope(0, 0, 0, 0, 0, 0, payload);
		bytes memory vaa = createVaaV2(0, r, s, envelope);

		(
			uint32 timestamp,
			uint32 nonce,
			uint16 emitterChainId,
			bytes32 emitterAddress,
			uint64 sequence,
			uint8 consistencyLevel,
			bytes memory decodedPayload
		) = VerificationHelper.verifyAndDecodeVaa(_verificationV2, vaa);

		assertEq(timestamp, 0);
		assertEq(nonce, 0);
		assertEq(emitterChainId, 0);
		assertEq(emitterAddress, 0);
		assertEq(sequence, 0);
		assertEq(consistencyLevel, 0);
		assertEq(decodedPayload.length, 49);
	}

	function testRevert_verifyAndDecodeVaaV2_signatureVerificationFailure() public {
		execAppendThresholdKey(_verificationV2, registerThresholdKeyVaa);

		address r = address(0xE46Df5BEa4597CEF7D3c6EfF36356A3F0bA33a56);
		uint256 s = 0x1c2d1ca6fd3830e653d2abfc57956f3700059a661d8cabae684ea1bc62294e4c;
		bytes memory payload = new bytes(50);
		bytes memory envelope = createVaaEnvelope(0, 0, 0, 0, 0, 0, payload);
		bytes memory vaa = createVaaV2(0, r, s, envelope);

		bytes memory result = VerificationHelper.expectFailureVerifyAndDecodeVaa(_verificationV2, vaa);
		(bytes4 error,) = result.asBytes4MemUnchecked(0);
		assertEq(error, ThresholdVerification.ThresholdSignatureVerificationFailed.selector);
		assertEq(result.length, 4);
	}

	// Shard update codepaths

	function test_getShardInfo() public {
		execAppendThresholdKey(_verificationV2, registerThresholdKeyVaa);

		bytes memory result = VerificationHelper.get(_verificationV2, VerificationHelper.getShards(0));

		(ShardInfo[] memory shards,) = VerificationHelper.decodeGetShards(result, 0);
		assertEq(shards.length, 1);
		assertEq(shards[0].shard, thresholdShards1[0].shard);
		assertEq(shards[0].id, thresholdShards1[0].id);
	}

	function test_setShardID() public {
		execAppendThresholdKey(_verificationV2, registerThresholdKeyVaa);

		uint256 fakeNonce = 10;
		bytes32 testID = bytes32(0x0000000000000000000000000000000000000000000000000000123412341234);
		bytes32 registerGuardianMessageHash = _verificationV2.getRegisterGuardianDigest(0, fakeNonce, testID);

		(uint8 v, bytes32 r, bytes32 s) = vm.sign(guardianPrivateKey1, registerGuardianMessageHash);
		v = v == 27 ? 0 : 1;
		
		VerificationHelper.execRegisterGuardian(
			_verificationV2,
			0,
			fakeNonce,
			testID,
			0,
			r, s, v
		);

		bytes memory result = VerificationHelper.get(_verificationV2, VerificationHelper.getShards(0));

		(ShardInfo[] memory shards,) = VerificationHelper.decodeGetShards(result, 0);
		assertEq(shards.length, 1);
		assertEq(shards[0].shard, thresholdShards1[0].shard);
		assertEq(shards[0].id, testID);
	}

	function testRevert_setShardID_invalidNonce() public {
		execAppendThresholdKey(_verificationV2, registerThresholdKeyVaa);

		uint256 nonce = 10;
		bytes32 testID = bytes32(0x0000000000000000000000000000000000000000000000000000123412341234);
		bytes32 register1 = _verificationV2.getRegisterGuardianDigest(0, nonce, testID);

		(uint8 v, bytes32 r, bytes32 s) = vm.sign(guardianPrivateKey1, register1);
		v = v == 27 ? 0 : 1;
		
		VerificationHelper.execRegisterGuardian(
			_verificationV2,
			0,
			nonce,
			testID,
			0,
			r, s, v
		);

		// Try to register using same nonce, should fail
		bytes32 register2 = _verificationV2.getRegisterGuardianDigest(0, nonce, thresholdShards1[0].id);
		(v, r, s) = vm.sign(guardianPrivateKey1, register2);
		v = v == 27 ? 0 : 1;

		bytes memory result = VerificationHelper.expectRevertRegisterGuardian(
			_verificationV2,
			0,
			nonce,
			thresholdShards1[0].id,
			0,
			r, s, v
		);
		(bytes4 error,) = result.asBytes4MemUnchecked(0);
		assertEq(error, VerificationV2.InvalidNonce.selector);
		assertEq(result.length, 4);

		result = VerificationHelper.get(_verificationV2, VerificationHelper.getShards(0));

		(ShardInfo[] memory shards,) = VerificationHelper.decodeGetShards(result, 0);
		assertEq(shards.length, 1);
		assertEq(shards[0].shard, thresholdShards1[0].shard);
		assertEq(shards[0].id, testID);
	}
}

contract VerificationV2Benchmark is Test {
	using BytesParsing for bytes;

	// Setup
	WormholeMock private _wormholeMock;
	VerificationV2 private _verificationV2;

	// V1 test data
	uint256 private constant guardianPrivateKey1 = 0x0123456701234567012345670123456701234567012345670123456701234567;
	uint256[] private guardianPrivateKeys1 = [guardianPrivateKey1];
	address private guardianPublicKey1;
	address[] private guardianKeys1;

	// V2 test data
	uint256 private constant thresholdKey1 = 0x79380e24c7cbb0f88706dd035135020063aab3e7f403398ff7f995af0b8a770c << 1;
  uint256 private constant thresholdKey2 = 0x44c90dfbe2a454987a65ce9e6f522c9c5c9d1dfb3c3aaaadcd0ae4f5366a2922 << 1;

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

		// Create a VAA to register the threshold key
		bytes memory payload = createThresholdKeyUpdatePayload(0, thresholdKey1, 0, thresholdShards1);
		bytes memory envelope = createVaaEnvelope(uint32(block.timestamp), 0, CHAIN_ID_SOLANA, GOVERNANCE_ADDRESS, 0, 0, payload);
		bytes memory registerThresholdKeyVaa = createVaaV1(0, guardianPrivateKeys1, envelope);

		bytes memory payload2 = createThresholdKeyUpdatePayload(1, thresholdKey2, 24 * 60 * 60, thresholdShards1);
		bytes memory envelope2 = createVaaEnvelope(uint32(block.timestamp), 0, CHAIN_ID_SOLANA, GOVERNANCE_ADDRESS, 0, 0, payload2);
		bytes memory registerThresholdKeyVaa2 = createVaaV1(0, guardianPrivateKeys1, envelope2);

		// Create a wormhole mock
		_wormholeMock = new WormholeMock(guardianKeys1);

		// Create the verification contract
		_verificationV2 = new VerificationV2(_wormholeMock, 0, 1);

		execAppendThresholdKey(_verificationV2, registerThresholdKeyVaa);
		execAppendThresholdKey(_verificationV2, registerThresholdKeyVaa2);
	}

	function test_benchmark_verifyVaaV2_direct() public {
		address r = address(0x636a8688ef4B82E5A121F7C74D821A5b07d695f3);
		uint256 s = 0xaa6d485b7d7b536442ea7777127d35af43ac539a491c0d85ee0f635eb7745b29;
		bytes memory payload = new bytes(49);
		bytes memory envelope = createVaaEnvelope(0, 0, 0, 0, 0, 0, payload);
		bytes memory vaa = createVaaV2(0, r, s, envelope);

		_verificationV2.verifyVaa(vaa);
	}

	function test_benchmark_verifyVaaV2_direct_big() public {
		bytes memory bigEnvelope = new bytes(5000);

		address r = 0xD970AcFC9e8ff8BE38b0Fd6C4Fe4DD4DDB744cb4;
		uint256 s = 0xfc201908d0a3aec1973711f48365deaa91180ef2771cb3744bccfc3ba77d6c77;
		bytes memory bigSchnorrVaa = createVaaV2(1, r, s, bigEnvelope);

		_verificationV2.verifyVaa(bigSchnorrVaa);
	}
}
