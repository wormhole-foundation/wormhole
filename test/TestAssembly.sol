// SPDX-License-Identifier: MIT

pragma solidity ^0.8.27;

import {console} from "forge-std/console.sol";
import {Test} from "forge-std/Test.sol";

import {CHAIN_ID_SOLANA} from "wormhole-solidity-sdk/constants/Chains.sol";
import {VaaBody} from "wormhole-solidity-sdk/libraries/VaaLib.sol";
import {RawDispatcher} from "wormhole-solidity-sdk/RawDispatcher.sol";

import {WormholeMock} from "./Test.sol";
import {VaaBuilder, ShardData} from "./TestOptimized.sol";
import {Verification, GOVERNANCE_ADDRESS, EXEC_PULL_MULTISIG_KEY_DATA, EXEC_APPEND_SCHNORR_KEY, EXEC_UPDATE_SHARD_ID, GET_SCHNORR_SHARD_DATA, MODULE_VERIFICATION_V2, ACTION_APPEND_SCHNORR_KEY, EXEC_APPEND_SCHNORR_SHARD_DATA} from "../src/evm/AssemblyOptimized.sol";

contract TestAssembly is Test, VaaBuilder {
	uint256 private constant SHARD_COUNT = 19;
	uint256 private constant SHARD_QUORUM = 13;
	uint256[] private guardianPrivateKeys1 = _getGuardianPrivateKeys(SHARD_COUNT);
	address[] private guardianSet1 = _getGuardianPublicKeys(guardianPrivateKeys1);

	uint256 private constant schnorrKey1 = 0x79380e24c7cbb0f88706dd035135020063aab3e7f403398ff7f995af0b8a770c << 1;
	uint256 private constant schnorrKey2 = 0x44c90dfbe2a454987a65ce9e6f522c9c5c9d1dfb3c3aaaadcd0ae4f5366a2922 << 1;

	bytes private multisigVaa;
	bytes private schnorrVaa;
	bytes private bigMultisigVaa;
	bytes private bigSchnorrVaa;
	bytes private invalidVersionVaa;
	bytes private invalidMultisigVaa;
	bytes private invalidSchnorrVaa;

	ShardData[] private schnorrShards;

	WormholeMock _wormhole = new WormholeMock(guardianSet1);
	Verification _verification = new Verification(_wormhole, 0, 0, 0);

	function _getGuardianPrivateKeys(uint256 count) internal pure returns (uint256[] memory) {
		uint256[] memory keys = new uint256[](count);
		uint256 baseKey = 0x1234567890123456789012345678901234567890123456789012345678900000;
		for (uint256 i = 0; i < count; i++) {
			keys[i] = baseKey + i;
		}
		return keys;
	}

	function _getGuardianPublicKeys(uint256[] memory privateKeys) internal pure returns (address[] memory) {
		address[] memory publicKeys = new address[](privateKeys.length);
		for (uint256 i = 0; i < privateKeys.length; i++) {
			publicKeys[i] = vm.addr(privateKeys[i]);
		}
		return publicKeys;
	}

	function _warmSchnorrSlots(uint32 keyIndex) internal {
		vm.warmSlot(address(_verification), bytes32(uint256((2 << 48) + keyIndex)));
		vm.warmSlot(address(_verification), bytes32(uint256((3 << 48) + keyIndex)));
	}

	function createAppendSchnorrKeyMessage2(
		uint32 newTSSIndex,
		uint256 newThresholdPubkey,
		uint32 expirationDelaySeconds,
		bytes32 initialShardDataHash
	) public pure returns (bytes memory) {
		return abi.encodePacked(
			MODULE_VERIFICATION_V2,
			ACTION_APPEND_SCHNORR_KEY,
			newTSSIndex,
			newThresholdPubkey,
			expirationDelaySeconds,
			initialShardDataHash
		);
	}

	function setUp() public {
		bytes memory smallEnvelope = new bytes(100);
		uint256[] memory guardianPrivateKeysSlice = new uint256[](SHARD_QUORUM);
		for (uint256 i = 0; i < SHARD_QUORUM; i++) {
			guardianPrivateKeysSlice[i] = guardianPrivateKeys1[i];
		}
		multisigVaa = createMultisigVaa(0, guardianPrivateKeysSlice, smallEnvelope);

		address r = address(0x636a8688ef4B82E5A121F7C74D821A5b07d695f3);
		uint256 s = 0xaa6d485b7d7b536442ea7777127d35af43ac539a491c0d85ee0f635eb7745b29;
		schnorrVaa = createSchnorrVaa(0, r, s, smallEnvelope);

		bytes memory bigEnvelope = new bytes(5000);
		bigMultisigVaa = createMultisigVaa(0, guardianPrivateKeys1, bigEnvelope);

		address r2 = 0xD970AcFC9e8ff8BE38b0Fd6C4Fe4DD4DDB744cb4;
		uint256 s2 = 0xfc201908d0a3aec1973711f48365deaa91180ef2771cb3744bccfc3ba77d6c77;
		bigSchnorrVaa = createSchnorrVaa(1, r2, s2, bigEnvelope);

		invalidVersionVaa = new bytes(100);

		invalidMultisigVaa = new bytes(100);
		invalidMultisigVaa[0] = 0x01;

		invalidSchnorrVaa = new bytes(100);
		invalidSchnorrVaa[0] = 0x02;

		// Initialize the contract with the multisig and schnorr keys
		schnorrShards = new ShardData[](SHARD_COUNT);
		for (uint256 i = 0; i < SHARD_COUNT; i++) {
			schnorrShards[i] = ShardData({
				shard: bytes32(vm.randomUint()),
				id: bytes32(vm.randomUint())
			});
		}

		bytes memory schnorrShardsRaw = new bytes(0);
		for (uint256 i = 0; i < SHARD_COUNT; i++) {
			schnorrShardsRaw = abi.encodePacked(schnorrShardsRaw, schnorrShards[i].shard, schnorrShards[i].id);
		}

		bytes32 schnorrShardDataHash = keccak256(schnorrShardsRaw);

		bytes memory payload = createAppendSchnorrKeyMessage2(0, schnorrKey1, 0, schnorrShardDataHash);
		bytes memory envelope = createVaaEnvelope(uint32(block.timestamp), 0, CHAIN_ID_SOLANA, GOVERNANCE_ADDRESS, 0, 0, payload);
		bytes memory registerSchnorrKeyVaa = createMultisigVaa(0, guardianPrivateKeys1, envelope);

		payload = createAppendSchnorrKeyMessage2(1, schnorrKey2, 24 * 60 * 60, schnorrShardDataHash);
		envelope = createVaaEnvelope(uint32(block.timestamp), 0, CHAIN_ID_SOLANA, GOVERNANCE_ADDRESS, 0, 0, payload);
		bytes memory registerSchnorrKeyVaa2 = createMultisigVaa(0, guardianPrivateKeys1, envelope);

		bytes memory message = abi.encodePacked(
			RawDispatcher.exec768.selector,
			EXEC_PULL_MULTISIG_KEY_DATA,
			uint32(1),
			EXEC_APPEND_SCHNORR_KEY,
			registerSchnorrKeyVaa,
			EXEC_APPEND_SCHNORR_SHARD_DATA,
			schnorrShardsRaw,
			EXEC_APPEND_SCHNORR_KEY,
			registerSchnorrKeyVaa2,
			EXEC_APPEND_SCHNORR_SHARD_DATA,
			schnorrShardsRaw
		);

		(bool success, ) = address(_verification).call(message);
		assert(success);
	}

	function testRevert_verifyInvalidVersionVaa() public {
		vm.expectRevert();
		_verification.verifyVaa_U7N5(invalidVersionVaa);
	}

	function test_verifyMultisigVaa() public view {
		_verification.verifyVaa_U7N5(multisigVaa);
	}

	function testRevert_verifyInvalidMultisigVaa() public {
		vm.expectRevert();
		_verification.verifyVaa_U7N5(invalidMultisigVaa);
	}

	function test_verifySchnorrVaa() public {
		_warmSchnorrSlots(0);
		_verification.verifyVaa_U7N5(schnorrVaa);
	}

	function testRevert_verifyInvalidSchnorrVaa() public {
		_warmSchnorrSlots(0);
		vm.expectRevert();
		_verification.verifyVaa_U7N5(invalidSchnorrVaa);
	}

	function test_verifyBigSchnorrVaa() public {
		_warmSchnorrSlots(1);
		_verification.verifyVaa_U7N5(bigSchnorrVaa);
	}

	function test_verifyBigMultisigVaa() public {
		_warmSchnorrSlots(1);
		_verification.verifyVaa_U7N5(bigMultisigVaa);
	}

	function test_verifyHashAndHeader() public {
		_warmSchnorrSlots(1);

		address r2 = 0xD970AcFC9e8ff8BE38b0Fd6C4Fe4DD4DDB744cb4;
		uint256 s2 = 0xfc201908d0a3aec1973711f48365deaa91180ef2771cb3744bccfc3ba77d6c77;
		bytes memory message = abi.encodePacked(
			bytes32(0x15835812e5abde361734ae28a46fa26e44b718a08f75ed2706537328701b8173),
			uint8(0x02),
			uint32(0x00000001),
			address(r2),
			uint256(s2)
		);

		_verification.verifyHashAndHeader(message);
	}

	function testRevert_verifyHashAndHeader_invalid() public {
		_warmSchnorrSlots(1);

		address r2 = 0xD970AcFC9e8ff8BE38b0Fd6C4Fe4DD4DDB744cb4;
		uint256 s2 = 0xfc201908d0a3aec1973711f48365deaa91180ef2771cb3744bccfc3ba77d6c77;

		vm.expectRevert();
		_verification.verifyHashAndHeader(abi.encodePacked(
			bytes32(0x15835812e5abde361734ae28a46fa26e44b718a08f75ed2706537328701b8174),
			uint8(0x02),
			uint32(0x00000001),
			address(r2),
			uint256(s2)
		));
	}

	/*
	function test_updateShardId() public {
		// TODO: Implement
	}
	
	function test_schnorrKeyExpiration() public {
		// TODO: Implement
	}

	function test_multisigKeyExpiration() public {
		// TODO: Implement
	}

	function test_getCurrentMultisigData() public view {
		// TODO: Implement
	}

	function test_getCurrentSchnorrData() public view {
		// TODO: Implement
	}

	function test_getMultisigData() public view {
		// TODO: Implement
	}

	function testRevert_getMultisigData_invalid() public {
		// TODO: Implement
	}

	function test_getSchnorrData() public view {
		// TODO: Implement
	}

	function testRevert_getSchnorrData_invalid() public {
		// TODO: Implement
	}
	*/

	function test_getShardData() public view {
		(bool success, bytes memory resultRaw) = address(_verification).staticcall(abi.encodePacked(
			RawDispatcher.get1959.selector,
			GET_SCHNORR_SHARD_DATA,
			uint32(1)
		));

		assertEq(success, true);
		bytes memory result = abi.decode(resultRaw, (bytes));
		
		for (uint256 i = 0; i < SHARD_COUNT; i++) {
			bytes32 shard;
			bytes32 id;

			assembly ("memory-safe") {
				shard := mload(add(add(0x20, result), shl(6, i)))
				id := mload(add(add(0x40, result), shl(6, i)))
			}

			assertEq(shard, schnorrShards[i].shard);
			assertEq(id, schnorrShards[i].id);
		}
	}

	function testRevert_getShardData_invalid() public {
		(bool success, bytes memory resultRaw) = address(_verification).staticcall(abi.encodePacked(
			RawDispatcher.get1959.selector,
			GET_SCHNORR_SHARD_DATA,
			uint32(1234)
		));

		assertEq(success, true);

		bytes memory result = abi.decode(resultRaw, (bytes));
		assertEq(result.length, 0);
	}
}
