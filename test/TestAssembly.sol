// SPDX-License-Identifier: MIT

pragma solidity ^0.8.27;

import {Test} from "forge-std/Test.sol";

import {CHAIN_ID_SOLANA} from "wormhole-solidity-sdk/constants/Chains.sol";

import {WormholeMock} from "./Test.sol";
import {VaaBuilder, ShardData} from "./TestOptimized.sol";
import {Verification, GOVERNANCE_ADDRESS, UPDATE_PULL_MULTISIG_KEY_DATA, UPDATE_APPEND_SCHNORR_KEY, UPDATE_PULL_MULTISIG_KEY_DATA} from "../src/evm/AssemblyOptimized.sol";

contract TestAssembly is Test, VaaBuilder {
	uint256 private constant guardianPrivateKey1 = 0x1234567890123456789012345678901234567890123456789012345678901234;
	uint256[] private guardianPrivateKeys1 = [guardianPrivateKey1];
	address[] private guardianSet1 = [vm.addr(guardianPrivateKey1)];

	uint256 private constant schnorrKey1 = 0x79380e24c7cbb0f88706dd035135020063aab3e7f403398ff7f995af0b8a770c << 1;
	ShardData[] private schnorrShards1 = [
		ShardData({
			shard: bytes32(0x0000000000000000000000000000000000000000000000000000000000001234),
			id: bytes32(0x0000000000000000000000000000000000000000000000000000000000005678)
		})
	];

	bytes private updatePullOneMultisigKey;
	bytes private updateRegisterSchnorrKey;
	bytes private registerSchnorrKeyVaa;
	bytes private schnorrVaa;

	WormholeMock _wormhole = new WormholeMock(guardianSet1);
	Verification _verification = new Verification(_wormhole, 0, 0, 0);

	function setUp() public {
		updatePullOneMultisigKey = abi.encodePacked(
			UPDATE_PULL_MULTISIG_KEY_DATA,
			uint32(1)
		);

		bytes memory payload = createAppendSchnorrKeyMessage(0, schnorrKey1, 0, schnorrShards1);
		bytes memory envelope = createVaaEnvelope(uint32(block.timestamp), 0, CHAIN_ID_SOLANA, GOVERNANCE_ADDRESS, 0, 0, payload);
		registerSchnorrKeyVaa = createMultisigVaa(0, guardianPrivateKeys1, envelope);
		updateRegisterSchnorrKey = abi.encodePacked(
			UPDATE_APPEND_SCHNORR_KEY,
			registerSchnorrKeyVaa
		);

		address r = address(0x636a8688ef4B82E5A121F7C74D821A5b07d695f3);
		uint256 s = 0xaa6d485b7d7b536442ea7777127d35af43ac539a491c0d85ee0f635eb7745b29;
		schnorrVaa = createSchnorrVaa(0, r, s, new bytes(100));
	}

	function test_verifySchnorrVaa() public {
		_verification.update(updatePullOneMultisigKey);
		_verification.update(updateRegisterSchnorrKey);
		_verification.verifyVaa_U7N5(schnorrVaa);
	}
}
