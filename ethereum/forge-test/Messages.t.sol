// test/Messages.sol
// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.0;

import "../contracts/Messages.sol";
import "../contracts/Structs.sol";
import "forge-std/Test.sol";
import "../contracts/libraries/external/BytesLib.sol";

contract TestMessages is Messages, Test {
  using BytesLib for bytes;
  address constant testGuardianPub = 0xbeFA429d57cD18b7F8A4d91A2da9AB4AF05d0FBe;

  // A valid VM2 with three observations and one signature from the testGuardianPublic key
  bytes validVM2 = hex"020000000001005201ab02c31301c4d3a2e27d5acb85272089eefb083cf9873aeff3a9cf54a15461d062b4de222a1aaa6655318b61f3ea5fabba9889afbc6956034174ca0f0a650103b986975c64018680bbeeb010310922cb3cd7dde2dfdcb318dadf60a3b327883766ecba817383d631066944c6f9f69b05db03dde58102eec081c930f5d87b461644ef21cc76eecb6a7670bcc6ceaa2918e60226c0058a8035e38992fb6b57c223030000000035000003e800000001000b0000000000000000000000000000000000000000000000000000000000000eee00000000000005390faaaa0100000036000003e900000001000b0000000000000000000000000000000000000000000000000000000000000eee000000000000053a0fbbbbbb0200000038000003ea00000001000b0000000000000000000000000000000000000000000000000000000000000eee000000000000053b0fcccccccccc";

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

  // This test ensures that individual hashes are not cached when a bad encoded VM2
  // is passed to parseAndVerifyBatchVM. The encoded VM2 is not valid in this case, since
  // a valid guardian set has not been stored yet.
  function testParseAndVerifyBatchVMFailure() public {
    uint8 expectedVersion = 2;
    bytes32 expectedHash1 = 0xb986975c64018680bbeeb010310922cb3cd7dde2dfdcb318dadf60a3b3278837;
    bytes32 expectedHash2 = 0x66ecba817383d631066944c6f9f69b05db03dde58102eec081c930f5d87b4616;

    // Confirm that a valid guaridan set has not been stored yet
    assertEq(getCurrentGuardianSetIndex(), 0);
    assertEq(getGuardianSet(getCurrentGuardianSetIndex()).keys.length, 0);

    // Confirm that the VM2 can be parsed correctly
    Structs.VM2 memory parsedVM2 = parseBatchVM(validVM2);

    // Sanity check a few values
    assertEq(parsedVM2.version, expectedVersion);
    assertEq(parsedVM2.hashes[0], expectedHash1);
    assertEq(parsedVM2.hashes[1], expectedHash2);

    // Make sure the parseAndVerifyBatchVM fails
    ( , bool valid, string memory reason) = this.parseAndVerifyBatchVM(validVM2, true);
    assertEq(valid, false);
    assertEq(reason, "invalid guardian set");

    // Confirm that hashes in the batch were not cached
    uint256 hashLength = parsedVM2.hashes.length;
    for (uint256 i = 0; i < hashLength; i++) {
      assertEq(verifiedHashCached(parsedVM2.hashes[i]), false);
    }
  }

  // This test confirms that verifyBatchVM verifies each IndexedObservation's hash correctly.
  function testInvalidObservationIndex() public {
    // Set the initial guardian set
    address[] memory initialGuardians = new address[](1);
    initialGuardians[0] = testGuardianPub;

    // Create a guardian set
    Structs.GuardianSet memory initialGuardianSet = Structs.GuardianSet({
        keys : initialGuardians,
        expirationTime : 0
    });

    storeGuardianSet(initialGuardianSet, 0);

    // Confirm that the test VM2 is valid
    (, bool valid, string memory reason) = this.parseAndVerifyBatchVM(validVM2, false);
    require(valid, reason);

    // Calculate the index of the first observation
    uint256 observationsIndex = 0;
    observationsIndex += 1; // version
    observationsIndex += 4; // guardian set index
    observationsIndex += 1; // number of signatures
    observationsIndex += 66 * 1; // signature * number of signatures
    observationsIndex += 1; // number of hashes
    observationsIndex += 32 * 3; // hashes
    observationsIndex += 1; // number of observations

    // Change the index of the first observation to a number within
    // the expected range of indices.
    uint8 newIndex = 2;
    bytes memory invalidIndexVm2 = abi.encodePacked(
      validVM2.slice(0, observationsIndex),
      newIndex,
      validVM2.slice(observationsIndex + 1, validVM2.length - observationsIndex - 1)
    );

    // Parse the invalidIndexVm2 to confirm the index was updated
    Structs.VM2 memory parsedInvalidIndexvm2 = parseBatchVM(invalidIndexVm2);
    assertEq(parsedInvalidIndexvm2.indexedObservations[0].index, newIndex);

    // Try to parse and verify the invalidIndexVm2
    (, bool valid2, string memory reason2) = this.parseAndVerifyBatchVM(invalidIndexVm2, false);
    assertEq(valid2, false);
    assertEq(reason2, "invalid observation");
  }

  // This test confirms that verifyBatchVM checks that each observation's index is within the
  // bounds of the array of observation hashes.
  function testOutOfBoundsObservationIndex() public {
    // Set the initial guardian set
    address[] memory initialGuardians = new address[](1);
    initialGuardians[0] = testGuardianPub;

    // Create a guardian set
    Structs.GuardianSet memory initialGuardianSet = Structs.GuardianSet({
        keys : initialGuardians,
        expirationTime : 0
    });

    storeGuardianSet(initialGuardianSet, 0);

    // Confirm that the test VM2 is valid
    (Structs.VM2 memory parsedValidVm2, bool valid, string memory reason) = this.parseAndVerifyBatchVM(validVM2, false);
    require(valid, reason);

    // Calculate the index of the first observation
    uint256 observationsIndex = 0;
    observationsIndex += 1; // version
    observationsIndex += 4; // guardian set index
    observationsIndex += 1; // number of signatures
    observationsIndex += 66 * 1; // signature * number of signatures
    observationsIndex += 1; // number of hashes
    observationsIndex += 32 * 3; // hashes
    observationsIndex += 1; // number of observations

    // Change the index of the first observation to a number not within
    // the bounds of the observation hashes array.
    uint8 newIndex = uint8(parsedValidVm2.hashes.length + 1);
    bytes memory outOfBoundsIndexVm2 = abi.encodePacked(
      validVM2.slice(0, observationsIndex),
      newIndex,
      validVM2.slice(observationsIndex + 1, validVM2.length - observationsIndex - 1)
    );

    // Parse the invalidIndexVm2 to confirm the index was updated
    Structs.VM2 memory parsedOutOfBoundsIndexVm2 = parseBatchVM(outOfBoundsIndexVm2);
    assertEq(parsedOutOfBoundsIndexVm2.indexedObservations[0].index, newIndex);

    // Try to parse and verify the outOfBoundsIndexVm2
    vm.expectRevert("observation index out of bounds");
    this.parseAndVerifyBatchVM(outOfBoundsIndexVm2, false);
  }

  // This test confirms that parseBatchVM reverts when parsing batches with more
  // observations than hashes in the batch hash array.
  function testMoreObservationsThanHashesInABatch() public {
    // Set the initial guardian set
    address[] memory initialGuardians = new address[](1);
    initialGuardians[0] = testGuardianPub;

    // Create a guardian set
    Structs.GuardianSet memory initialGuardianSet = Structs.GuardianSet({
        keys : initialGuardians,
        expirationTime : 0
    });

    storeGuardianSet(initialGuardianSet, 0);

    // Confirm that the test VM2 is valid
    (Structs.VM2 memory parsedValidVm2, bool valid, string memory reason) = this.parseAndVerifyBatchVM(validVM2, false);
    require(valid, reason);

    // Calculate the index of the observations count byte
    uint256 observationsCountIndex = 0;
    observationsCountIndex += 1; // version
    observationsCountIndex += 4; // guardian set index
    observationsCountIndex += 1; // number of signatures
    observationsCountIndex += 66 * 1; // signature * number of signatures
    observationsCountIndex += 1; // number of hashes
    observationsCountIndex += 32 * 3; // hashes

    // Change the observations count byte to a number larger than
    // the hashes array length.
    uint8 newObservationsCount = uint8(parsedValidVm2.indexedObservations.length + 1);
    bytes memory invalidObservationsCountVm2 = abi.encodePacked(
      validVM2.slice(0, observationsCountIndex),
      newObservationsCount,
      validVM2.slice(observationsCountIndex + 1, validVM2.length - observationsCountIndex - 1)
    );

    // Parsing the invalidObsevationsCountVm2 should fail
    vm.expectRevert("invalid number of observations");
    parseBatchVM(invalidObservationsCountVm2);
  }

  // This test confirms that parseBatchVM reverts when parsing partial batches with an
  // observations count less than the actual number of observations in a batch.
  function testInvalidObservationCount() public {
    // Set the initial guardian set
    address[] memory initialGuardians = new address[](1);
    initialGuardians[0] = testGuardianPub;

    // Create a guardian set
    Structs.GuardianSet memory initialGuardianSet = Structs.GuardianSet({
        keys : initialGuardians,
        expirationTime : 0
    });

    storeGuardianSet(initialGuardianSet, 0);

    // Confirm that the test VM2 is valid
    (Structs.VM2 memory parsedValidVm2, bool valid, string memory reason) = this.parseAndVerifyBatchVM(validVM2, false);
    require(valid, reason);

    // Calculate the index of the observations count byte
    uint256 observationsCountIndex = 0;
    observationsCountIndex += 1; // version
    observationsCountIndex += 4; // guardian set index
    observationsCountIndex += 1; // number of signatures
    observationsCountIndex += 66 * 1; // signature * number of signatures
    observationsCountIndex += 1; // number of hashes
    observationsCountIndex += 32 * 3; // hashes

    // Change the observations count byte to a number less than
    // the actual number of encoded observations in the VM.
    uint8 newObservationsCount = uint8(parsedValidVm2.indexedObservations.length - 1);
    bytes memory invalidObservationsCountVm2 = abi.encodePacked(
      validVM2.slice(0, observationsCountIndex),
      newObservationsCount,
      validVM2.slice(observationsCountIndex + 1, validVM2.length - observationsCountIndex - 1)
    );

    // Parsing the invalidObsevationsCountVm2 should fail
    vm.expectRevert("invalid VM2");
    parseBatchVM(invalidObservationsCountVm2);
  }

  // This test confirms that verifyBatchVM reverts when observation indices are not ascending.
  function testAscendingObservationIndices() public {
    // Set the initial guardian set
    address[] memory initialGuardians = new address[](1);
    initialGuardians[0] = testGuardianPub;

    // Create a guardian set
    Structs.GuardianSet memory initialGuardianSet = Structs.GuardianSet({
        keys : initialGuardians,
        expirationTime : 0
    });

    storeGuardianSet(initialGuardianSet, 0);

    // Confirm that the test vm is valid
    (Structs.VM2 memory originalVm2, bool valid, string memory reason) = this.parseAndVerifyBatchVM(validVM2, false);
    require(valid, reason);

    // Parse the VM2 up to the first observation
    uint256 index = 0;
    index += 1; // version
    index += 4; // guardian set index
    index += 1; // number of signatures
    index += 66 * 1; // signature * number of signatures
    index += 1; // number of hashes
    index += 32 * 3; // hashes
    index += 1; // number of observations

    // Calculate the start and end index of the first observation
    uint256 observationOneStartIndex = index;
    uint256 observationOneEndIndex = observationOneStartIndex + 5 + validVM2.toUint32(observationOneStartIndex + 1);

    // Calculate the start and end index of the second observation
    uint256 observationTwoStartIndex = observationOneEndIndex;
    uint256 observationTwoEndIndex = observationTwoStartIndex + 5 + validVM2.toUint32(observationTwoStartIndex + 1);

    // Change the order of the observations in the VM2
    bytes memory modifiedVm2 = abi.encodePacked(
      validVM2.slice(0, observationOneStartIndex),
      validVM2.slice(observationTwoStartIndex, observationTwoEndIndex - observationTwoStartIndex),
      validVM2.slice(observationOneStartIndex, observationOneEndIndex - observationOneStartIndex),
      validVM2.slice(observationTwoEndIndex, validVM2.length - observationTwoEndIndex)
    );

    // Parse the modifiedVm2 and validate the observation swap
    Structs.VM2 memory parsedModifiedVm2 = parseBatchVM(modifiedVm2);
    assertEq(originalVm2.indexedObservations[0].index, parsedModifiedVm2.indexedObservations[1].index);
    assertEq(originalVm2.indexedObservations[0].observation, parsedModifiedVm2.indexedObservations[1].observation);
    assertEq(originalVm2.indexedObservations[1].index, parsedModifiedVm2.indexedObservations[0].index);
    assertEq(originalVm2.indexedObservations[1].observation, parsedModifiedVm2.indexedObservations[0].observation);
    assertEq(originalVm2.indexedObservations[2].index, parsedModifiedVm2.indexedObservations[2].index);
    assertEq(originalVm2.indexedObservations[2].observation, parsedModifiedVm2.indexedObservations[2].observation);

    for (uint256 i = 0; i < parsedModifiedVm2.hashes.length; i++) {
      assertEq(originalVm2.hashes[i], parsedModifiedVm2.hashes[i]);
    }

    // Verifying the parsedModifiedVm2 should fail
    vm.expectRevert("observation indices must be ascending");
    this.verifyBatchVM(parsedModifiedVm2, false);
  }

  // This test confirms that parseObservation deserializes observations correctly.
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

  // This test confirms that parseBatchVM deserializes encoded batches correctly.
  function testParseBatchVM(
    uint32 guardianSetIndex,
    uint8 numObservations,
    uint8 numSignatures,
    bytes32 message,
    Structs.Observation memory testObservation
  ) public {
    vm.assume(numSignatures > 0 && numSignatures <= 19);
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

      // Create packed version of the signatures for VM2 creation
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
        testObservation.timestamp,
        testObservation.nonce,
        testObservation.emitterChainId,
        testObservation.emitterAddress,
        testObservation.sequence,
        testObservation.consistencyLevel,
        testObservation.payload
      );
      observations[i] = observation;
      observationHashes[i] = doubleKeccak256(observation);

      // Create packed version of the observations for VM2 creation
      packedObservations = abi.encodePacked(packedObservations, uint8(i), uint32(observation.length), observation);
    }

    // Create the arbitrary VM2
    bytes memory VM2 = abi.encodePacked(
      uint8(2), // VM version
      guardianSetIndex,
      numSignatures,
      packedSignatures,
      numObservations,
      observationHashes,
      numObservations, // full batch
      packedObservations
    );

    // Make sure there numObservations is greater than 0
    if (numObservations == 0) {
      vm.expectRevert("invalid number of observations");
    }

    // Parse the VM2
    Structs.VM2 memory vm2 = parseBatchVM(VM2);

    // Validate the parsed output
    assertEq(vm2.version, 2);
    assertEq(vm2.guardianSetIndex, guardianSetIndex);

    // Validate signatures
    for (uint8 i = 0; i < numSignatures; i++) {
      assertEq(vm2.signatures[i].r, signatures[i].r);
      assertEq(vm2.signatures[i].s, signatures[i].s);
      assertEq(vm2.signatures[i].v, signatures[i].v + 27);
      assertEq(vm2.signatures[i].guardianIndex, signatures[i].guardianIndex);
    }

    // Validate hashes and observations
    for (uint8 i = 0; i < numObservations; i++) {
      // Observations index
      assertEq(vm2.indexedObservations[i].index, i);
      // Observations
      assertEq(vm2.indexedObservations[i].observation, abi.encodePacked(uint8(3), observations[i]));
      // Hash
      assertEq(vm2.hashes[i], observationHashes[i]);
    }

    // Compute the batch hash and compare it to the parsed batch hash
    bytes32 batchHash = doubleKeccak256(abi.encodePacked(observationHashes));
    assertEq(vm2.hash, batchHash);
  }

  // This test confirms that parseVM deserializes encoded VM3s (Headless VMs) correctly
  function testParseHeadlessVM(
    uint32 guardianSetIndex,
    address[] memory guardianKeys,
    Structs.Observation memory testObservation
  ) public {
    vm.assume(guardianKeys.length <= 19 && guardianKeys.length > 0);

    // Create a guardian set
    Structs.GuardianSet memory initialGuardianSet = Structs.GuardianSet({
        keys : guardianKeys,
        expirationTime : 0
    });
    storeGuardianSet(initialGuardianSet, guardianSetIndex);
    updateGuardianSetIndex(guardianSetIndex);

    // Confirm the guardian set is set correctly
    assertEq(getCurrentGuardianSetIndex(), guardianSetIndex);
    assertEq(getGuardianSet(guardianSetIndex).keys, guardianKeys);

    // Create a headless VAA by prepending the version type to the observation
    bytes memory headlessVM = abi.encodePacked(
      uint8(3),
      testObservation.timestamp,
      testObservation.nonce,
      testObservation.emitterChainId,
      testObservation.emitterAddress,
      testObservation.sequence,
      testObservation.consistencyLevel,
      testObservation.payload
    );

    // Parse the headless VAA
    Structs.VM memory vm = parseVM(headlessVM);

    // Validate the parsed values
    assertEq(vm.version, 3);
    assertEq(vm.timestamp, testObservation.timestamp);
    assertEq(vm.nonce, testObservation.nonce);
    assertEq(vm.emitterChainId, testObservation.emitterChainId);
    assertEq(vm.emitterAddress, testObservation.emitterAddress);
    assertEq(vm.sequence, testObservation.sequence);
    assertEq(vm.consistencyLevel, testObservation.consistencyLevel);
    assertEq(vm.payload, testObservation.payload);
    assertEq(vm.hash, doubleKeccak256(headlessVM.slice(1, headlessVM.length - 1)));

    // Confirm that guardianSetIndex is zero and there are no signatures
    assertEq(vm.guardianSetIndex, 0);
    assertEq(vm.signatures.length, 0);
  }
}