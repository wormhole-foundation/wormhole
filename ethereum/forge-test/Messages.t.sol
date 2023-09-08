// test/Messages.sol
// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.0;

import "../contracts/Messages.sol";
import "../contracts/Setters.sol";
import "../contracts/Structs.sol";

import "forge-std/Test.sol";
import "forge-std/Vm.sol";

contract WormholeSigner is Test {
  // Signer wallet. 
  struct Wallet {
    address addr;
    uint256 key;
  }

  function encodeAndSignMessage(
    Structs.VM memory vm_, 
    uint256[] memory guardianKeys, 
    uint32 guardianSetIndex
  ) public pure returns (bytes memory signedMessage) {
    // Compute the hash of the body
    bytes memory body = abi.encodePacked(
        vm_.timestamp,
        vm_.nonce,
        vm_.emitterChainId,
        vm_.emitterAddress,
        vm_.sequence,
        vm_.consistencyLevel,
        vm_.payload
    );
    vm_.hash = keccak256(abi.encodePacked(keccak256(body)));

    // Sign the hash with the specified guardian private keys.
    uint256 guardianCount = guardianKeys.length;
    bytes memory signatures = abi.encodePacked(uint8(guardianCount));
    for (uint256 i = 0; i < guardianCount; ++i) {
      (uint8 v, bytes32 r, bytes32 s) = vm.sign(guardianKeys[i], vm_.hash);
      signatures = abi.encodePacked(signatures, uint8(i), r, s, v - 27);
    }

    signedMessage = abi.encodePacked(
      vm_.version,
      guardianSetIndex,
      signatures,
      body
    );
  } 
}

contract ExportedMessages is Messages, Setters {
    function storeGuardianSetPub(Structs.GuardianSet memory set, uint32 index) public {
        return super.storeGuardianSet(set, index);
    }
}

contract TestMessages is Test {
  address constant testGuardianPub = 0xbeFA429d57cD18b7F8A4d91A2da9AB4AF05d0FBe;

  // A valid VM with one signature from the testGuardianPublic key
  bytes validVM = hex"01000000000100867b55fec41778414f0683e80a430b766b78801b7070f9198ded5e62f48ac7a44b379a6cf9920e42dbd06c5ebf5ec07a934a00a572aefc201e9f91c33ba766d900000003e800000001000b0000000000000000000000000000000000000000000000000000000000000eee00000000000005390faaaa";

  uint256 constant testGuardian = 93941733246223705020089879371323733820373732307041878556247502674739205313440;

  ExportedMessages messages;
  WormholeSigner wormholeSimulator; 

  Structs.GuardianSet guardianSet;

  // Guardian set with 19 guardians and wallets with each signing key. 
  Structs.GuardianSet guardianSetOpt;
  uint256[] guardianKeys = new uint256[](19);

  function setupSingleGuardian() internal {
    // initialize guardian set with one guardian
    address[] memory keys = new address[](1);
    keys[0] = vm.addr(testGuardian);
    guardianSet = Structs.GuardianSet(keys, 0);
    require(messages.quorum(guardianSet.keys.length) == 1, "Quorum should be 1");
  }

  function setupMultiGuardian() internal {
    // initialize guardian set with 19 guardians 
    address[] memory keys = new address[](19);
    for (uint256 i = 0; i < 19; ++i) {
      // create a keypair for each guardian 
      VmSafe.Wallet memory wallet = vm.createWallet(string(abi.encodePacked("guardian", i)));
      keys[i] = wallet.addr; 
      guardianKeys[i] = wallet.privateKey; 
    }
    guardianSetOpt = Structs.GuardianSet(keys, 0); 
    require(messages.quorum(guardianSetOpt.keys.length) == 13, "Quorum should be 13"); 
  }

  function setUp() public {
    messages = new ExportedMessages();
    wormholeSimulator = new WormholeSigner();
    setupSingleGuardian();
    setupMultiGuardian();
  } 

  function getSignedVM(
    bytes memory payload,
    bytes32 emitterAddress,
    uint16 emitterChainId,
    uint256[] memory _guardianKeys,
    uint32 guardianSetIndex
  ) internal view returns (bytes memory signedTransfer) {
    // construct `TransferWithPayload` Wormhole message
    Structs.VM memory vm;

    // set the vm values inline
    vm.version = uint8(1);
    vm.timestamp = uint32(block.timestamp);
    vm.emitterChainId = emitterChainId;
    vm.emitterAddress = emitterAddress;
    vm.sequence = messages.nextSequence(
        address(uint160(uint256(emitterAddress)))
    );
    vm.consistencyLevel = 15;
    vm.payload = payload;

    // encode the bservation
    signedTransfer = wormholeSimulator.encodeAndSignMessage(
      vm,
      _guardianKeys,
      guardianSetIndex
    );
  }

  function testQuorum() public {
    assertEq(messages.quorum(0), 1);
    assertEq(messages.quorum(1), 1);
    assertEq(messages.quorum(2), 2);
    assertEq(messages.quorum(3), 3);
    assertEq(messages.quorum(4), 3);
    assertEq(messages.quorum(5), 4);
    assertEq(messages.quorum(6), 5);
    assertEq(messages.quorum(7), 5);
    assertEq(messages.quorum(8), 6);
    assertEq(messages.quorum(9), 7);
    assertEq(messages.quorum(10), 7);
    assertEq(messages.quorum(11), 8);
    assertEq(messages.quorum(12), 9);
    assertEq(messages.quorum(19), 13);
    assertEq(messages.quorum(20), 14);
  }

  function testQuorumCanAlwaysBeReached(uint256 numGuardians) public {
    vm.assume(numGuardians > 0);

    if (numGuardians >= 256) {
      vm.expectRevert("too many guardians");
    }
    // test that quorums is never greater than the number of guardians
    assert(messages.quorum(numGuardians) <= numGuardians);
  }

  // This test ensures that submitting more signatures than expected will
  // trigger a "guardian index out of bounds" error.
  function testCannotVerifySignaturesWithOutOfBoundsSignature(bytes memory encoded) public {
    vm.assume(encoded.length > 0);

    bytes32 message = keccak256(encoded);

    // First generate a legitimate signature.
    Structs.Signature memory goodSignature = Structs.Signature(message, 0, 0, 0);
    (goodSignature.v, goodSignature.r, goodSignature.s) = vm.sign(testGuardian, message);
    assertEq(ecrecover(message, goodSignature.v, goodSignature.r, goodSignature.s), vm.addr(testGuardian));

    // Reuse legitimate signature above for the next signature. This will
    // bypass the "invalid signature" revert.
    Structs.Signature memory outOfBoundsSignature = goodSignature;
    outOfBoundsSignature.guardianIndex = 1;

    // Attempt to verify signatures.
    Structs.Signature[] memory sigs = new Structs.Signature[](2);
    sigs[0] = goodSignature;
    sigs[1] = outOfBoundsSignature;

    vm.expectRevert("guardian index out of bounds");
    messages.verifySignatures(message, sigs, guardianSet);
  }

  // This test ensures that submitting an invalid signature fails when
  // verifySignatures is called. Calling ecrecover should fail.
  function testCannotVerifySignaturesWithInvalidSignature(bytes memory encoded) public {
    vm.assume(encoded.length > 0);

    bytes32 message = keccak256(encoded);

    // Generate an invalid signature.
    Structs.Signature memory badSignature = Structs.Signature(message, 0, 0, 0);
    assertEq(ecrecover(message, badSignature.v, badSignature.r, badSignature.s), address(0));

    // Attempt to verify signatures.
    Structs.Signature[] memory sigs = new Structs.Signature[](2);
    sigs[0] = badSignature;

    vm.expectRevert("ecrecover failed with signature");
    messages.verifySignatures(message, sigs, guardianSet);
  }

  function testVerifySignatures(bytes memory encoded) public {
    vm.assume(encoded.length > 0);

    bytes32 message = keccak256(encoded);

    // Generate legitimate signature.
    Structs.Signature memory goodSignature;
    (goodSignature.v, goodSignature.r, goodSignature.s) = vm.sign(testGuardian, message);
    assertEq(ecrecover(message, goodSignature.v, goodSignature.r, goodSignature.s), vm.addr(testGuardian));
    goodSignature.guardianIndex = 0;

    // Attempt to verify signatures.
    Structs.Signature[] memory sigs = new Structs.Signature[](1);
    sigs[0] = goodSignature;

    (bool valid, string memory reason) = messages.verifySignatures(message, sigs, guardianSet);
    assertEq(valid, true);
    assertEq(bytes(reason).length, 0);
  }

  // This test checks the possibility of getting a unsigned message verified through verifyVM
  function testHashMismatchedVMIsNotVerified() public {
    // Set the initial guardian set
    address[] memory initialGuardians = new address[](1);
    initialGuardians[0] = testGuardianPub;

    // Create a guardian set
    Structs.GuardianSet memory initialGuardianSet = Structs.GuardianSet({
      keys: initialGuardians,
      expirationTime: 0
    });

    messages.storeGuardianSetPub(initialGuardianSet, uint32(0));

    // Confirm that the test VM is valid
    (Structs.VM memory parsedValidVm, bool valid, string memory reason) = messages.parseAndVerifyVM(validVM);
    require(valid, reason);
    assertEq(valid, true);
    assertEq(reason, "");

    // Manipulate the payload of the vm
    Structs.VM memory invalidVm = parsedValidVm;
    invalidVm.payload = abi.encodePacked(
        parsedValidVm.payload,
        "malicious bytes in payload"
    );

    // Confirm that the verifyVM fails on invalid VM
    (valid, reason) = messages.verifyVM(invalidVm);
    assertEq(valid, false);
    assertEq(reason, "vm.hash doesn't match body");
  }

  function testParseGuardianSetOptimized(uint8 guardianCount) public view {
    vm.assume(guardianCount > 0 && guardianCount <= 19);

    // Encode the guardian set.
    bytes memory encodedGuardianSet;
    for (uint256 i = 0; i < guardianCount; ++i) {
      encodedGuardianSet = abi.encodePacked(encodedGuardianSet, guardianSetOpt.keys[i]);
    }
    encodedGuardianSet = abi.encodePacked(encodedGuardianSet, guardianSetOpt.expirationTime);

    // Parse the guardian set. 
    Structs.GuardianSet memory parsedSet = messages.parseGuardianSetOptimized(encodedGuardianSet);

    // Validate the results by comparing the parsed set to the original set.
    for (uint256 i = 0; i < guardianCount; ++i) {
      assert(parsedSet.keys[i] == guardianSetOpt.keys[i]);
    } 
    assert(parsedSet.expirationTime == guardianSetOpt.expirationTime);
  }

  function testParseAndVerifyVMOptimized(bytes memory payload) public {
    vm.assume(payload.length > 0 && payload.length < 1000);

    uint16 emitterChainId = 16;
    bytes32 emitterAddress = bytes32(uint256(uint160(makeAddr("foreignEmitter"))));
    uint32 currentSetIndex = messages.getCurrentGuardianSetIndex();

    // Set the guardian set to the optimized guardian set.
    messages.storeGuardianSetPub(guardianSetOpt, currentSetIndex);
    messages.setGuardianSetHash(currentSetIndex);

    // Create a message with an arbitrary payload. 
    bytes memory signedMessage = getSignedVM(
      payload,
      emitterAddress,
      emitterChainId,
      guardianKeys,
      currentSetIndex
    );

    // Parse and verify the VM. 
    (Structs.VM memory vm_, bool valid,) = messages.parseAndVerifyVM(signedMessage);
    assertEq(valid, true);

    // Parse and verify the VM using the optimized endpoint. 
    (Structs.VM memory vmOptimized, bool valid_,) = messages.parseAndVerifyVMOptimized(
      signedMessage, 
      messages.getEncodedGuardianSet(currentSetIndex), 
      currentSetIndex
    );
    assertEq(valid_, true);

    // Validate the results by comparing the parsed VM to the optimized VM.
    assertEq(vm_.version, vmOptimized.version);
    assertEq(vm_.timestamp, vmOptimized.timestamp);
    assertEq(vm_.nonce, vmOptimized.nonce);
    assertEq(vm_.emitterChainId, vmOptimized.emitterChainId);
    assertEq(vm_.emitterAddress, vmOptimized.emitterAddress);
    assertEq(vm_.sequence, vmOptimized.sequence);
    assertEq(vm_.consistencyLevel, vmOptimized.consistencyLevel);
    assertEq(vm_.payload, vmOptimized.payload);
    assertEq(vm_.guardianSetIndex, vmOptimized.guardianSetIndex);
    assertEq(vm_.signatures.length, vmOptimized.signatures.length);
  }
}
