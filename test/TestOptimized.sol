// SPDX-License-Identifier: MIT
pragma solidity ^0.8.27;

import {Test} from "forge-std/Test.sol";
import {console} from "forge-std/console.sol";

import {WormholeMock} from "./Test.sol";

import {CHAIN_ID_SOLANA} from "wormhole-solidity-sdk/constants/Chains.sol";
import {keccak256Word, keccak256SliceUnchecked} from "wormhole-solidity-sdk/utils/Keccak.sol";

import {
	Verification,
	MODULE_VERIFICATION_V2,
	ACTION_APPEND_SCHNORR_KEY,
	GOVERNANCE_ADDRESS,
	ShardData,
	VerificationError
} from "../src/evm/Optimized.sol";

contract VaaBuilder is Test {
	function createMultisigVaa(uint32 guardianSetIndex, uint256[] memory guardianPrivateKeys, bytes memory envelope) public pure returns (bytes memory) {
		uint256 guardianCount = guardianPrivateKeys.length;
		bytes memory signatures = new bytes(guardianCount * 66); // 66 bytes per signature (1 byte index, 32 bytes r, 32 bytes s, 1 byte v)
		bytes32 vaaDoubleHash = keccak256Word(keccak256SliceUnchecked(envelope, 0, envelope.length));

		for (uint256 i = 0; i < guardianCount; i++) {
			(uint8 v, bytes32 r, bytes32 s) = vm.sign(guardianPrivateKeys[i], vaaDoubleHash);

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

	function createAppendSchnorrKeyMessage(
		uint32 newTSSIndex,
		uint256 newThresholdPubkey,
		uint32 expirationDelaySeconds,
		ShardData[] memory shards
	) public pure returns (bytes memory) {
		bytes32[] memory shardsData = new bytes32[](shards.length * 2);
		for (uint256 i = 0; i < shards.length; i++) {
			shardsData[i * 2] = shards[i].shard;
			shardsData[i * 2 + 1] = shards[i].id;
		}

		return abi.encodePacked(
			MODULE_VERIFICATION_V2,
			ACTION_APPEND_SCHNORR_KEY,
			newTSSIndex,
			newThresholdPubkey,
			expirationDelaySeconds,
			shardsData
		);
	}
}

contract VerificationTests is Test, VaaBuilder {
	uint256 private constant guardianPrivateKey1 = 0x0123456701234567012345670123456701234567012345670123456701234567;
	uint256[] private guardianPrivateKeys1 = [guardianPrivateKey1];
	address private guardianPublicKey1 = vm.addr(guardianPrivateKey1);
	address[] private guardianKeys1 = [guardianPublicKey1];

	uint256 private constant schnorrKey1 = 0x79380e24c7cbb0f88706dd035135020063aab3e7f403398ff7f995af0b8a770c << 1;
	ShardData[] private schnorrShards1 = [
		ShardData({
			shard: bytes32(0x0000000000000000000000000000000000000000000000000000000000001234),
			id: bytes32(0x0000000000000000000000000000000000000000000000000000000000005678)
		})
	];

	bytes private registerSchnorrKeyVaa;
	bytes private schnorrVaa;

	WormholeMock public wormholeMock = new WormholeMock(guardianKeys1);

	Verification public verification = new Verification(wormholeMock, 0, 1);

	function setUp() public {
		bytes memory payload = createAppendSchnorrKeyMessage(0, schnorrKey1, 0, schnorrShards1);
		bytes memory envelope = createVaaEnvelope(uint32(block.timestamp), 0, CHAIN_ID_SOLANA, GOVERNANCE_ADDRESS, 0, 0, payload);
		registerSchnorrKeyVaa = createMultisigVaa(0, guardianPrivateKeys1, envelope);

		address r = address(0x636a8688ef4B82E5A121F7C74D821A5b07d695f3);
		uint256 s = 0xaa6d485b7d7b536442ea7777127d35af43ac539a491c0d85ee0f635eb7745b29;
		schnorrVaa = createVaaV2(0, r, s, new bytes(100));
	}

	function test_verifyVaaSchnorr() public {
		bytes memory registerSchnorrKeyMessage = abi.encodePacked(
			uint8(1),
			uint16(registerSchnorrKeyVaa.length),
			registerSchnorrKeyVaa
		);

		verification.update(registerSchnorrKeyMessage);

		verification.verifyVaa_U7N5(schnorrVaa);
	}

	function test_verifyVaaSchnorrVaaEssentials() public {
		bytes memory registerSchnorrKeyMessage = abi.encodePacked(
			uint8(1),
			uint16(registerSchnorrKeyVaa.length),
			registerSchnorrKeyVaa
		);

		verification.update(registerSchnorrKeyMessage);

		(uint16 emitterChainId, bytes32 emitterAddress, uint32 sequence, bytes memory payload) = verification.verifyVaaDecodeEssentials_gRd6(schnorrVaa);
		assertEq(emitterChainId, 0);
		assertEq(emitterAddress, bytes32(0));
		assertEq(sequence, 0);
		assertEq(payload.length, 49);
	}
}
