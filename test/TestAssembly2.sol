// SPDX-License-Identifier: MIT

pragma solidity ^0.8.27;

import {console} from "forge-std/console.sol";
import {Test} from "forge-std/Test.sol";

import {CHAIN_ID_SOLANA, CHAIN_ID_ETHEREUM} from "wormhole-solidity-sdk/constants/Chains.sol";
import {keccak256Word, keccak256SliceUnchecked} from "wormhole-solidity-sdk/utils/Keccak.sol";
import {ICoreBridge, CoreBridgeVM, GuardianSet} from "wormhole-solidity-sdk/interfaces/ICoreBridge.sol";
import {VaaLib} from "wormhole-solidity-sdk/libraries/VaaLib.sol";

import {WormholeVerifier, MODULE_VERIFICATION_V2, ACTION_APPEND_SCHNORR_KEY, GOVERNANCE_ADDRESS, UPDATE_PULL_MULTISIG_KEY_DATA, UPDATE_APPEND_SCHNORR_KEY} from "../src/evm/AssemblyOptimized2.sol";

struct ShardData {
  bytes32 shard;
  bytes32 id;
}

abstract contract VaaBuilder {
  function newAppendSchnorrKeyMessage(
    uint32 newTSSIndex,
    uint256 newThresholdPubkey,
    uint32 expirationDelaySeconds,
    bytes32 initialShardDataHash
  ) internal pure returns (bytes memory) {
    return abi.encodePacked(
      MODULE_VERIFICATION_V2,
      ACTION_APPEND_SCHNORR_KEY,
      newTSSIndex,
      newThresholdPubkey,
      expirationDelaySeconds,
      initialShardDataHash
    );
  }

  // Convert the shard data to a raw bytes array
  // NOTE: This will destroy the original shard data array's length
  function shardDataToBytes(ShardData[] memory shards) internal pure returns (bytes memory result) {
    assembly ("memory-safe") {
      mstore(result, shl(6, mload(shards)))
    }
  }

  function newVaaEnvelope(
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

  function newMultisigVaa(uint32 keyIndex, bytes memory signatures, bytes memory envelope) public pure returns (bytes memory) {
    uint8 signatureCount = uint8(signatures.length / 66);

    return abi.encodePacked(
      uint8(1), // version
      keyIndex,
      signatureCount,
      signatures,
      envelope
    );
  }

  function newSchnorrVaa(
    uint32 keyIndex,
    address r,
    uint256 s,
    bytes memory envelope
  ) public pure returns (bytes memory) {
    return abi.encodePacked(
      uint8(2), // version
      keyIndex,
      r,
      s,
      envelope
    );
  }
}

abstract contract VerificationTestAPI is Test, VaaBuilder {
  function newKeySet(uint256 count) internal returns (uint256[] memory privateKeys, address[] memory publicKeys) {
    privateKeys = new uint256[](count);
    publicKeys = new address[](count);

    for (uint256 i = 0; i < count; i++) {
      uint256 privateKey = vm.randomUint();
      privateKeys[i] = privateKey;
      publicKeys[i] = vm.addr(privateKey);
    }
  }

  function getEnvelopeDigest(bytes memory envelope) internal pure returns (bytes32) {
    return keccak256Word(keccak256SliceUnchecked(envelope, 0, envelope.length));
  }

  function signMultisig(bytes32 digest, uint256[] memory privateKeys) internal pure returns (bytes memory signatures) {
    signatures = new bytes(privateKeys.length * 66);

    for (uint256 i = 0; i < privateKeys.length; i++) {
      (uint8 v, bytes32 r, bytes32 s) = vm.sign(privateKeys[i], digest);

      assembly ("memory-safe") {
        let offset := add(add(signatures, 32), mul(i, 66))
        mstore8(offset, i)
        mstore(add(offset, 1), r)
        mstore(add(offset, 33), s)	
        mstore8(add(offset, 65), eq(v, 28))
      }
    }
  }

  function signMultisig(bytes memory envelope, uint256[] memory privateKeys) internal pure returns (bytes memory signatures) {
    return signMultisig(getEnvelopeDigest(envelope), privateKeys);
  }
}

contract WormholeV1Mock is ICoreBridge {
  GuardianSet[] private _guardianSets;

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

contract TestAssembly2 is VerificationTestAPI {
  using VaaLib for bytes;

  uint32 private constant EXPIRATION_DELAY_SECONDS = 24 * 60 * 60;

  uint256 private constant SHARD_COUNT = 1;
  uint256 private constant SHARD_QUORUM = 1;

  uint256[] private guardianPrivateKeys;
  address[] private guardianPublicKeys;

  bytes private smallMultisigVaa;
  bytes private bigMultisigVaa;
  bytes private smallSchnorrVaa;
  bytes private bigSchnorrVaa;

  bytes private invalidVersionVaa;
  bytes private invalidMultisigVaa;
  bytes private invalidSchnorrVaa;

  WormholeV1Mock private immutable _wormholeV1Mock = new WormholeV1Mock();
  WormholeVerifier private immutable _wormholeVerifierV2 = new WormholeVerifier(_wormholeV1Mock, 0, 0, 0);

  function setUp() public {
    // Generate the initial guardian set
    (guardianPrivateKeys, guardianPublicKeys) = newKeySet(SHARD_COUNT);

    // Append the initial guardian set to the wormholeV1Mock
    _wormholeV1Mock.appendGuardianSet(GuardianSet({
      expirationTime: uint32(block.timestamp + EXPIRATION_DELAY_SECONDS),
      keys: guardianPublicKeys
    }));

    // Create a slice of the guardian private keys for the multisig to hit quorum without wasting gas
    uint256[] memory guardianPrivateKeysSlice = new uint256[](SHARD_QUORUM);
    for (uint256 i = 0; i < SHARD_QUORUM; i++) {
      guardianPrivateKeysSlice[i] = guardianPrivateKeys[i];
    }

    // Create VAAs
    bytes memory smallEnvelope = new bytes(100);
    bytes memory smallMultisigSignatures = signMultisig(smallEnvelope, guardianPrivateKeysSlice);
    smallMultisigVaa = newMultisigVaa(0, smallMultisigSignatures, smallEnvelope);

    uint256 pk1 = 0x79380e24c7cbb0f88706dd035135020063aab3e7f403398ff7f995af0b8a770c << 1;
    address r1 = address(0x636a8688ef4B82E5A121F7C74D821A5b07d695f3);
    uint256 s1 = 0xaa6d485b7d7b536442ea7777127d35af43ac539a491c0d85ee0f635eb7745b29;
    smallSchnorrVaa = newSchnorrVaa(0, r1, s1, smallEnvelope);

    bytes memory bigEnvelope = new bytes(5000);
    bytes memory bigMultisigSignatures = signMultisig(bigEnvelope, guardianPrivateKeysSlice);
    bigMultisigVaa = newMultisigVaa(0, bigMultisigSignatures, bigEnvelope);

    uint256 pk2 = 0x44c90dfbe2a454987a65ce9e6f522c9c5c9d1dfb3c3aaaadcd0ae4f5366a2922 << 1;
    address r2 = 0xD970AcFC9e8ff8BE38b0Fd6C4Fe4DD4DDB744cb4;
    uint256 s2 = 0xfc201908d0a3aec1973711f48365deaa91180ef2771cb3744bccfc3ba77d6c77;
    bigSchnorrVaa = newSchnorrVaa(1, r2, s2, bigEnvelope);

    invalidVersionVaa = new bytes(100);

    invalidMultisigVaa = new bytes(100);
    invalidMultisigVaa[0] = 0x01;

    invalidSchnorrVaa = new bytes(100);
    invalidSchnorrVaa[0] = 0x02;

    // Geneate shard data
    ShardData[] memory schnorrShards = new ShardData[](SHARD_COUNT);
    for (uint256 i = 0; i < SHARD_COUNT; i++) {
      schnorrShards[i] = ShardData({
        shard: bytes32(vm.randomUint()),
        id: bytes32(vm.randomUint())
      });
    }

    bytes memory schnorrShardsRaw = shardDataToBytes(schnorrShards);
    require(schnorrShardsRaw.length == SHARD_COUNT*64);

    bytes32 schnorrShardDataHash = keccak256(schnorrShardsRaw);
    bytes memory appendSchnorrKeyMessage1 = newAppendSchnorrKeyMessage(0, pk1, 0, schnorrShardDataHash);
    bytes memory appendSchnorrKeyEnvelope1 = newVaaEnvelope(uint32(block.timestamp), 0, CHAIN_ID_SOLANA, GOVERNANCE_ADDRESS, 0, 0, appendSchnorrKeyMessage1);
    bytes memory appendSchnorrKeyVaa1 = newMultisigVaa(0, signMultisig(appendSchnorrKeyEnvelope1, guardianPrivateKeys), appendSchnorrKeyEnvelope1);

    bytes memory appendSchnorrKeyMessage2 = newAppendSchnorrKeyMessage(1, pk2, EXPIRATION_DELAY_SECONDS, schnorrShardDataHash);
    bytes memory appendSchnorrKeyEnvelope2 = newVaaEnvelope(uint32(block.timestamp), 0, CHAIN_ID_SOLANA, GOVERNANCE_ADDRESS, 0, 0, appendSchnorrKeyMessage2);
    bytes memory appendSchnorrKeyVaa2 = newMultisigVaa(0, signMultisig(appendSchnorrKeyEnvelope2, guardianPrivateKeys), appendSchnorrKeyEnvelope2);

    bytes memory message = abi.encodePacked(
      UPDATE_PULL_MULTISIG_KEY_DATA,
      uint32(1),
      UPDATE_APPEND_SCHNORR_KEY,
      appendSchnorrKeyVaa1,
      schnorrShardsRaw,
      UPDATE_APPEND_SCHNORR_KEY,
      appendSchnorrKeyVaa2,
      schnorrShardsRaw
    );

    _wormholeVerifierV2.update(message);
  }

  function test_verifyMultisig() public view {
    (uint16 emitterChain, bytes32 emitterAddress, uint64 sequence, uint16 payloadOffset) = _wormholeVerifierV2.verify(smallMultisigVaa);
    vm.assertEq(emitterChain, 0);
    vm.assertEq(emitterAddress, bytes32(0));
    vm.assertEq(sequence, 0);
    vm.assertEq(payloadOffset, 1 + 4 + 1 + 66*SHARD_QUORUM + 4 + 4 + 2 + 32 + 8 + 1);
  }

  function test_verifySchnorr() public view {
    (uint16 emitterChain, bytes32 emitterAddress, uint64 sequence, uint16 payloadOffset) = _wormholeVerifierV2.verify(smallSchnorrVaa);
    vm.assertEq(emitterChain, 0);
    vm.assertEq(emitterAddress, bytes32(0));
    vm.assertEq(sequence, 0);
    vm.assertEq(payloadOffset, 1 + 4 + 20 + 32 + 4 + 4 + 2 + 32 + 8 + 1);
  }

  function test_verifyMultisigBig() public view {
    (uint16 emitterChain, bytes32 emitterAddress, uint64 sequence, uint16 payloadOffset) = _wormholeVerifierV2.verify(bigMultisigVaa);
    vm.assertEq(emitterChain, 0);
    vm.assertEq(emitterAddress, bytes32(0));
    vm.assertEq(sequence, 0);
    vm.assertEq(payloadOffset, 1 + 4 + 1 + 66*SHARD_QUORUM + 4 + 4 + 2 + 32 + 8 + 1);
  }

  function test_verifySchnorrBig() public view {
    (uint16 emitterChain, bytes32 emitterAddress, uint64 sequence, uint16 payloadOffset) = _wormholeVerifierV2.verify(bigSchnorrVaa);
    vm.assertEq(emitterChain, 0);
    vm.assertEq(emitterAddress, bytes32(0));
    vm.assertEq(sequence, 0);
    vm.assertEq(payloadOffset, 1 + 4 + 20 + 32 + 4 + 4 + 2 + 32 + 8 + 1);
  }
}
