// SPDX-License-Identifier: MIT

pragma solidity ^0.8.27;

import {console} from "forge-std/console.sol";
import {Test} from "forge-std/Test.sol";

import {CHAIN_ID_SOLANA, CHAIN_ID_ETHEREUM} from "wormhole-solidity-sdk/constants/Chains.sol";
import {keccak256Word, keccak256SliceUnchecked} from "wormhole-solidity-sdk/utils/Keccak.sol";
import {ICoreBridge, CoreBridgeVM, GuardianSet, GuardianSignature} from "wormhole-solidity-sdk/interfaces/ICoreBridge.sol";
import {VaaLib} from "wormhole-solidity-sdk/libraries/VaaLib.sol";
import {CoreBridgeLib} from "wormhole-solidity-sdk/libraries/CoreBridge.sol";
import {BytesParsing} from "wormhole-solidity-sdk/libraries/BytesParsing.sol";

import {EIP712Encoding} from "../src/evm/EIP712Encoding.sol";

import {
  WormholeVerifier,

  MODULE_VERIFICATION_V2,
  ACTION_APPEND_SCHNORR_KEY,
  GOVERNANCE_ADDRESS,

  UPDATE_PULL_MULTISIG_KEY_DATA,
  UPDATE_APPEND_SCHNORR_KEY,
  UPDATE_SET_SHARD_ID,

  MASK_UPDATE_RESULT_INVALID_SCHNORR_KEY_INDEX,
  MASK_UPDATE_RESULT_NONCE_ALREADY_CONSUMED,
  MASK_UPDATE_RESULT_INVALID_SIGNER_INDEX,
  MASK_UPDATE_RESULT_SIGNATURE_MISMATCH,
  MASK_UPDATE_RESULT_INVALID_KEY_INDEX,
  MASK_UPDATE_RESULT_INVALID_SCHNORR_KEY,
  MASK_UPDATE_RESULT_SHARD_DATA_MISMATCH,

  MASK_VERIFY_RESULT_INVALID_VERSION,
  MASK_VERIFY_RESULT_SIGNATURE_MISMATCH,
  MASK_VERIFY_RESULT_INVALID_SIGNATURE_COUNT,
  MASK_VERIFY_RESULT_INVALID_SIGNATURE,
  MASK_VERIFY_RESULT_INVALID_KEY_DATA_SIZE,
  MASK_VERIFY_RESULT_INVALID_KEY,

  VERIFY_ANY,
  VERIFY_MULTISIG,
  VERIFY_MULTISIG_UNIFORM,
  VERIFY_SCHNORR,
  VERIFY_SCHNORR_UNIFORM,

  GET_CURRENT_SCHNORR_KEY_DATA,
  GET_CURRENT_MULTISIG_KEY_DATA,
  GET_SCHNORR_KEY_DATA,
  GET_MULTISIG_KEY_DATA,
  GET_SCHNORR_SHARD_DATA
} from "../src/evm/WormholeVerifier.sol";


uint256 constant LENGTH_WORD = 0x20;

struct ShardData {
  bytes32 shard;
  bytes32 id;
}

abstract contract VerificationMessageBuilder {
  function newAppendSchnorrKeyMessage(
    uint32 newSchnorrKeyIndex,
    uint32 expectedMultisigKeyIndex,
    uint256 newSchnorrPubkey,
    uint32 expirationDelaySeconds,
    bytes32 initialShardDataHash
  ) internal pure returns (bytes memory) {
    return abi.encodePacked(
      MODULE_VERIFICATION_V2,
      ACTION_APPEND_SCHNORR_KEY,
      newSchnorrKeyIndex,
      expectedMultisigKeyIndex,
      newSchnorrPubkey,
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

abstract contract VerificationTestAPI is Test, VerificationMessageBuilder {
  using BytesParsing for bytes;

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

  function signUpdateShardIdMessage(
    WormholeVerifier wormholeVerifier,
    uint32 keyIndex,
    uint32 nonce,
    bytes32 shardId,
    uint8 signerIndex,
    uint256 privateKey
  ) internal view returns (bytes memory signedMessage) {
    bytes32 digest = wormholeVerifier.getRegisterGuardianDigest(keyIndex, nonce, shardId);

    (uint8 v, bytes32 r, bytes32 s) = vm.sign(privateKey, digest);

    return abi.encodePacked(
      keyIndex,
      nonce,
      shardId,
      signerIndex,
      r,
      s,
      v - 27
    );
  }

  function pullGuardianSets(WormholeVerifier verifier, uint32 limit) public {
    bytes memory message = abi.encodePacked(
      UPDATE_PULL_MULTISIG_KEY_DATA,
      uint32(limit)
    );

    verifier.update(message);
  }

  function appendSchnorrKey(
    WormholeVerifier verifier,
    bytes memory appendVaa,
    bytes memory shards
  ) public {
    bytes memory message = abi.encodePacked(
      UPDATE_APPEND_SCHNORR_KEY,
      appendVaa,
      shards
    );

    verifier.update(message);
  }

  function getCurrentGuardianSet() public pure returns (bytes memory) {
    return abi.encodePacked(
      GET_CURRENT_MULTISIG_KEY_DATA
    );
  }

  function decodeGetCurrentGuardianSet(bytes memory result, uint256 offset) public pure returns (
    address[] memory guardianSetAddrs,
    uint32 guardianSetIndex,
    uint256 newOffset
  ) {
    uint8 guardianCount;
    (guardianSetIndex, newOffset) = result.asUint32MemUnchecked(offset);
    (guardianCount, newOffset) = result.asUint8MemUnchecked(newOffset);

    guardianSetAddrs = new address[](guardianCount);
    for (uint256 i = 0; i < guardianCount; i++) {
      (guardianSetAddrs[i], newOffset) = result.asAddressMemUnchecked(newOffset + 12);
    }
  }

  function getGuardianSet(uint32 index) public pure returns (bytes memory) {
    return abi.encodePacked(
      GET_MULTISIG_KEY_DATA,
      uint32(index)
    );
  }

  function decodeGetGuardianSet(bytes memory result, uint256 offset) public pure returns (
    address[] memory guardianSetAddrs,
    uint32 expirationTime,
    uint256 newOffset
  ) {
    uint8 guardianCount;
    (guardianCount, newOffset) = result.asUint8MemUnchecked(offset);

    guardianSetAddrs = new address[](guardianCount);
    for (uint256 i = 0; i < guardianCount; i++) {
      (guardianSetAddrs[i], newOffset) = result.asAddressMemUnchecked(newOffset + 12);
    }

    (expirationTime, newOffset) = result.asUint32MemUnchecked(newOffset);
  }

  function getCurrentSchnorrKey() public pure returns (bytes memory) {
    return abi.encodePacked(
      GET_CURRENT_SCHNORR_KEY_DATA
    );
  }

  function decodeGetCurrentSchnorrKey(bytes memory result, uint256 offset) public pure returns (
    uint32  schnorrKeyIndex,
    uint256 schnorrKeyPubkey,
    uint32  expirationTime,
    uint8   shardCount,
    uint32  guardianSet,
    uint256 newOffset
  ) {
    (schnorrKeyIndex,  newOffset) = result.asUint32MemUnchecked(offset);
    (schnorrKeyPubkey, newOffset) = result.asUint256MemUnchecked(newOffset);
    (expirationTime,   newOffset) = result.asUint32MemUnchecked(newOffset);
    (shardCount,       newOffset) = result.asUint8MemUnchecked(newOffset);
    (guardianSet,      newOffset) = result.asUint32MemUnchecked(newOffset);
    assertGe(result.length, newOffset);
  }


  function getSchnorrKey(uint32 index) public pure returns (bytes memory) {
    return abi.encodePacked(
      GET_SCHNORR_KEY_DATA,
      uint32(index)
    );
  }

  function decodeGetSchnorrKey(bytes memory result, uint256 offset) public pure returns (
    uint256 schnorrKeyPubkey,
    uint32  expirationTime,
    uint8   shardCount,
    uint32  guardianSet,
    uint256 newOffset
  ) {
    (schnorrKeyPubkey, newOffset) = result.asUint256MemUnchecked(offset);
    (expirationTime,   newOffset) = result.asUint32MemUnchecked(newOffset);
    (shardCount,       newOffset) = result.asUint8MemUnchecked(newOffset);
    (guardianSet,      newOffset) = result.asUint32MemUnchecked(newOffset);
    assertGe(result.length, newOffset);
  }

  function getShardData(uint32 index) public pure returns (bytes memory) {
    return abi.encodePacked(
      GET_SCHNORR_SHARD_DATA,
      uint32(index)
    );
  }

  function decodeShardData(bytes memory result, uint256 offset) public pure returns (
    ShardData[] memory shardData,
    uint256     newOffset
  ) {
    uint256 shards;
    (shards, newOffset) = result.asUint8MemUnchecked(offset);
    shardData = new ShardData[](shards);

    for (uint i = 0; i < shards; ++i) {
      bytes32 shard;
      bytes32 id;
      (shard, newOffset) = result.asBytes32MemUnchecked(newOffset);
      (id,    newOffset) = result.asBytes32MemUnchecked(newOffset);
      shardData[i].shard = shard;
      shardData[i].id = id;
    }
    assertGe(result.length, newOffset);
  }
}

contract WormholeV1MockVerification {
  using BytesParsing for bytes;
  WormholeV1Mock core;

  constructor(WormholeV1Mock initCore) {
    core = initCore;
  }

  function parseAndVerifyVM(bytes calldata vaa) external view returns (
    CoreBridgeVM memory,
    bool,
    string memory
  ) {
    (
      uint32  timestamp,
      uint32  nonce,
      uint16  emitterChainId,
      bytes32 emitterAddress,
      uint64  sequence,
      uint8   consistencyLevel,
      bytes calldata payload
    ) = CoreBridgeLib.decodeAndVerifyVaaCd(address(core), vaa);

    (uint32 guardianSet,) = vaa.asUint32CdUnchecked(1);

    CoreBridgeVM memory result = CoreBridgeVM({
      version: 1,
      timestamp: timestamp,
      nonce: nonce,
      emitterChainId: emitterChainId,
      emitterAddress: emitterAddress,
      sequence: sequence,
      consistencyLevel: consistencyLevel,
      payload: payload,
      guardianSetIndex: guardianSet,
      signatures: new GuardianSignature[](0),
      hash: 0x0
    });
    return (result, true, "");
  }
}

contract WormholeV1Mock is ICoreBridge {
  GuardianSet[] private _guardianSets;
  WormholeV1MockVerification wrapVerify;

  constructor() {
    wrapVerify = new WormholeV1MockVerification(this);
  }

  function messageFee() external pure returns (uint256) {
    revert("Not implemented");
  }

  function publishMessage(uint32, bytes memory, uint8) external payable returns (uint64) {
    revert("Not implemented");
  }

  function parseAndVerifyVM(bytes calldata vaa) external view returns (
    CoreBridgeVM memory result,
    bool valid,
    string memory reason
  ) {
    try wrapVerify.parseAndVerifyVM(vaa) returns (CoreBridgeVM memory result2, bool valid2, string memory reason2) {
      result = result2;
      valid = valid2;
      reason = reason2;
    }
    catch {
      valid = false;
      reason = "Verification failed";
    }
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

contract TestAssembly2Benchmark is VerificationTestAPI {
  using VaaLib for bytes;
  using BytesParsing for bytes;

  uint32 private constant EXPIRATION_DELAY_SECONDS = 24 * 60 * 60;

  uint256 private constant SHARD_COUNT = 19;
  uint256 private constant SHARD_QUORUM = 13;

  uint256[] private guardianPrivateKeys;
  address[] private guardianPublicKeys;

  uint256[] private schnorrPublicKeys;

  bytes private smallMultisigVaa;
  bytes private bigMultisigVaa;
  bytes private smallSchnorrVaa;
  bytes private bigSchnorrVaa;

  bytes private invalidVersionVaa;
  bytes private invalidMultisigVaa;
  bytes private invalidSchnorrVaa;

  bytes private batchMessage;
  bytes private batchMultisigMessage;
  bytes private batchSchnorrMessage;
  bytes private batchMultisigUniformMessage;
  bytes private batchSchnorrUniformMessage;

  WormholeV1Mock private immutable _wormholeV1Mock = new WormholeV1Mock();
  WormholeVerifier private immutable _wormholeVerifierV2 = new WormholeVerifier(_wormholeV1Mock, 0, 0, new bytes(0));

  function setUpMessages1(bytes memory smallEnvelope, bytes memory bigEnvelope, uint256[] memory guardianPrivateKeysSlice) internal {
    bytes memory smallMultisigSignatures = signMultisig(smallEnvelope, guardianPrivateKeysSlice);
    smallMultisigVaa = newMultisigVaa(0, smallMultisigSignatures, smallEnvelope);

    uint256 pk1 = 0x79380e24c7cbb0f88706dd035135020063aab3e7f403398ff7f995af0b8a770c << 1;
    address r1 = address(0x636a8688ef4B82E5A121F7C74D821A5b07d695f3);
    uint256 s1 = 0xaa6d485b7d7b536442ea7777127d35af43ac539a491c0d85ee0f635eb7745b29;
    smallSchnorrVaa = newSchnorrVaa(0, r1, s1, smallEnvelope);

    bytes memory bigMultisigSignatures = signMultisig(bigEnvelope, guardianPrivateKeysSlice);
    bigMultisigVaa = newMultisigVaa(0, bigMultisigSignatures, bigEnvelope);

    uint256 pk2 = 0x44c90dfbe2a454987a65ce9e6f522c9c5c9d1dfb3c3aaaadcd0ae4f5366a2922 << 1;
    address r2 = 0xD970AcFC9e8ff8BE38b0Fd6C4Fe4DD4DDB744cb4;
    uint256 s2 = 0xfc201908d0a3aec1973711f48365deaa91180ef2771cb3744bccfc3ba77d6c77;
    bigSchnorrVaa = newSchnorrVaa(1, r2, s2, bigEnvelope);

    schnorrPublicKeys = new uint256[](2);
    schnorrPublicKeys[0] = pk1;
    schnorrPublicKeys[1] = pk2;

    invalidVersionVaa = new bytes(100);

    invalidMultisigVaa = new bytes(100);
    invalidMultisigVaa[0] = 0x01;

    invalidSchnorrVaa = new bytes(100);
    invalidSchnorrVaa[0] = 0x02;

    uint256 multisigVaaHeaderLength2 = 4+1+66*SHARD_QUORUM;
    uint256 schnorrVaaHeaderLength2 = 4+20+32;

    bytes memory smallMultisigVaaHeader2;
    (smallMultisigVaaHeader2,) = smallMultisigVaa.sliceMemUnchecked(1, multisigVaaHeaderLength2);
    bytes memory smallSchnorrVaaHeader2 = new bytes(schnorrVaaHeaderLength2);
    bytes memory bigMultisigVaaHeader2;
    (bigMultisigVaaHeader2,) = bigMultisigVaa.sliceMemUnchecked(1, multisigVaaHeaderLength2);
    bytes memory bigSchnorrVaaHeader2 = new bytes(schnorrVaaHeaderLength2);

    for (uint256 i = 0; i < schnorrVaaHeaderLength2; i++) {
      smallSchnorrVaaHeader2[i] = smallSchnorrVaa[i];
      bigSchnorrVaaHeader2[i] = bigSchnorrVaa[i];
    }

    batchMultisigMessage = abi.encodePacked(
      WormholeVerifier.verifyBatch.selector,
      VERIFY_MULTISIG,
      smallMultisigVaaHeader2,
      getEnvelopeDigest(smallEnvelope),
      bigMultisigVaaHeader2,
      getEnvelopeDigest(bigEnvelope),
      bigMultisigVaaHeader2,
      getEnvelopeDigest(bigEnvelope),
      bigMultisigVaaHeader2,
      getEnvelopeDigest(bigEnvelope)
    );

    batchSchnorrMessage = abi.encodePacked(
      WormholeVerifier.verifyBatch.selector,
      VERIFY_SCHNORR,
      smallSchnorrVaaHeader2,
      getEnvelopeDigest(smallEnvelope),
      bigSchnorrVaaHeader2,
      getEnvelopeDigest(bigEnvelope),
      bigSchnorrVaaHeader2,
      getEnvelopeDigest(bigEnvelope),
      bigSchnorrVaaHeader2,
      getEnvelopeDigest(bigEnvelope)
    );

    uint256 multisigVaaHeaderLength3 = 1+66*SHARD_QUORUM;
    uint256 schnorrVaaHeaderLength3 = 20+32;

    bytes memory smallMultisigVaaHeader3 = new bytes(multisigVaaHeaderLength3);
    bytes memory smallSchnorrVaaHeader3 = new bytes(schnorrVaaHeaderLength3);

    for (uint256 i = 0; i < multisigVaaHeaderLength3; i++) {
      smallMultisigVaaHeader3[i] = smallMultisigVaa[i];
    }

    for (uint256 i = 0; i < schnorrVaaHeaderLength3; i++) {
      smallSchnorrVaaHeader3[i] = smallSchnorrVaa[i];
    }

    batchMultisigUniformMessage = abi.encodePacked(
      WormholeVerifier.verifyBatch.selector,
      VERIFY_MULTISIG_UNIFORM,
      uint32(0),
      smallMultisigVaaHeader3,
      getEnvelopeDigest(smallEnvelope),
      smallMultisigVaaHeader3,
      getEnvelopeDigest(smallEnvelope)
    );

    batchSchnorrUniformMessage = abi.encodePacked(
      WormholeVerifier.verifyBatch.selector,
      VERIFY_SCHNORR_UNIFORM,
      uint32(0),
      smallSchnorrVaaHeader3,
      getEnvelopeDigest(smallEnvelope),
      smallSchnorrVaaHeader3,
      getEnvelopeDigest(smallEnvelope)
    );
  }

  function setUpMessages2(bytes memory smallEnvelope, bytes memory bigEnvelope) internal {
    uint256 multisigVaaHeaderLength = 1+4+1+66*SHARD_QUORUM;
    bytes memory smallMultisigVaaHeader = new bytes(multisigVaaHeaderLength);
    bytes memory bigMultisigVaaHeader = new bytes(multisigVaaHeaderLength);

    for (uint256 i = 0; i < multisigVaaHeaderLength; i++) {
      smallMultisigVaaHeader[i] = smallMultisigVaa[i];
      bigMultisigVaaHeader[i] = bigMultisigVaa[i];
    }

    uint256 schnorrVaaHeaderLength = 1+4+20+32;
    bytes memory smallSchnorrVaaHeader = new bytes(schnorrVaaHeaderLength);
    bytes memory bigSchnorrVaaHeader = new bytes(schnorrVaaHeaderLength);

    for (uint256 i = 0; i < schnorrVaaHeaderLength; i++) {
      smallSchnorrVaaHeader[i] = smallSchnorrVaa[i];
      bigSchnorrVaaHeader[i] = bigSchnorrVaa[i];
    }

    batchMessage = abi.encodePacked(
      VERIFY_ANY,
      smallMultisigVaaHeader,
      getEnvelopeDigest(smallEnvelope),
      smallSchnorrVaaHeader,
      getEnvelopeDigest(smallEnvelope),
      bigMultisigVaaHeader,
      getEnvelopeDigest(bigEnvelope),
      bigSchnorrVaaHeader,
      getEnvelopeDigest(bigEnvelope)
    );
  }

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

    bytes memory smallEnvelope = new bytes(100);
    bytes memory bigEnvelope = new bytes(5000);

    // Create VAAs
    setUpMessages1(smallEnvelope, bigEnvelope, guardianPrivateKeysSlice);
    setUpMessages2(smallEnvelope, bigEnvelope);

    // Generate shard data
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
    bytes memory appendSchnorrKeyMessage1 = newAppendSchnorrKeyMessage(0, 0, schnorrPublicKeys[0], 0, schnorrShardDataHash);
    bytes memory appendSchnorrKeyEnvelope1 = newVaaEnvelope(uint32(block.timestamp), 0, CHAIN_ID_SOLANA, GOVERNANCE_ADDRESS, 0, 0, appendSchnorrKeyMessage1);
    bytes memory appendSchnorrKeyVaa1 = newMultisigVaa(0, signMultisig(appendSchnorrKeyEnvelope1, guardianPrivateKeys), appendSchnorrKeyEnvelope1);

    bytes memory appendSchnorrKeyMessage2 = newAppendSchnorrKeyMessage(1, 0, schnorrPublicKeys[1], EXPIRATION_DELAY_SECONDS, schnorrShardDataHash);
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

  function test_updateShardId_success() public {
    bytes32 id = bytes32(vm.randomUint());
    bytes memory signedMessage = signUpdateShardIdMessage(_wormholeVerifierV2, 1, 1, id, 0, guardianPrivateKeys[0]);
    _wormholeVerifierV2.update(abi.encodePacked(UPDATE_SET_SHARD_ID, signedMessage));
  }

  function test_updateShardIdInvalidKeyIndex() public {
    bytes32 id = bytes32(vm.randomUint());
    bytes memory signedMessage = signUpdateShardIdMessage(_wormholeVerifierV2, 2, 1, id, 0, guardianPrivateKeys[0]);
    vm.expectRevert(abi.encodeWithSelector(WormholeVerifier.UpdateFailed.selector, MASK_UPDATE_RESULT_INVALID_SCHNORR_KEY_INDEX | 1));
    _wormholeVerifierV2.update(abi.encodePacked(UPDATE_SET_SHARD_ID, signedMessage));
  }

  function test_updateShardIdInvalidNonce() public {
    bytes32 id = bytes32(vm.randomUint());
    bytes memory signedMessage = signUpdateShardIdMessage(_wormholeVerifierV2, 1, 1, id, 0, guardianPrivateKeys[0]);
    vm.expectRevert(abi.encodeWithSelector(WormholeVerifier.UpdateFailed.selector, MASK_UPDATE_RESULT_NONCE_ALREADY_CONSUMED | 0x6C));
    _wormholeVerifierV2.update(abi.encodePacked(UPDATE_SET_SHARD_ID, signedMessage, UPDATE_SET_SHARD_ID, signedMessage));
  }

  function test_updateShardIdInvalidSignerIndex() public {
    bytes32 id = bytes32(vm.randomUint());
    bytes memory signedMessage = signUpdateShardIdMessage(_wormholeVerifierV2, 1, 1, id, 0xFF, guardianPrivateKeys[0]);
    vm.expectRevert(abi.encodeWithSelector(WormholeVerifier.UpdateFailed.selector, MASK_UPDATE_RESULT_INVALID_SIGNER_INDEX | 1));
    _wormholeVerifierV2.update(abi.encodePacked(UPDATE_SET_SHARD_ID, signedMessage));
  }

  function test_updateShardIdInvalidSignature() public {
    bytes32 id = bytes32(vm.randomUint());
    bytes memory signedMessage = signUpdateShardIdMessage(_wormholeVerifierV2, 1, 1, id, 0, guardianPrivateKeys[1]);
    vm.expectRevert(abi.encodeWithSelector(WormholeVerifier.UpdateFailed.selector, MASK_UPDATE_RESULT_SIGNATURE_MISMATCH | 1));
    _wormholeVerifierV2.update(abi.encodePacked(UPDATE_SET_SHARD_ID, signedMessage));
  }

  function test_benchmark_verifyMultisig() public view {
    (uint16 emitterChain, bytes32 emitterAddress, uint64 sequence, uint16 payloadOffset) =
      _wormholeVerifierV2.verify(smallMultisigVaa);
    vm.assertEq(emitterChain, 0);
    vm.assertEq(emitterAddress, bytes32(0));
    vm.assertEq(sequence, 0);
    vm.assertEq(payloadOffset, 1 + 4 + 1 + 66*SHARD_QUORUM + 4 + 4 + 2 + 32 + 8 + 1);
  }

  function test_benchmark_verifySchnorr() public view {
    (uint16 emitterChain, bytes32 emitterAddress, uint64 sequence, uint16 payloadOffset) =
      _wormholeVerifierV2.verify(smallSchnorrVaa);
    vm.assertEq(emitterChain, 0);
    vm.assertEq(emitterAddress, bytes32(0));
    vm.assertEq(sequence, 0);
    vm.assertEq(payloadOffset, 1 + 4 + 20 + 32 + 4 + 4 + 2 + 32 + 8 + 1);
  }

  function test_benchmark_verifyMultisigBig() public view {
    (uint16 emitterChain, bytes32 emitterAddress, uint64 sequence, uint16 payloadOffset) =
      _wormholeVerifierV2.verify(bigMultisigVaa);
    vm.assertEq(emitterChain, 0);
    vm.assertEq(emitterAddress, bytes32(0));
    vm.assertEq(sequence, 0);
    vm.assertEq(payloadOffset, 1 + 4 + 1 + 66*SHARD_QUORUM + 4 + 4 + 2 + 32 + 8 + 1);
  }

  function test_benchmark_verifySchnorrBig() public view {
    (uint16 emitterChain, bytes32 emitterAddress, uint64 sequence, uint16 payloadOffset) =
      _wormholeVerifierV2.verify(bigSchnorrVaa);
    vm.assertEq(emitterChain, 0);
    vm.assertEq(emitterAddress, bytes32(0));
    vm.assertEq(sequence, 0);
    vm.assertEq(payloadOffset, 1 + 4 + 20 + 32 + 4 + 4 + 2 + 32 + 8 + 1);
  }

  function test_verifyInvalidVersion() public {
    vm.expectRevert(abi.encodeWithSelector(WormholeVerifier.VerificationFailed.selector, MASK_VERIFY_RESULT_INVALID_VERSION));
    _wormholeVerifierV2.verify(invalidVersionVaa);
  }

  function test_verifyInvalidMultisig() public {
    vm.expectRevert(abi.encodeWithSelector(WormholeVerifier.VerificationFailed.selector, MASK_VERIFY_RESULT_INVALID_SIGNATURE_COUNT));
    _wormholeVerifierV2.verify(invalidMultisigVaa);
  }

  function test_verifyInvalidSchnorr() public {
    vm.expectRevert(abi.encodeWithSelector(WormholeVerifier.VerificationFailed.selector, MASK_VERIFY_RESULT_SIGNATURE_MISMATCH | MASK_VERIFY_RESULT_INVALID_SIGNATURE));
    _wormholeVerifierV2.verify(invalidSchnorrVaa);
  }

  function test_benchmark_verifyBatchMultisig() public {
    (bool success, bytes memory data) = address(_wormholeVerifierV2).call(batchMultisigMessage);

    vm.assertEq(success, true);
    vm.assertEq(data.length, 0);
  }

  function test_benchmark_verifyBatchSchnorr() public {
    (bool success, bytes memory data) = address(_wormholeVerifierV2).call(batchSchnorrMessage);

    vm.assertEq(success, true);
    vm.assertEq(data.length, 0);
  }

  function test_verifyBatchSchnorrInvalidMessageLength() public {
    (bool success, bytes memory data) = address(_wormholeVerifierV2).call(abi.encodePacked(
      WormholeVerifier.verifyBatch.selector,
      VERIFY_SCHNORR,
      new bytes(4+20+32+1)
    ));

    vm.assertEq(success, false);
    vm.assertEq(data.length, 4+32);
  }
}

contract TestAssembly2 is VerificationTestAPI {
  using VaaLib for bytes;
  using BytesParsing for bytes;

  uint32 private constant EXPIRATION_DELAY_SECONDS = 24 * 60 * 60;

  uint256 private constant SHARD_COUNT = 1;
  uint256 private constant SHARD_QUORUM = 1;

  uint256[] private guardianPrivateKeysSet0;
  address[] private guardianPublicKeysSet0;
  uint32 private expirationTimeSet0;

  // Used to rotate to a new guardian set in some tests
  uint256[] private guardianPrivateKeysSet1;
  address[] private guardianPublicKeysSet1;

  bytes private smallMultisigVaa;
  bytes private bigMultisigVaa;
  bytes private smallSchnorrVaa;
  bytes private bigSchnorrVaa;

  bytes private constant invalidVersionVaa = new bytes(100);
  bytes private invalidMultisigVaa;
  bytes private invalidSchnorrVaa;

  bytes private schnorrShardsRaw;

  bytes private appendSchnorrKeyVaa1;
  bytes private appendSchnorrKeyVaa2;

  bytes private pullGuardianSetMessage = abi.encodePacked(
    UPDATE_PULL_MULTISIG_KEY_DATA,
    uint32(1)
  );

  WormholeV1Mock private immutable _wormholeV1Mock = new WormholeV1Mock();
  WormholeVerifier private immutable _wormholeVerifierV2 = new WormholeVerifier(_wormholeV1Mock, 0, 0, new bytes(0));

  function setUp() public {
    // Generate the guardian sets
    (guardianPrivateKeysSet0, guardianPublicKeysSet0) = newKeySet(SHARD_COUNT);
    (guardianPrivateKeysSet1, guardianPublicKeysSet1) = newKeySet(SHARD_COUNT);

    expirationTimeSet0 = uint32(block.timestamp + EXPIRATION_DELAY_SECONDS);
    // Append the initial guardian set to the wormholeV1Mock
    _wormholeV1Mock.appendGuardianSet(GuardianSet({
      expirationTime: expirationTimeSet0,
      keys: guardianPublicKeysSet0
    }));

    // Create a slice of the guardian private keys for the multisig to hit quorum without wasting gas
    uint256[] memory guardianPrivateKeysSlice = new uint256[](SHARD_QUORUM);
    for (uint256 i = 0; i < SHARD_QUORUM; i++) {
      guardianPrivateKeysSlice[i] = guardianPrivateKeysSet0[i];
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

    invalidMultisigVaa = new bytes(100);
    invalidMultisigVaa[0] = 0x01;

    invalidSchnorrVaa = new bytes(100);
    invalidSchnorrVaa[0] = 0x02;

    // Generate shard data
    ShardData[] memory schnorrShards = new ShardData[](SHARD_COUNT);
    for (uint256 i = 0; i < SHARD_COUNT; i++) {
      schnorrShards[i] = ShardData({
        shard: bytes32(vm.randomUint()),
        id: bytes32(vm.randomUint())
      });
    }

    schnorrShardsRaw = shardDataToBytes(schnorrShards);
    require(schnorrShardsRaw.length == SHARD_COUNT*64);

    bytes32 schnorrShardDataHash = keccak256(schnorrShardsRaw);
    bytes memory appendSchnorrKeyMessage1 = newAppendSchnorrKeyMessage(0, 0, pk1, 0, schnorrShardDataHash);
    bytes memory appendSchnorrKeyEnvelope1 = newVaaEnvelope(uint32(block.timestamp), 0, CHAIN_ID_SOLANA, GOVERNANCE_ADDRESS, 0, 0, appendSchnorrKeyMessage1);
    appendSchnorrKeyVaa1 = newMultisigVaa(0, signMultisig(appendSchnorrKeyEnvelope1, guardianPrivateKeysSet0), appendSchnorrKeyEnvelope1);

    bytes memory appendSchnorrKeyMessage2 = newAppendSchnorrKeyMessage(1, 0, pk2, EXPIRATION_DELAY_SECONDS, schnorrShardDataHash);
    bytes memory appendSchnorrKeyEnvelope2 = newVaaEnvelope(uint32(block.timestamp), 0, CHAIN_ID_SOLANA, GOVERNANCE_ADDRESS, 0, 0, appendSchnorrKeyMessage2);
    appendSchnorrKeyVaa2 = newMultisigVaa(0, signMultisig(appendSchnorrKeyEnvelope2, guardianPrivateKeysSet0), appendSchnorrKeyEnvelope2);

    // bytes memory message = abi.encodePacked(
    //   UPDATE_PULL_MULTISIG_KEY_DATA,
    //   uint32(1),
    //   UPDATE_APPEND_SCHNORR_KEY,
    //   appendSchnorrKeyVaa1,
    //   schnorrShardsRaw,
    //   UPDATE_APPEND_SCHNORR_KEY,
    //   appendSchnorrKeyVaa2,
    //   schnorrShardsRaw
    // );

    // _wormholeVerifierV2.update(message);
  }

  function test_verifyMultisig() public {
    pullGuardianSets(_wormholeVerifierV2, 1);
    (uint16 emitterChain, bytes32 emitterAddress, uint64 sequence, uint16 payloadOffset) =
      _wormholeVerifierV2.verify(smallMultisigVaa);
    vm.assertEq(emitterChain, 0);
    vm.assertEq(emitterAddress, bytes32(0));
    vm.assertEq(sequence, 0);
    vm.assertEq(payloadOffset, 1 + 4 + 1 + 66*SHARD_QUORUM + 4 + 4 + 2 + 32 + 8 + 1);
  }

  function test_verifySchnorr() public {
    pullGuardianSets(_wormholeVerifierV2, 1);
    appendSchnorrKey(_wormholeVerifierV2, appendSchnorrKeyVaa1, schnorrShardsRaw);
    (uint16 emitterChain, bytes32 emitterAddress, uint64 sequence, uint16 payloadOffset) =
      _wormholeVerifierV2.verify(smallSchnorrVaa);
    vm.assertEq(emitterChain, 0);
    vm.assertEq(emitterAddress, bytes32(0));
    vm.assertEq(sequence, 0);
    vm.assertEq(payloadOffset, 1 + 4 + 20 + 32 + 4 + 4 + 2 + 32 + 8 + 1);
  }

  function test_verifyMultisigBig() public {
    pullGuardianSets(_wormholeVerifierV2, 1);
    (uint16 emitterChain, bytes32 emitterAddress, uint64 sequence, uint16 payloadOffset) =
      _wormholeVerifierV2.verify(bigMultisigVaa);
    vm.assertEq(emitterChain, 0);
    vm.assertEq(emitterAddress, bytes32(0));
    vm.assertEq(sequence, 0);
    vm.assertEq(payloadOffset, 1 + 4 + 1 + 66*SHARD_QUORUM + 4 + 4 + 2 + 32 + 8 + 1);
  }

  function test_verifySchnorrBig() public {
    pullGuardianSets(_wormholeVerifierV2, 1);
    appendSchnorrKey(_wormholeVerifierV2, appendSchnorrKeyVaa1, schnorrShardsRaw);
    appendSchnorrKey(_wormholeVerifierV2, appendSchnorrKeyVaa2, schnorrShardsRaw);
    (uint16 emitterChain, bytes32 emitterAddress, uint64 sequence, uint16 payloadOffset) =
      _wormholeVerifierV2.verify(bigSchnorrVaa);
    vm.assertEq(emitterChain, 0);
    vm.assertEq(emitterAddress, bytes32(0));
    vm.assertEq(sequence, 0);
    vm.assertEq(payloadOffset, 1 + 4 + 20 + 32 + 4 + 4 + 2 + 32 + 8 + 1);
  }

  // V1 codepaths

  function test_pullGuardianSets_pullsOne() public {
    pullGuardianSets(_wormholeVerifierV2, 1);
  }

  function test_pullGuardianSets_pullsWithExceedingLimit() public {
    pullGuardianSets(_wormholeVerifierV2, 5);
  }

  function test_getCurrentGuardianSet() public {
    pullGuardianSets(_wormholeVerifierV2, 1);
    bytes memory result = _wormholeVerifierV2.get(getCurrentGuardianSet());

    (address[] memory guardianSetAddrs, uint32 guardianSetIndex,) = decodeGetCurrentGuardianSet(result, 0);
    assertEq(guardianSetAddrs.length, 1);
    assertEq(guardianSetAddrs[0], guardianPublicKeysSet0[0]);
    assertEq(guardianSetIndex, 0);
  }

  function test_getGuardianSet() public {
    // Add a new guardian set
    uint32 expirationTimeSet1 = uint32(block.timestamp + 10);
    _wormholeV1Mock.appendGuardianSet(GuardianSet({
      keys: guardianPublicKeysSet1,
      expirationTime: expirationTimeSet1
    }));

    pullGuardianSets(_wormholeVerifierV2, 4);

    // Get the old guardian set
    bytes memory result = _wormholeVerifierV2.get(getGuardianSet(0));

    // Decode the guardian set
    (address[] memory guardianSetAddrs, uint32 expirationTime,) = decodeGetGuardianSet(result, 0);
    assertEq(guardianSetAddrs.length, 1);
    assertEq(guardianSetAddrs[0], guardianPublicKeysSet0[0]);
    assertEq(expirationTime, expirationTimeSet0);

    // Get the new guardian set
    bytes memory result2 = _wormholeVerifierV2.get(getGuardianSet(1));

    // Decode the guardian set
    (address[] memory guardianSetAddrs2, uint32 expirationTime2,) = decodeGetGuardianSet(result2, 0);
    assertEq(guardianSetAddrs2.length, 1);
    assertEq(guardianSetAddrs2[0], guardianPublicKeysSet1[0]);
    assertEq(expirationTime2, expirationTimeSet1);
  }

  function test_getGuardianSet_unknownGuardianSet() public {
    pullGuardianSets(_wormholeVerifierV2, 4);

    uint32 uninitializedGuardianSet = 10000;
    vm.expectRevert(abi.encodeWithSelector(
      WormholeVerifier.UnknownGuardianSet.selector,
      uninitializedGuardianSet
    ));
    _wormholeVerifierV2.get(getGuardianSet(uninitializedGuardianSet));
  }

  function testRevert_verifyVaaV1() public {
    pullGuardianSets(_wormholeVerifierV2, 5);
    vm.expectRevert(abi.encodeWithSelector(
      WormholeVerifier.VerificationFailed.selector,
      MASK_VERIFY_RESULT_INVALID_VERSION
    ));
    _wormholeVerifierV2.verify(invalidVersionVaa);
  }

  function testRevert_verifyVaaV1_notRegisteredGuardianSet() public {
    uint32 fakeGuardianSetIndex = 5;

    uint256[] memory guardianPrivateKeysSlice = new uint256[](SHARD_QUORUM);
    for (uint256 i = 0; i < SHARD_QUORUM; i++) {
      guardianPrivateKeysSlice[i] = guardianPrivateKeysSet0[i];
    }
    bytes memory smallEnvelope = new bytes(100);
    bytes memory smallMultisigSignatures = signMultisig(smallEnvelope, guardianPrivateKeysSlice);
    bytes memory vaa = newMultisigVaa(fakeGuardianSetIndex, smallMultisigSignatures, smallEnvelope);

    vm.expectRevert(abi.encodeWithSelector(
      WormholeVerifier.VerificationFailed.selector,
      MASK_VERIFY_RESULT_INVALID_KEY_DATA_SIZE
    ));
    _wormholeVerifierV2.verify(vaa);
  }

  function testRevert_verifyVaaV1_skippedGuardianSet() public {
    WormholeV1Mock wormholeMock = new WormholeV1Mock();

    wormholeMock.appendGuardianSet(GuardianSet({
      keys: guardianPublicKeysSet0,
      expirationTime: 0
    }));

    wormholeMock.appendGuardianSet(GuardianSet({
      keys: guardianPublicKeysSet0,
      expirationTime: 0
    }));

    wormholeMock.appendGuardianSet(GuardianSet({
      keys: guardianPublicKeysSet0,
      expirationTime: 0
    }));

    uint32 initGuardianSetIndex = 2;
    WormholeVerifier tempVerifier = new WormholeVerifier(wormholeMock, initGuardianSetIndex, 0, pullGuardianSetMessage);

    // small multisig VAA has guardian set 0 in the header
    vm.expectRevert(abi.encodeWithSelector(
      WormholeVerifier.VerificationFailed.selector,
      MASK_VERIFY_RESULT_INVALID_KEY_DATA_SIZE
    ));
    tempVerifier.verify(smallMultisigVaa);
  }

  // V2 codepaths

  function test_appendSchnorrKey() public {
    pullGuardianSets(_wormholeVerifierV2, 1);
    appendSchnorrKey(_wormholeVerifierV2, appendSchnorrKeyVaa1, schnorrShardsRaw);
  }

  function test_appendMultipleSchnorrKey() public {
    pullGuardianSets(_wormholeVerifierV2, 1);

    uint256 pk1 = 0x79380e24c7cbb0f88706dd035135020063aab3e7f403398ff7f995af0b8a770c << 1;
    bytes32 schnorrShardDataHash = keccak256(schnorrShardsRaw);

    uint32 schnorrKeyIndex = 2;
    bytes memory appendSchnorrKeyMessage3 = newAppendSchnorrKeyMessage(schnorrKeyIndex, 0, pk1, 0, schnorrShardDataHash);
    bytes memory appendSchnorrKeyEnvelope3 = newVaaEnvelope(uint32(block.timestamp), 0, CHAIN_ID_SOLANA, GOVERNANCE_ADDRESS, 0, 0, appendSchnorrKeyMessage3);
    bytes memory appendSchnorrKeyVaa3 = newMultisigVaa(0, signMultisig(appendSchnorrKeyEnvelope3, guardianPrivateKeysSet0), appendSchnorrKeyEnvelope3);

    uint32 schnorrKeyIndex2 = 3;
    bytes memory appendSchnorrKeyMessage4 = newAppendSchnorrKeyMessage(schnorrKeyIndex2, 0, pk1, 0, schnorrShardDataHash);
    bytes memory appendSchnorrKeyEnvelope4 = newVaaEnvelope(uint32(block.timestamp), 0, CHAIN_ID_SOLANA, GOVERNANCE_ADDRESS, 0, 0, appendSchnorrKeyMessage4);
    bytes memory appendSchnorrKeyVaa4 = newMultisigVaa(0, signMultisig(appendSchnorrKeyEnvelope4, guardianPrivateKeysSet0), appendSchnorrKeyEnvelope4);

    appendSchnorrKey(_wormholeVerifierV2, appendSchnorrKeyVaa1, schnorrShardsRaw);
    appendSchnorrKey(_wormholeVerifierV2, appendSchnorrKeyVaa2, schnorrShardsRaw);
    appendSchnorrKey(_wormholeVerifierV2, appendSchnorrKeyVaa3, schnorrShardsRaw);
    appendSchnorrKey(_wormholeVerifierV2, appendSchnorrKeyVaa4, schnorrShardsRaw);
  }

  function test_appendSchnorrKey_canSkipIndicesOnDeploy() public {
    uint32 schnorrKeyIndex2 = 3;
    WormholeVerifier tempVerifier = new WormholeVerifier(_wormholeV1Mock, 0, schnorrKeyIndex2, pullGuardianSetMessage);

    uint256 pk1 = 0x79380e24c7cbb0f88706dd035135020063aab3e7f403398ff7f995af0b8a770c << 1;
    bytes32 schnorrShardDataHash = keccak256(schnorrShardsRaw);

    bytes memory appendSchnorrKeyMessage4 = newAppendSchnorrKeyMessage(schnorrKeyIndex2, 0, pk1, 0, schnorrShardDataHash);
    bytes memory appendSchnorrKeyEnvelope4 = newVaaEnvelope(uint32(block.timestamp), 0, CHAIN_ID_SOLANA, GOVERNANCE_ADDRESS, 0, 0, appendSchnorrKeyMessage4);
    bytes memory appendSchnorrKeyVaa4 = newMultisigVaa(0, signMultisig(appendSchnorrKeyEnvelope4, guardianPrivateKeysSet0), appendSchnorrKeyEnvelope4);

    appendSchnorrKey(tempVerifier, appendSchnorrKeyVaa4, schnorrShardsRaw);
  }

  function test_appendSchnorrKey_deployCanPreventSubmissionOfOldIndices() public {
    uint32 initialSchnorrKeyIndex = 3;
    WormholeVerifier tempVerifier = new WormholeVerifier(_wormholeV1Mock, 0, initialSchnorrKeyIndex, pullGuardianSetMessage);

    vm.expectRevert(abi.encodeWithSelector(
      WormholeVerifier.UpdateFailed.selector,
      MASK_UPDATE_RESULT_INVALID_KEY_INDEX | 0xe9
    ));
    appendSchnorrKey(tempVerifier, appendSchnorrKeyVaa1, schnorrShardsRaw);
  }

  function testRevert_appendSchnorrKey() public {
    vm.expectRevert(abi.encodeWithSelector(
      WormholeVerifier.VerificationFailed.selector,
      MASK_VERIFY_RESULT_INVALID_KEY_DATA_SIZE
    ));
    appendSchnorrKey(_wormholeVerifierV2, invalidMultisigVaa, schnorrShardsRaw);
  }

  function testRevert_appendSchnorrKey_duplicatedKey() public {
    pullGuardianSets(_wormholeVerifierV2, 1);
    appendSchnorrKey(_wormholeVerifierV2, appendSchnorrKeyVaa1, schnorrShardsRaw);

    vm.expectRevert(abi.encodeWithSelector(
      WormholeVerifier.UpdateFailed.selector,
      MASK_UPDATE_RESULT_INVALID_KEY_INDEX | 0xe9
    ));
    appendSchnorrKey(_wormholeVerifierV2, appendSchnorrKeyVaa1, schnorrShardsRaw);
  }

  function testRevert_appendOldSchnorrKey() public {
    pullGuardianSets(_wormholeVerifierV2, 1);

    uint256 pk1 = 0x79380e24c7cbb0f88706dd035135020063aab3e7f403398ff7f995af0b8a770c << 1;
    bytes32 schnorrShardDataHash = keccak256(schnorrShardsRaw);

    uint32 schnorrKeyIndex = 0;
    bytes memory appendSchnorrKeyMessage3 = newAppendSchnorrKeyMessage(schnorrKeyIndex, 0, pk1, 0, schnorrShardDataHash);
    bytes memory appendSchnorrKeyEnvelope3 = newVaaEnvelope(uint32(block.timestamp), 0, CHAIN_ID_SOLANA, GOVERNANCE_ADDRESS, 0, 0, appendSchnorrKeyMessage3);
    bytes memory appendSchnorrKeyVaa3 = newMultisigVaa(0, signMultisig(appendSchnorrKeyEnvelope3, guardianPrivateKeysSet0), appendSchnorrKeyEnvelope3);

    appendSchnorrKey(_wormholeVerifierV2, appendSchnorrKeyVaa1, schnorrShardsRaw);
    appendSchnorrKey(_wormholeVerifierV2, appendSchnorrKeyVaa2, schnorrShardsRaw);

    vm.expectRevert(abi.encodeWithSelector(
      WormholeVerifier.UpdateFailed.selector,
      MASK_UPDATE_RESULT_INVALID_KEY_INDEX | 0xe9
    ));
    appendSchnorrKey(_wormholeVerifierV2, appendSchnorrKeyVaa3, schnorrShardsRaw);
  }

  function testRevert_appendMaxSchnorrKey() public {
    uint32 schnorrKeyIndex = type(uint32).max;
    WormholeVerifier tempVerifier = new WormholeVerifier(_wormholeV1Mock, 0, schnorrKeyIndex, pullGuardianSetMessage);

    uint256 pk1 = 0x79380e24c7cbb0f88706dd035135020063aab3e7f403398ff7f995af0b8a770c << 1;
    bytes32 schnorrShardDataHash = keccak256(schnorrShardsRaw);

    bytes memory appendSchnorrKeyMessage3 = newAppendSchnorrKeyMessage(schnorrKeyIndex, 0, pk1, 0, schnorrShardDataHash);
    bytes memory appendSchnorrKeyEnvelope3 = newVaaEnvelope(uint32(block.timestamp), 0, CHAIN_ID_SOLANA, GOVERNANCE_ADDRESS, 0, 0, appendSchnorrKeyMessage3);
    bytes memory appendSchnorrKeyVaa3 = newMultisigVaa(0, signMultisig(appendSchnorrKeyEnvelope3, guardianPrivateKeysSet0), appendSchnorrKeyEnvelope3);

    vm.expectRevert(abi.encodeWithSelector(
      WormholeVerifier.UpdateFailed.selector,
      MASK_UPDATE_RESULT_INVALID_KEY_INDEX | 0xe9
    ));
    appendSchnorrKey(tempVerifier, appendSchnorrKeyVaa3, schnorrShardsRaw);
  }

  function test_getCurrentSchnorrKey() public {
    pullGuardianSets(_wormholeVerifierV2, 1);
    appendSchnorrKey(_wormholeVerifierV2, appendSchnorrKeyVaa1, schnorrShardsRaw);
    appendSchnorrKey(_wormholeVerifierV2, appendSchnorrKeyVaa2, schnorrShardsRaw);

    uint256 pk2 = 0x44c90dfbe2a454987a65ce9e6f522c9c5c9d1dfb3c3aaaadcd0ae4f5366a2922 << 1;

    bytes memory result = _wormholeVerifierV2.get(getCurrentSchnorrKey());

    (
      uint32 schnorrKeyIndex,
      uint256 schnorrKeyPubkey,
      uint32 expirationTime,
      uint8 shardCount,
      uint32 guardianSet,
    ) = decodeGetCurrentSchnorrKey(result, 0);
    assertEq(schnorrKeyIndex, 1);
    assertEq(schnorrKeyPubkey, pk2);
    assertEq(expirationTime, 0);
    assertEq(shardCount, SHARD_COUNT);
    assertEq(guardianSet, 0);
  }

  function test_getSchnorrShards() public {
    pullGuardianSets(_wormholeVerifierV2, 1);
    appendSchnorrKey(_wormholeVerifierV2, appendSchnorrKeyVaa1, schnorrShardsRaw);

    uint32 schnorrKeyIndex = 0;
    bytes memory result = _wormholeVerifierV2.get(getShardData(schnorrKeyIndex));

    (
      ShardData[] memory shards,
    ) = decodeShardData(result, 0);
    uint256 shardCount = schnorrShardsRaw.length / (LENGTH_WORD * 2);
    assertEq(shards.length, shardCount);

    uint256 offset = 0;
    for (uint i = 0; i < shardCount; ++i) {
      bytes32 shard;
      bytes32 id;
      (shard, offset) = schnorrShardsRaw.asBytes32MemUnchecked(offset);
      (id,    offset) = schnorrShardsRaw.asBytes32MemUnchecked(offset);

      assertEq(shard, shards[i].shard);
      assertEq(id,    shards[i].id);
    }
  }

  function test_getSchnorrKey() public {
    pullGuardianSets(_wormholeVerifierV2, 1);
    appendSchnorrKey(_wormholeVerifierV2, appendSchnorrKeyVaa1, schnorrShardsRaw);
    appendSchnorrKey(_wormholeVerifierV2, appendSchnorrKeyVaa2, schnorrShardsRaw);

    uint256 pk1 = 0x79380e24c7cbb0f88706dd035135020063aab3e7f403398ff7f995af0b8a770c << 1;

    bytes memory result = _wormholeVerifierV2.get(getSchnorrKey(0));

    (
      uint256 schnorrKeyPubkey,
      uint32 expirationTime,
      uint8 shardCount,
      uint32 guardianSet,
    ) = decodeGetSchnorrKey(result, 0);
    assertEq(schnorrKeyPubkey, pk1);
    assertEq(expirationTime, expirationTimeSet0);
    assertEq(shardCount, SHARD_COUNT);
    assertEq(guardianSet, 0);
  }


  function testRevert_verifyVaaV2_unregisteredKey() public {
    pullGuardianSets(_wormholeVerifierV2, 1);
    appendSchnorrKey(_wormholeVerifierV2, appendSchnorrKeyVaa1, schnorrShardsRaw);

    vm.expectRevert(abi.encodeWithSelector(
      WormholeVerifier.VerificationFailed.selector,
      MASK_VERIFY_RESULT_INVALID_KEY | MASK_VERIFY_RESULT_SIGNATURE_MISMATCH
    ));
    _wormholeVerifierV2.verify(bigSchnorrVaa);
  }

  function test_verifyVaaV2_skippedKey() public {
    WormholeVerifier tempVerifier = new WormholeVerifier(_wormholeV1Mock, 0, 1, pullGuardianSetMessage);
    pullGuardianSets(tempVerifier, 1);
    appendSchnorrKey(tempVerifier, appendSchnorrKeyVaa2, schnorrShardsRaw);

    vm.expectRevert(abi.encodeWithSelector(
      WormholeVerifier.VerificationFailed.selector,
      MASK_VERIFY_RESULT_INVALID_KEY | MASK_VERIFY_RESULT_SIGNATURE_MISMATCH
    ));
    _wormholeVerifierV2.verify(smallSchnorrVaa);
  }
}

