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
}
