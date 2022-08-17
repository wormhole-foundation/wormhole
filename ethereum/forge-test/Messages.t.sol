// test/Messages.sol
// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.0;

import "../contracts/Messages.sol";
import "../contracts/Structs.sol";
import "forge-std/Test.sol";

contract TestMessages is Messages, Test {
  address constant testGuardianPub = 0xbeFA429d57cD18b7F8A4d91A2da9AB4AF05d0FBe;

  function testQuorum() public {
    assertEq(quorum(0), 1);
    assertEq(quorum(1), 1);
    assertEq(quorum(2), 2);
    assertEq(quorum(3), 3);
    assertEq(quorum(4), 3);
    assertEq(quorum(5), 4);
    assertEq(quorum(6), 5);
    assertEq(quorum(7), 5);
    assertEq(quorum(8), 6);
    assertEq(quorum(9), 7);
    assertEq(quorum(10), 7);
    assertEq(quorum(11), 8);
    assertEq(quorum(12), 9);
    assertEq(quorum(19), 13);
    assertEq(quorum(20), 14);
  }

  function testQuorumCanAlwaysBeReached(uint numGuardians) public {
    if (numGuardians == 0) {
      return;
    }
    if (numGuardians >= 256) {
      vm.expectRevert("too many guardians");
    }
    // test that quorums is never greater than the number of guardians
    assert(quorum(numGuardians) <= numGuardians);
  }

  // This test ensures that submitting invalid signatures for non-existent
  // guardians fails.
  //
  // The main purpose of this test is to ensure that there's no surprising
  // behaviour arising from solidity's handling of invalid signatures and out of
  // bounds memory access. In particular, pubkey recovery of an invalid
  // signature returns 0, and in some cases out of bounds memory access also
  // just returns 0.
  function testOutOfBoundsSignature() public {
    // Initialise a guardian set with a single guardian.
    address[] memory keys = new address[](1);
    keys[0] = testGuardianPub;
    Structs.GuardianSet memory guardianSet = Structs.GuardianSet(keys, 0);
    require(quorum(guardianSet.keys.length) == 1, "Quorum should be 1");

    // Two invalid signatures, for guardian index 2 and 3 respectively.
    // These guardian indices are out of bounds for the guardian set.
    bytes32 message = "hello";
    Structs.Signature memory bad1 = Structs.Signature(message, 0, 0, 2);
    Structs.Signature memory bad2 = Structs.Signature(message, 0, 0, 3);
    // ecrecover on an invalid signature returns 0 instead of reverting
    require(ecrecover(message, bad1.v, bad1.r, bad1.s) == address(0), "ecrecover should return the 0 address for an invalid signature");

    Structs.Signature[] memory badSigs = new Structs.Signature[](2);
    badSigs[0] = bad1;
    badSigs[1] = bad2;
    vm.expectRevert(bytes("guardian index out of bounds"));
    verifySignatures(0, badSigs, guardianSet);
  }

  function doubleKeccak256(bytes memory body) internal pure returns (bytes32 hash) {
    hash = keccak256(abi.encodePacked(keccak256(body)));
  }

  // This test ensures that individual hashes are not cached
  // when a bad encoded VM2 is passed to parseAndVerifyVM2.
  function testParseAndVerifyVM2Failure() public {
    bytes memory invalidVM2 = hex"020000000001005201ab02c31301c4d3a2e27d5acb85272089eefb083cf9873aeff3a9cf54a15461d062b4de222a1aaa6655318b61f3ea5fabba9889afbc6956034174ca0f0a650102b986975c64018680bbeeb010310922cb3cd7dde2dfdcb318dadf60a3b327883766ecba817383d631066944c6f9f69b05db03dde58102eec081c930f5d87b461600000035000003e800000001000b0000000000000000000000000000000000000000000000000000000000000eee00000000000005390faaaa00000036000003e900000001000b0000000000000000000000000000000000000000000000000000000000000eee000000000000053a0fbbbbbb";
    uint8 expectedVersion = 2;
    bytes32 expectedHash1 = 0xb986975c64018680bbeeb010310922cb3cd7dde2dfdcb318dadf60a3b3278837;
    bytes32 expectedHash2 = 0x66ecba817383d631066944c6f9f69b05db03dde58102eec081c930f5d87b4616;

    // Confirm that the VM2 can be parsed correctly
    Structs.VM2 memory parsedVM2 = parseVM2(invalidVM2);

    // Sanity check a few values
    assertEq(parsedVM2.header.version, expectedVersion);
    assertEq(parsedVM2.header.hashes[0], expectedHash1);
    assertEq(parsedVM2.header.hashes[1], expectedHash2);

    // Make sure the parseAndVerifyVM2 fails
    ( , bool valid, ) = this.parseAndVerifyVM2(invalidVM2);
    assertEq(valid, false);

    // Confirm that hashes in the batch were not cached
    uint256 hashLength = parsedVM2.header.hashes.length;
    for (uint256 i = 0; i < hashLength; i++) {
      assertEq(verifiedHashCached(parsedVM2.header.hashes[i]), false);
    }
  }

  // This test confirms that parseObservation deserializes observations correctly
  function testParseObservation(
    uint32 timestamp,
    uint32 nonce,
    uint16 emitterChainId,
		bytes32 emitterAddress,
		uint64 sequence,
		uint8 consistencyLevel,
		bytes memory payload
  ) public {
    // Make sure that the observation length is < uint32
    vm.assume(payload.length < 4294967295);

    // Encode the observation
    bytes memory observation = abi.encodePacked(
      timestamp,
      nonce,
      emitterChainId,
      emitterAddress,
      sequence,
      consistencyLevel,
      payload
    );

    // Parse the observation
    Structs.Observation memory parsedObservation = parseObservation(0, observation.length, observation);

    // Confirm that everything was parsed correctly
    assertEq(parsedObservation.timestamp, timestamp);
    assertEq(parsedObservation.nonce, nonce);
    assertEq(parsedObservation.emitterChainId, emitterChainId);
    assertEq(parsedObservation.emitterAddress, emitterAddress);
    assertEq(parsedObservation.sequence, sequence);
    assertEq(parsedObservation.consistencyLevel, consistencyLevel);
    assertEq(parsedObservation.payload, payload);
  }

  // This test confirms that parseVM2 deserializes encodedVM2s correctly
  function testParseVM2(
    uint32 guardianSetIndex,
    uint8 numObservations,
    uint8 numSignatures,
    bytes32 message,
    Structs.Observation memory testObservation
  ) public {
    vm.assume(numSignatures <= 19);
    vm.assume(testObservation.timestamp > 0);
    vm.assume(testObservation.nonce > 0);
    vm.assume(testObservation.sequence > 0);
    vm.assume(testObservation.payload.length < 4294967295);

    // Create arbitrary signatures
    Structs.Signature[] memory signatures = new Structs.Signature[](numSignatures);
    bytes memory packedSignatures;

    for (uint8 i = 0; i < numSignatures; i++) {
      Structs.Signature memory arbitrarySig = Structs.Signature(message, message, i, i);
      signatures[i] = arbitrarySig;

      // Create packed version of the signatures for VM creation
      // in the order that it's parsed in the contract
      packedSignatures = abi.encodePacked(
        packedSignatures,
        abi.encodePacked(i, message, message, i)
      );
    }

    // Create arbitrary observations and hash them
    bytes32[] memory observationHashes = new bytes32[](numObservations);
    bytes[] memory observations = new bytes[](numObservations);
    bytes memory packedObservations;

    for (uint8 i = 0; i < numObservations; i++) {
      bytes memory observation = abi.encodePacked(
        testObservation.timestamp / (i + 1),
        testObservation.nonce / (i + 1),
        testObservation.emitterChainId,
        testObservation.emitterAddress,
        testObservation.sequence / (i + 1),
        testObservation.consistencyLevel,
        testObservation.payload
      );
      observations[i] = observation;
      observationHashes[i] = doubleKeccak256(observation);

      // Create packed version of the observations for VM creation
      packedObservations = abi.encodePacked(packedObservations, uint32(observation.length), observation);
    }

    // Create the arbitrary VM2
    bytes memory VM2 = abi.encodePacked(
      uint8(2), // VM version
      guardianSetIndex,
      numSignatures,
      packedSignatures,
      numObservations,
      observationHashes,
      packedObservations
    );

    // Parse the VM2
    Structs.VM2 memory vm2 = parseVM2(VM2);

    // Validate the parsed output
    assertEq(vm2.header.version, 2);
    assertEq(vm2.header.guardianSetIndex, guardianSetIndex);

    // Validate signatures
    for (uint8 i = 0; i < numSignatures; i++) {
      assertEq(vm2.header.signatures[i].r, signatures[i].r);
      assertEq(vm2.header.signatures[i].s, signatures[i].s);
      assertEq(vm2.header.signatures[i].v, signatures[i].v + 27);
      assertEq(vm2.header.signatures[i].guardianIndex, signatures[i].guardianIndex);
    }

    // Validate hashes and observations
    for (uint8 i = 0; i < numObservations; i++) {
      // Hashes
      assertEq(vm2.header.hashes[i], observationHashes[i]);
      // Observations
      assertEq(vm2.observations[i], abi.encodePacked(uint8(3), observations[i]));
    }

    // Compute the batch hash and compare it to the parsed batch hash
    bytes32 batchHash = doubleKeccak256(abi.encodePacked(observationHashes));
    assertEq(vm2.header.hash, batchHash);
  }
}