pragma solidity ^0.8.0;

import "truffle/Assert.sol";
import "truffle/DeployedAddresses.sol";
import "../contracts/Messages.sol";

contract TestMessages {
  function testQuorum() {
    // TODO: Setup cases and loop through them with assert expectations
    // setup test signatures
    // setup test guardianSet

    Assert.equal(quorum(signatures, guardianSet), true, "We should have quorum");
    Assert.equal(quorum(signatures, guardianSet), true, "We should not have quorum");
  }
}